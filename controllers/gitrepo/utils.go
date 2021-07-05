package gitrepo

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Create will create the gitRepo object if it doesn't already exist.
func Create(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
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
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, obj.GroupVersionKind()),
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

// UpdateURLAndHostKeys finds the objects and updates the URL and the Host Keys.
func UpdateURLAndHostKeys(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	gitRepo := &synv1alpha1.GitRepo{}
	repoNamespacedName := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
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
