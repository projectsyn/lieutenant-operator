package tenant

import (
	"context"

	"github.com/projectsyn/lieutenant-operator/pkg/controller/gitrepo"
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

	// Fetch the Tenant instance
	instance := &synv1alpha1.Tenant{}
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
		FinalizerName: "",
	}

	steps := []pipeline.Step{
		{Name: "copy original object", F: pipeline.DeepCopyOriginal},
		{Name: "tenant specific steps", F: tenantSpecificSteps},
		{Name: "create git repo", F: gitrepo.CreateGitRepo},
		{Name: "set gitrepo url and hostkeys", F: gitrepo.SetGitRepoURLAndHostKeys},
		{Name: "common", F: pipeline.Common},
	}
	res := pipeline.RunPipeline(instance, data, steps)

	return reconcile.Result{Requeue: res.Requeue}, res.Err
}

func tenantSpecificSteps(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	steps := []pipeline.Step{
		{Name: "apply template from TenantTemplate", F: applyTemplateFromTenantTemplate},
		{Name: "add default class file", F: addDefaultClassFile},
		{Name: "uptade tenant git repo", F: updateTenantGitRepo},
		{Name: "set global git repo url", F: setGlobalGitRepoURL},
	}

	return pipeline.RunPipeline(obj, data, steps)
}
