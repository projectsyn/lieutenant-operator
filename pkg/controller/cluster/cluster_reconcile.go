package cluster

import (
	"context"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/controller/gitrepo"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"github.com/projectsyn/lieutenant-operator/pkg/vault"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	finalizerName = "cluster.lieutenant.syn.tools"
)

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Cluster")

	instance := &synv1alpha1.Cluster{}

	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	data := &pipeline.Context{
		Client:        r.client,
		Log:           reqLogger,
		FinalizerName: finalizerName,
		Reconciler:    r,
	}

	steps := []pipeline.Step{
		{Name: "copy original object", F: pipeline.DeepCopyOriginal},
		{Name: "cluster specific steps", F: clusterSpecificSteps},
		{Name: "create git repo", F: gitrepo.CreateGitRepo},
		{Name: "set gitrepo url and hostkeys", F: gitrepo.SetGitRepoURLAndHostKeys},
		{Name: "add tenant label", F: pipeline.AddTenantLabel},
		{Name: "Common", F: pipeline.Common},
	}

	res := pipeline.RunPipeline(instance, data, steps)

	return reconcile.Result{Requeue: res.Requeue}, res.Err

}

func clusterSpecificSteps(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	steps := []pipeline.Step{
		{Name: "create cluster RBAC", F: createClusterRBAC},
		{Name: "deletion check", F: pipeline.CheckIfDeleted},
		{Name: "set bootstrap token", F: setBootstrapToken},
		{Name: "create or update vault", F: vault.CreateOrUpdateVault},
		{Name: "delete vault entries", F: vault.HandleVaultDeletion},
		{Name: "set tenant owner", F: setTenantOwner},
		{Name: "apply cluster template from tenant", F: applyClusterTemplateFromTenant},
		{Name: "update Role", F: clusterUpdateRole},
	}

	return pipeline.RunPipeline(obj, data, steps)
}
