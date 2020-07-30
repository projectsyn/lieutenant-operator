package cluster

import (
	"context"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusterClassContent = `classes:
- %s.%s
`
	finalizerName = "cluster.lieutenant.syn.tools"
)

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Cluster")

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		instance := &synv1alpha1.Cluster{}

		err := r.client.Get(context.TODO(), request.NamespacedName, instance)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		instanceCopy := instance.DeepCopy()

		nsName := request.NamespacedName
		nsName.Name = instance.Spec.TenantRef.Name

		tenant := &synv1alpha1.Tenant{}

		if err := r.client.Get(context.TODO(), nsName, tenant); err != nil {
			return fmt.Errorf("Couldn't find tenant: %w", err)
		}

		if err := applyClusterTemplate(instance, tenant); err != nil {
			return err
		}

		data := &pipeline.ExecutionContext{
			Client:        r.client,
			Log:           reqLogger,
			FinalizerName: finalizerName,
		}

		return pipeline.ReconcileCluster(instance, data)
	})

	return reconcile.Result{}, err
}
