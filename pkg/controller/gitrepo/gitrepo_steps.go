package gitrepo

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func gitRepoSpecificSteps(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	instance, ok := obj.(*synv1alpha1.GitRepo)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a GitRepository")}
	}

	err := fetchGitRepoTemplate(instance, data)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	if instance.Spec.RepoType == synv1alpha1.UnmanagedRepoType {
		data.Log.Info("Skipping GitRepo because it is unmanaged")
		return pipeline.Result{}
	}

	repo, hostKeys, err := manager.GetGitClient(&instance.Spec.GitRepoTemplate, instance.GetNamespace(), data.Log, data.Client)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	instance.Status.HostKeys = hostKeys

	if !repoExists(repo) {
		data.Log.Info("creating git repo", manager.SecretEndpointName, repo.FullURL())
		err := repo.Create()
		if err != nil {
			return pipeline.Result{Err: handleRepoError(err, instance, repo, data.Client)}

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
		return pipeline.Result{Err: handleRepoError(err, instance, repo, data.Client)}
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

// CreateGitRepo will create the gitRepo object if it doesn't already exist.
func CreateGitRepo(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	template := obj.GetGitTemplate()

	if template == nil {
		return pipeline.Result{}
	}

	if template.DisplayName == "" {
		template.DisplayName = obj.GetDisplayName()
	}

	if obj.GetTenantRef().Name == "" {
		return pipeline.Result{
			Abort: true,
			Err:   fmt.Errorf("the tenant name is empty"),
		}
	}

	if template.DeletionPolicy == "" {
		if obj.GetDeletionPolicy() == "" {
			template.DeletionPolicy = pipeline.GetDefaultDeletionPolicy()
		} else {
			template.DeletionPolicy = obj.GetDeletionPolicy()
		}
	}

	if template.RepoType == synv1alpha1.DefaultRepoType {
		template.RepoType = synv1alpha1.AutoRepoType
	}

	repo := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetObjectMeta().GetName(),
			Namespace: obj.GetObjectMeta().GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj.GetObjectMeta(), obj.GroupVersionKind()),
			},
		},
		Spec: synv1alpha1.GitRepoSpec{
			GitRepoTemplate: *template,
			TenantRef:       obj.GetTenantRef(),
		},
	}

	err := data.Client.Create(context.TODO(), repo)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return pipeline.Result{}
		}
	}

	return pipeline.Result{Err: err}

}

func SetGitRepoURLAndHostKeys(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	gitRepo := &synv1alpha1.GitRepo{}
	repoNamespacedName := types.NamespacedName{
		Namespace: obj.GetObjectMeta().GetNamespace(),
		Name:      obj.GetObjectMeta().GetName(),
	}
	err := data.Client.Get(context.TODO(), repoNamespacedName, gitRepo)
	if err != nil {
		if errors.IsNotFound(err) {
			return pipeline.Result{}
		}
		return pipeline.Result{Abort: true, Err: err}
	}

	if gitRepo.Spec.RepoType != synv1alpha1.UnmanagedRepoType {
		obj.SetGitRepoURLAndHostKeys(gitRepo.Status.URL, gitRepo.Status.HostKeys)
	}
	return pipeline.Result{}
}

func repoExists(repo manager.Repo) bool {
	if err := repo.Read(); err == nil {
		return true
	}

	return false
}

func handleRepoError(err error, instance *synv1alpha1.GitRepo, repo manager.Repo, client client.Client) error {
	phase := synv1alpha1.Failed
	instance.Status.Phase = &phase
	if updateErr := client.Status().Update(context.TODO(), instance); updateErr != nil {
		return fmt.Errorf("could not set status while handling error: %s: %s", updateErr, err)
	}
	return err
}

func fetchGitRepoTemplate(obj *synv1alpha1.GitRepo, data *pipeline.Context) error {
	tenant := &synv1alpha1.Tenant{}

	tenantName := types.NamespacedName{Name: obj.GetObjectMeta().GetName(), Namespace: obj.GetObjectMeta().GetNamespace()}

	err := data.Client.Get(context.TODO(), tenantName, tenant)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if tenant != nil && tenant.Spec.GitRepoTemplate != nil {
		obj.Spec.GitRepoTemplate = *tenant.Spec.GitRepoTemplate
	}

	cluster := &synv1alpha1.Cluster{}

	clusterName := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}

	err = data.Client.Get(context.TODO(), clusterName, cluster)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if cluster != nil && cluster.Spec.GitRepoTemplate != nil {
		obj.Spec.GitRepoTemplate = *cluster.Spec.GitRepoTemplate
	}

	return nil
}
