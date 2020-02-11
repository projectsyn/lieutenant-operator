package helpers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateGitRepo will create the gitRepo object if it doesn't already exist. If the owner object itself is a tenant tenantRef can be set nil.
func CreateGitRepo(obj metav1.Object, gvk schema.GroupVersionKind, template *synv1alpha1.GitRepoTemplate, client client.Client, tenantRef corev1.LocalObjectReference) (bool, error) {

	if template == nil {
		return false, fmt.Errorf("gitRepo template is empty")
	}

	if tenantRef.Name == "" {
		return false, fmt.Errorf("the tenant name is empty")
	}

	repo := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, gvk),
			},
		},
		Spec: synv1alpha1.GitRepoSpec{
			GitRepoTemplate: *template,
			TenantRef:       corev1.LocalObjectReference{Name: tenantRef.Name},
		},
	}

	err := client.Create(context.TODO(), repo)
	if err != nil && errors.IsAlreadyExists(err) {
		existingRepo := &synv1alpha1.GitRepo{}

		namespacedName := types.NamespacedName{
			Name:      repo.GetName(),
			Namespace: repo.GetNamespace(),
		}

		err = client.Get(context.TODO(), namespacedName, existingRepo)
		if err != nil {
			return false, fmt.Errorf("could not update existing repo: %v", err)
		}

		existingRepo.Spec = repo.Spec

		return false, client.Update(context.TODO(), existingRepo)
	} else if err != nil {
		return false, err
	}
	return true, nil
}
