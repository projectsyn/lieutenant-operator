package tenant

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// CommonClassName is the name of the tenant's common class
	CommonClassName = "common"
)

// Reconcile The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTenant) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Tenant")

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// Fetch the Tenant instance
		instance := &synv1alpha1.Tenant{}
		err := r.client.Get(context.TODO(), request.NamespacedName, instance)
		if err != nil {
			if errors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				return nil
			}
			// Error reading the object - requeue the request.
			return err
		}
		instanceCopy := instance.DeepCopy()

		if len(instance.Spec.GitRepoTemplate.DisplayName) == 0 {
			instance.Spec.GitRepoTemplate.DisplayName = instance.Spec.DisplayName
		}

		commonClassFile := CommonClassName + ".yml"
		if instance.Spec.GitRepoTemplate.TemplateFiles == nil {
			instance.Spec.GitRepoTemplate.TemplateFiles = map[string]string{}
		}
		if _, ok := instance.Spec.GitRepoTemplate.TemplateFiles[commonClassFile]; !ok {
			instance.Spec.GitRepoTemplate.TemplateFiles[commonClassFile] = ""
		}

		instance.Spec.GitRepoTemplate.DeletionPolicy = instance.Spec.DeletionPolicy

		result, err := helpers.CreateOrUpdateGitRepo(instance, r.scheme, instance.Spec.GitRepoTemplate, r.client, corev1.LocalObjectReference{Name: instance.GetName()})
		if err != nil {
			return err
		}

		if result != controllerutil.OperationResultCreated {
			instance.Spec.GitRepoURL, _, err = helpers.GetGitRepoURLAndHostKeys(instance, r.client)
			if err != nil {
				return err
			}
		}

		helpers.AddDeletionProtection(instance)
		if !equality.Semantic.DeepEqual(instanceCopy, instance) {
			if err := r.client.Update(context.TODO(), instance); err != nil {
				return err
			}
		}
		return nil
	})
	return reconcile.Result{}, err
}
