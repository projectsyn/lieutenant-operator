package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// DisplayName is the display name of the tenant.
	DisplayName string `json:"displayName"`
	// GitRepoURL git repository storing the tenant configuration. If this is set, no gitRepoTemplate is needed.
	GitRepoURL string `json:"gitRepoURL,omitempty"`
	// GitRepoTemplate Template for managing the GitRepo object. If not set, no GitRepo object will be created.
	GitRepoTemplate *GitRepoTemplate `json:"gitRepoTemplate,omitempty"`
	// DeletionPolicy defines how the external resources should be treated upon CR deletion.
	// Retain: will not delete any external resources
	// Delete: will delete the external resources
	// Archive: will archive the external resources, if it supports that
	// +kubebuilder:validation:Enum=Delete;Retain;Archive
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
	// ClusterCatalog defines configuration for cluster catalogs of this tenant
	ClusterCatalog ClusterCatalog `json:"clusterCatalog,omitempty"`
}

// ClusterCatalog defines configuration for cluster catalogs of this tenant
type ClusterCatalog struct {
	// GitRepoTemplate defines the template to be used for the catalog git repos of this tenant.
	// Go template directives can be used in these strings. The `ClusterID`, `TenantID` and
	// all cluster facts can be used within the template.
	GitRepoTemplate TenantClusterCatalogTemplate `json:"gitRepoTemplate,omitempty"`
}

// TenantClusterCatalogTemplate defines the template to be used for the catalog git repos of a tenant
type TenantClusterCatalogTemplate struct {
	// Path to Git repository
	Path string `json:"path,omitempty"`
	// RepoName of Git repository
	RepoName string `json:"repoName,omitempty"`
	// APISecretRef pointing to the secret containing information for connecting to the API of the git server
	APISecretRef corev1.SecretReference `json:"apiSecretRef,omitempty"`
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
