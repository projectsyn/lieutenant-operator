package helpers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"strings"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateGitRepo will create the gitRepo object if it doesn't already exist. If the owner object itself is a tenant tenantRef can be set nil.
func CreateGitRepo(obj metav1.Object, gvk schema.GroupVersionKind, template *synv1alpha1.GitRepoTemplate, client client.Client, tenantRef *corev1.LocalObjectReference) error {

	if template == nil {
		return fmt.Errorf("gitRepo template is empty")
	}

	tenantName := obj.GetName()
	tenantNamespace := obj.GetNamespace()
	if tenantRef != nil {
		tenantName = tenantRef.Name
		tenantNamespace = obj.GetNamespace()
	} else {
		tenantRef = &corev1.LocalObjectReference{}
	}

	repo := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetRepoName(tenantName, gvk),
			Namespace: tenantNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, gvk),
			},
		},
		Spec: synv1alpha1.GitRepoSpec{
			GitRepoTemplate: *template,
			TenantRef:       *tenantRef,
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
			return fmt.Errorf("could not update existing repo: %v", err)
		}

		existingRepo.Spec = repo.Spec

		return client.Update(context.TODO(), existingRepo)
	} else if err != nil {
		return err
	}
	return nil
}

// GetRepoName will return the stable repo name for a given parent f.e. Cluster or Tenant
func GetRepoName(tenantName string, gvk schema.GroupVersionKind) string {
	return strings.ToLower(tenantName + "-" + gvk.Kind)
}
