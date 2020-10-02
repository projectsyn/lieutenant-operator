package gitrepo

import (
	"context"
	"fmt"

	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the GitRepo instance
		instance := &synv1alpha1.GitRepo{}

		err := r.client.Get(context.TODO(), request.NamespacedName, instance)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		instanceCopy := instance.DeepCopy()

		if instance.Spec.RepoType == synv1alpha1.UnmanagedRepoType {
			reqLogger.Info("Skipping GitRepo because it is unmanaged")
			return nil
		}

		repo, hostKeys, err := manager.GetGitClient(&instance.Spec.GitRepoTemplate, instance.GetNamespace(), reqLogger, r.client)
		if err != nil {
			return err
		}

		instance.Status.HostKeys = hostKeys

		if !r.repoExists(repo) {
			reqLogger.Info("creating git repo", manager.SecretEndpointName, repo.FullURL())
			err := repo.Create()
			if err != nil {
				return r.handleRepoError(err, instance, repo)
			}
			reqLogger.Info("successfully created the repository")
		}

		deleted := helpers.HandleDeletion(instance, finalizerName, r.client)
		if deleted.FinalizerRemoved {
			err := repo.Remove()
			if err != nil {
				return err
			}
		}
		if deleted.Deleted {
			return r.client.Update(context.TODO(), instance)
		}

		err = repo.CommitTemplateFiles()
		if err != nil {
			return r.handleRepoError(err, instance, repo)
		}

		changed, err := repo.Update()
		if err != nil {
			return err
		}

		if changed {
			reqLogger.Info("keys differed from CRD, keys re-applied to repository")
		}

		helpers.AddTenantLabel(&instance.ObjectMeta, instance.Spec.TenantRef.Name)
		helpers.AddDeletionProtection(instance)

		controllerutil.AddFinalizer(instance, finalizerName)

		if !equality.Semantic.DeepEqual(instanceCopy, instance) {
			err = r.client.Update(context.TODO(), instance)
		}
		if err != nil {
			return err
		}
		phase := synv1alpha1.Created
		instance.Status.Phase = &phase
		instance.Status.URL = repo.FullURL().String()
		instance.Status.Type = synv1alpha1.GitType(repo.Type())
		if !equality.Semantic.DeepEqual(instanceCopy.Status, instance.Status) {
			if err := r.client.Status().Update(context.TODO(), instance); err != nil {
				return err
			}
		}
		return nil
	})

	return reconcile.Result{}, err
}

func (r *ReconcileGitRepo) repoExists(repo manager.Repo) bool {
	if err := repo.Read(); err == nil {
		return true
	}

	return false
}

func (r *ReconcileGitRepo) handleRepoError(repoErr error, instance *synv1alpha1.GitRepo, repo manager.Repo) error {
	instanceCopy := instance.DeepCopy()
	phase := synv1alpha1.Failed
	instance.Status.Phase = &phase
	instance.Status.URL = repo.FullURL().String()
	if !equality.Semantic.DeepEqual(instanceCopy.Status, instance.Status) {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return fmt.Errorf("could not set status while handling error: %w: %s", err, repoErr)
		}
	}
	return repoErr
}
