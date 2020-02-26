package tenant

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconcile The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTenant) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Tenant")

	// Fetch the Tenant instance
	instance := &synv1alpha1.Tenant{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	gvk := schema.GroupVersionKind{
		Version: instance.APIVersion,
		Kind:    instance.Kind,
	}

	err = helpers.CreateOrUpdateGitRepo(instance, gvk, instance.Spec.GitRepoTemplate, r.client, corev1.LocalObjectReference{Name: instance.GetName()})
	if err != nil {
		return reconcile.Result{}, err
	}
	instance.Spec.GitRepoURL, err = helpers.GetGitRepoURL(instance, r.client)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, r.client.Update(context.TODO(), instance)
}
