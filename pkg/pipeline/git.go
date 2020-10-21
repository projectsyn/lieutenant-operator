package pipeline

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func gitRepoSpecificSteps(obj PipelineObject, data *ExecutionContext) ExecutionResult {

	instance, ok := obj.(*synv1alpha1.GitRepo)
	if !ok {
		return ExecutionResult{Err: fmt.Errorf("object is not a GitRepository")}
	}

	err := fetchGitRepoTemplate(instance, data)
	if err != nil {
		return ExecutionResult{Err: err}
	}

	if instance.Spec.RepoType == synv1alpha1.UnmanagedRepoType {
		data.Log.Info("Skipping GitRepo because it is unmanaged")
		return ExecutionResult{}
	}

	repo, hostKeys, err := manager.GetGitClient(&instance.Spec.GitRepoTemplate, instance.GetNamespace(), data.Log, data.Client)
	if err != nil {
		return ExecutionResult{Err: err}
	}

	instance.Status.HostKeys = hostKeys

	if !repoExists(repo) {
		data.Log.Info("creating git repo", manager.SecretEndpointName, repo.FullURL())
		err := repo.Create()
		if err != nil {
			return ExecutionResult{Err: handleRepoError(err, instance, repo, data.Client)}

		}
		data.Log.Info("successfully created the repository")
	}

	if data.Deleted {
		err := repo.Remove()
		if err != nil {
			return ExecutionResult{Err: err}
		}
		return ExecutionResult{}
	}

	err = repo.CommitTemplateFiles()
	if err != nil {
		return ExecutionResult{Err: handleRepoError(err, instance, repo, data.Client)}
	}

	changed, err := repo.Update()
	if err != nil {
		return ExecutionResult{Err: err}
	}

	if changed {
		data.Log.Info("keys differed from CRD, keys re-applied to repository")
	}

	phase := synv1alpha1.Created
	instance.Status.Phase = &phase
	instance.Status.URL = repo.FullURL().String()
	instance.Status.Type = synv1alpha1.GitType(repo.Type())

	return ExecutionResult{}
}

// createGitRepo will create the gitRepo object if it doesn't already exist.
func createGitRepo(obj PipelineObject, data *ExecutionContext) ExecutionResult {

	template := obj.GetGitTemplate()

	if template == nil {
		return ExecutionResult{}
	}

	if template.DisplayName == "" {
		template.DisplayName = obj.GetDisplayName()
	}

	if template == nil {
		return ExecutionResult{
			Abort: true,
			Err:   fmt.Errorf("gitRepo template is empty"),
		}
	}

	if obj.GetTenantRef().Name == "" {
		return ExecutionResult{
			Abort: true,
			Err:   fmt.Errorf("the tenant name is empty"),
		}
	}

	if template.DeletionPolicy == "" {
		if obj.GetDeletionPolicy() == "" {
			template.DeletionPolicy = getDefaultDeletionPolicy()
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
			return ExecutionResult{}
		}
	}

	for file, content := range template.TemplateFiles {
		if content == manager.DeletionMagicString {
			delete(template.TemplateFiles, file)
		}
	}

	return ExecutionResult{Err: err}
}

func setGitRepoURLAndHostKeys(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	gitRepo := &synv1alpha1.GitRepo{}
	repoNamespacedName := types.NamespacedName{
		Namespace: obj.GetObjectMeta().GetNamespace(),
		Name:      obj.GetObjectMeta().GetName(),
	}
	err := data.Client.Get(context.TODO(), repoNamespacedName, gitRepo)
	if err != nil {
		if errors.IsNotFound(err) {
			return ExecutionResult{}
		}
		return ExecutionResult{Abort: true, Err: err}
	}

	obj.SetGitRepoURLAndHostKeys(gitRepo.Status.URL, gitRepo.Status.HostKeys)

	return ExecutionResult{}
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
	instance.Status.URL = repo.FullURL().String()
	if updateErr := client.Status().Update(context.TODO(), instance); updateErr != nil {
		return fmt.Errorf("could not set status while handling error: %s: %s", updateErr, err)
	}
	return err
}

func fetchGitRepoTemplate(obj *synv1alpha1.GitRepo, data *ExecutionContext) error {
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
