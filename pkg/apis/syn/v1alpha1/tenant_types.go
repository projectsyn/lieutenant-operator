package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// DisplayName is the display name of the tenant.
	DisplayName string `json:"displayName"`
	// GitRepoURL git repository storing the tenant configuration. If this is set, no gitRepoTemplate is needed.
	GitRepoURL string `json:"gitRepoURL,omitempty"`
	// GitRepoTemplate Template for managing the GitRepo object. If not set, no  GitRepo object will be created.
	GitRepoTemplate *GitRepoTemplate `json:"gitRepoTemplate,omitempty"`
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
