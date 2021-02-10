package gitrepo

import (
	"context"

	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	finalizerName = "gitrepo.lieutenant.syn.tools"
)

// Reconcile will create or delete a git repository based on the event.
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
// ATTENTION: do not manipulate the spec here, this will lead to loops, as the specs are
// defined in the gitrepotemplates of the other CRDs (tenant, cluster).
func (r *ReconcileGitRepo) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GitRepo")

	// Fetch the GitRepo instance
	instance := &synv1alpha1.GitRepo{}

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
	}

	steps := []pipeline.Step{
		{Name: "copy original object", F: pipeline.DeepCopyOriginal},
		{Name: "deletion check", F: pipeline.CheckIfDeleted},
		{Name: "git repo specific steps", F: gitRepoSpecificSteps},
		{Name: "add tenant label", F: pipeline.AddTenantLabel},
		{Name: "Common", F: pipeline.Common},
	}

	res := pipeline.RunPipeline(instance, data, steps)

	return reconcile.Result{Requeue: res.Requeue}, res.Err

}
