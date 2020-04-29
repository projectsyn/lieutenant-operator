package helpers

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateGitRepo will create the gitRepo object if it doesn't already exist. If the owner object itself is a tenant tenantRef can be set nil.
func CreateOrUpdateGitRepo(obj metav1.Object, gvk schema.GroupVersionKind, template *synv1alpha1.GitRepoTemplate, client client.Client, tenantRef corev1.LocalObjectReference) error {

	if template == nil {
		return fmt.Errorf("gitRepo template is empty")
	}

	if tenantRef.Name == "" {
		return fmt.Errorf("the tenant name is empty")
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
			return fmt.Errorf("could not update existing repo: %v", err)
		}

		if !reflect.DeepEqual(existingRepo.Spec, repo.Spec) {
			existingRepo.Spec = repo.Spec

			err = client.Update(context.TODO(), existingRepo)
		}
	}
	return err
}

// AddTenantLabel adds the tenant label to an object
func AddTenantLabel(meta *metav1.ObjectMeta, tenant string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	meta.Labels[apis.LabelNameTenant] = tenant
}

func GetGitRepoURLAndHostKeys(obj metav1.Object, client client.Client) (string, string, error) {
	gitRepo := &synv1alpha1.GitRepo{}
	repoNamespacedName := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	err := client.Get(context.TODO(), repoNamespacedName, gitRepo)
	if err != nil {
		return "", "", err
	}

	return gitRepo.Status.URL, gitRepo.Status.HostKeys, nil
}

type SecretSortList corev1.SecretList

func (s SecretSortList) Len() int      { return len(s.Items) }
func (s SecretSortList) Swap(i, j int) { s.Items[i], s.Items[j] = s.Items[j], s.Items[i] }

func (s SecretSortList) Less(i, j int) bool {

	if s.Items[i].CreationTimestamp.Equal(&s.Items[j].CreationTimestamp) {
		return s.Items[i].Name < s.Items[j].Name
	}

	return s.Items[i].CreationTimestamp.Before(&s.Items[j].CreationTimestamp)
}
