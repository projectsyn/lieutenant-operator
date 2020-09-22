package helpers

import (
	"context"
	"fmt"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	protectionSettingEnvVar = "LIEUTENANT_DELETE_PROTECTION"
)

// DeletionState of an object
type DeletionState struct {
	// FinalizerRemoved but not yet deleted
	FinalizerRemoved bool
	// Deleted and gone
	Deleted bool
}

// CreateOrUpdateGitRepo will create the gitRepo object if it doesn't already exist. If the owner object itself is a tenant tenantRef can be set nil.
func CreateOrUpdateGitRepo(obj metav1.Object, gvk schema.GroupVersionKind, template *synv1alpha1.GitRepoTemplate, client client.Client, tenantRef corev1.LocalObjectReference) error {

	if template == nil {
		return fmt.Errorf("gitRepo template is empty")
	}

	if tenantRef.Name == "" {
		return fmt.Errorf("the tenant name is empty")
	}

	if template.DeletionPolicy == "" {
		template.DeletionPolicy = GetDeletionPolicy()
	}

	if template.RepoType == synv1alpha1.DefaultRepoType {
		template.RepoType = synv1alpha1.AutoRepoType
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

	AddDeletionProtection(repo)

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

		err = client.Update(context.TODO(), existingRepo)

	}

	for file, content := range template.TemplateFiles {
		if content == manager.DeletionMagicString {
			delete(template.TemplateFiles, file)
		}
	}

	return err
}

// AddTenantLabel adds the tenant label to an object.
func AddTenantLabel(meta *metav1.ObjectMeta, tenant string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	if meta.Labels[apis.LabelNameTenant] != tenant {
		meta.Labels[apis.LabelNameTenant] = tenant
	}
}

// GetGitRepoURLAndHostKeys gets the URL and host keys for a Git repo
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

// SecretSortList is a list of secrets
type SecretSortList corev1.SecretList

func (s SecretSortList) Len() int      { return len(s.Items) }
func (s SecretSortList) Swap(i, j int) { s.Items[i], s.Items[j] = s.Items[j], s.Items[i] }

func (s SecretSortList) Less(i, j int) bool {

	if s.Items[i].CreationTimestamp.Equal(&s.Items[j].CreationTimestamp) {
		return s.Items[i].Name < s.Items[j].Name
	}

	return s.Items[i].CreationTimestamp.Before(&s.Items[j].CreationTimestamp)
}

// SliceContainsString checks if the slice of strings contains a specific string
func SliceContainsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// HandleDeletion will handle the finalizers if the object was deleted.
// It will return true, if the finalizer was removed. If the object was
// removed the reconcile can be returned.
func HandleDeletion(instance metav1.Object, finalizerName string, client client.Client) DeletionState {
	if instance.GetDeletionTimestamp().IsZero() {
		return DeletionState{FinalizerRemoved: false, Deleted: false}
	}

	annotationValue, exists := instance.GetAnnotations()[DeleteProtectionAnnotation]

	var protected bool
	var err error
	if exists {
		protected, err = strconv.ParseBool(annotationValue)
		// Assume true if it can't be parsed
		if err != nil {
			protected = true
			// We need to reset the error again, so we don't trigger any unwanted side effects...
			err = nil
		}
	} else {
		protected = false
	}

	if SliceContainsString(instance.GetFinalizers(), finalizerName) && !protected {

		controllerutil.RemoveFinalizer(instance, finalizerName)

		return DeletionState{Deleted: true, FinalizerRemoved: true}
	}

	return DeletionState{Deleted: true, FinalizerRemoved: false}
}

// AddDeletionProtection annotation to an object
func AddDeletionProtection(instance metav1.Object) {
	config := os.Getenv(protectionSettingEnvVar)

	protected, err := strconv.ParseBool(config)
	if err != nil {
		protected = true
	}

	if protected {
		annotations := instance.GetAnnotations()

		if annotations == nil {
			annotations = make(map[string]string)
		}

		if _, ok := annotations[DeleteProtectionAnnotation]; !ok {
			annotations[DeleteProtectionAnnotation] = "true"
		}

		instance.SetAnnotations(annotations)
	}

}
