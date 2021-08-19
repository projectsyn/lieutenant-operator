package gitrepo

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"

	// Register Gitrepo implementation - DONOT REMOVE
	_ "github.com/projectsyn/lieutenant-operator/git"
	"github.com/projectsyn/lieutenant-operator/git/manager"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Steps(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	instance, ok := obj.(*synv1alpha1.GitRepo)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object '%s/%s' is not of kind GitRepository", obj.GetNamespace(), obj.GetName())}
	}

	err := fetchGitRepoTemplate(instance, data)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	if instance.Spec.RepoType == synv1alpha1.UnmanagedRepoType {
		data.Log.Info("Skipping GitRepo '%s/%s' because it is unmanaged", obj.GetNamespace(), obj.GetName())
		return pipeline.Result{}
	}

	repo, hostKeys, err := manager.GetGitClient(data.Context(), &instance.Spec.GitRepoTemplate, instance.GetNamespace(), data.Log, data.Client)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	instance.Status.HostKeys = hostKeys

	if !repoExists(repo) {
		data.Log.Info("creating git repo", manager.SecretEndpointName, repo.FullURL())
		err := repo.Create()
		if err != nil {
			return pipeline.Result{Err: handleRepoError(data.Context(), err, instance, data.Client)}

		}
		data.Log.Info("successfully created the repository")
	}

	if data.Deleted {
		err := repo.Remove()
		if err != nil {
			return pipeline.Result{Err: err}
		}
		return pipeline.Result{}
	}

	err = repo.CommitTemplateFiles()
	if err != nil {
		return pipeline.Result{Err: handleRepoError(data.Context(), err, instance, data.Client)}
	}

	changed, err := repo.Update()
	if err != nil {
		return pipeline.Result{Err: err}
	}

	if changed {
		data.Log.Info("keys differed from CRD, keys re-applied to repository")
	}

	phase := synv1alpha1.Created
	instance.Status.Phase = &phase
	instance.Status.URL = repo.FullURL().String()
	instance.Status.Type = synv1alpha1.GitType(repo.Type())

	return pipeline.Result{}
}

func repoExists(repo manager.Repo) bool {
	if err := repo.Read(); err == nil {
		return true
	}

	return false
}

func handleRepoError(ctx context.Context, err error, instance *synv1alpha1.GitRepo, client client.Client) error {
	phase := synv1alpha1.Failed
	instance.Status.Phase = &phase
	if updateErr := client.Status().Update(ctx, instance); updateErr != nil {
		return fmt.Errorf("could not set status while handling error: %s: %s", updateErr, err)
	}
	return err
}
