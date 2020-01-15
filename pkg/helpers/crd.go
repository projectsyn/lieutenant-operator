package helpers

import (
	"context"
	"fmt"
	"strings"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateGitRepo will create the gitRepo object if it doesn't already exist. If the owner object itself is a tenant tenantRef can be set nil.
func CreateGitRepo(obj metav1.Object, gvk schema.GroupVersionKind, template *synv1alpha1.GitRepoTemplate, client client.Client, tenantRef *synv1alpha1.TenantRef) error {

	tenantName := obj.GetName()
	tenantNamespace := obj.GetNamespace()
	if tenantRef != nil {
		template.Spec.TenantRef = tenantRef
		tenantName = tenantRef.Name
		tenantNamespace = tenantRef.Namespace
	}

	repo := &synv1alpha1.GitRepo{
		ObjectMeta: metav1.ObjectMeta{
			// Names have to be lowercase
			Name:      strings.ToLower(tenantName + "-" + gvk.Kind),
			Namespace: tenantNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, gvk),
			},
		},
		Spec: template.Spec,
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
