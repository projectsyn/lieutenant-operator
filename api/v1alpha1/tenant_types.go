package v1alpha1

import (
	"fmt"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// DisplayName is the display name of the tenant.
	DisplayName string `json:"displayName,omitempty"`
	// GitRepoURL git repository storing the tenant configuration. If this is set, no gitRepoTemplate is needed.
	GitRepoURL string `json:"gitRepoURL,omitempty"`
	// GitRepoRevision allows to configure the revision of the tenant configuration to use. It can be any git tree-ish reference. Defaults to HEAD if left empty.
	GitRepoRevision string `json:"gitRepoRevision,omitempty"`
	// GlobalGitRepoURL git repository storing the global configuration.
	GlobalGitRepoURL string `json:"globalGitRepoURL,omitempty"`
	// GlobalGitRepoRevision allows to configure the revision of the global configuration to use. It can be any git tree-ish reference. Defaults to HEAD if left empty.
	GlobalGitRepoRevision string `json:"globalGitRepoRevision,omitempty"`
	// GitRepoTemplate Template for managing the GitRepo object. If not set, no GitRepo object will be created.
	GitRepoTemplate *GitRepoTemplate `json:"gitRepoTemplate,omitempty"`
	// DeletionPolicy defines how the external resources should be treated upon CR deletion.
	// Retain: will not delete any external resources
	// Delete: will delete the external resources
	// Archive: will archive the external resources, if it supports that
	// +kubebuilder:validation:Enum=Delete;Retain;Archive
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
	// ClusterTemplate defines a template which will be used to set defaults for the clusters of this tenant.
	// The fields whitin this can use Go templating.
	// See [the docs](https://syn.tools/lieutenant-operator/explanation/templating.html) for details.
	ClusterTemplate *ClusterSpec `json:"clusterTemplate,omitempty"`
}

// TenantStatus defines the observed state of Tenant
type TenantStatus struct {
	// TBD
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Tenant is the Schema for the tenants API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=tenants,scope=Namespaced
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantList contains a list of Tenant
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}

// GetGitTemplate returns the git repository template
func (t *Tenant) GetGitTemplate() *GitRepoTemplate {
	if t.Spec.GitRepoTemplate == nil {
		t.Spec.GitRepoTemplate = &GitRepoTemplate{}
	}
	return t.Spec.GitRepoTemplate
}

// GetTenantRef returns the tenant of this CR
func (t *Tenant) GetTenantRef() corev1.LocalObjectReference {
	return corev1.LocalObjectReference{Name: t.GetName()}
}

// GetDeletionPolicy returns the object's deletion policy
func (t *Tenant) GetDeletionPolicy() DeletionPolicy {
	return t.Spec.DeletionPolicy
}

// GetDisplayName returns the display name of the object
func (t *Tenant) GetDisplayName() string {
	return t.Spec.DisplayName
}

// SetGitRepoURLAndHostKeys will only set the URL for the tenant
func (t *Tenant) SetGitRepoURLAndHostKeys(URL, _ string) {
	t.Spec.GitRepoURL = URL
}

func (t *Tenant) GetSpec() interface{} {
	return t.Spec
}

func (t *Tenant) GetMeta() metav1.ObjectMeta {
	return t.ObjectMeta
}

func (t *Tenant) GetStatus() interface{} {
	return t.Status
}

// ApplyTemplate recursively merges in the values of the given template.
// The values of the tenant takes precedence.
func (t *Tenant) ApplyTemplate(template *TenantTemplate) error {
	if template == nil {
		return nil
	}

	if err := mergo.Merge(&t.Spec, template.Spec); err != nil {
		return fmt.Errorf("failed to merge tenant template into tenant: %w", err)
	}

	if t.ObjectMeta.Annotations == nil {
		t.ObjectMeta.Annotations = map[string]string{}
	}

	t.ObjectMeta.Annotations["lieutenant.syn.tools/tenant-template"] = template.ObjectMeta.Name

	return nil
}
