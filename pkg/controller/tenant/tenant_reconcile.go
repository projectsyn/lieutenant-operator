package tenant

import (
	"context"
	"os"

	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"k8s.io/apimachinery/pkg/api/errors"
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
				return nil
			}
			return err
		}
		instanceCopy := instance.DeepCopy()

		data := &pipeline.ExecutionContext{
			Client:        r.client,
			Log:           reqLogger,
			FinalizerName: "",
		}

		err = pipeline.ReconcileTenant(instance, data)
		if err != nil {
			return err
		}

		return nil
	})
	return reconcile.Result{}, err
}
