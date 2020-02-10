package gitrepo

import (
	"context"
	"fmt"

	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	SecretTokenName = "token"
)

// Reconcile will create or delete a git repository based on the event.
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
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

	secret := &corev1.Secret{}

	namespacedName := types.NamespacedName{
		Name:      instance.Spec.APISecretRef.Name,
		Namespace: instance.Spec.APISecretRef.Namespace,
	}

	err = r.client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error getting git secret: %v", err)
	}

	repoOptions := manager.RepoOptions{
		Credentials: manager.Credentials{
			Token: string(secret.Data[SecretTokenName]),
		},
		DeployKeys: instance.Spec.DeployKeys,
		Logger:     reqLogger,
	}

	endpoint := string(secret.Data["endpoint"]) + "/" + instance.Spec.Path + "/" + instance.Spec.RepoName

	repo, err := manager.NewRepo(endpoint, repoOptions)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = repo.Connect()
	if err != nil {
		return reconcile.Result{}, err
	}

	if !r.repoExists(repo) {
		reqLogger.Info("creating git repo", "endpoint", endpoint)

		err = repo.Create()
		if err != nil {
			err1 := r.updateStatus(instance, synv1alpha1.Failed, "git repo creation failed", "failure", repo)
			if err1 != nil {
				err = fmt.Errorf("could not set status while handling error: %s: %s", err1, err)
			}
			return reconcile.Result{}, err
		}

		reqLogger.Info("successfully created the repository")

		return reconcile.Result{}, r.updateStatus(instance, synv1alpha1.Created, "Git repo is ready to be used", "ready", repo)

	} else {
		changed, err := repo.Update()
		if err != nil {
			return reconcile.Result{}, err
		}

		if changed {
			reqLogger.Info("keys differed from CRD, keys re-applied to repository")
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileGitRepo) updateStatus(gitRepo *synv1alpha1.GitRepo, phaseToSet synv1alpha1.GitPhase, reason, conditionType string, repoImpl manager.Repo) error {

	gitRepo.Status.Phase = &phaseToSet

	if phaseToSet == synv1alpha1.Created {
		gitRepo.Status.URL = repoImpl.FullURL().String()
	}

	return r.client.Status().Update(context.TODO(), gitRepo)
}

func (r *ReconcileGitRepo) repoExists(repo manager.Repo) bool {
	if err := repo.Read(); err == nil {
		return true
	}

	return false
}
