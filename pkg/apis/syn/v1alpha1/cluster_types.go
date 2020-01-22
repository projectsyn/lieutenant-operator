package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FactKey is the key for the facts map
type FactKey string

// FactValue is the value for the facts map
type FactValue string

// Facts is a map of arbitrary facts for the cluster
type Facts map[FactKey]FactValue

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// APIEndpointSecretRef references the secret containing connection information to Kubernetes API endpoint of the registered Kubernetes cluster.
	APIEndpointSecretRef *corev1.SecretReference `json:"apiEndpointSecretRef,omitempty"`
	// DisplayName of cluster which could be different from metadata.name. Allows cluster renaming should it be needed.
	DisplayName string `json:"displayName,omitempty"`
	// GitRepoURL git repository storing the cluster configuration catalog. If this is set, no gitRepoTemplate is needed.
	GitRepoURL string `json:"gitRepoURL,omitempty"`
	// GitRepoTemplate template for managing the GitRepo object.
	GitRepoTemplate *GitRepoTemplate `json:"gitRepoTemplate,omitempty"`
	// TenantRef reference to Tenant object the cluster belongs to.
	TenantRef TenantRef `json:"tenantRef,omitempty"`
	// TokenLifetime set the token lifetime
	TokenLifeTime string `json:"tokenLifeTime,omitempty"`
	// Facts are key/value pairs for statically configured facts
	Facts *Facts `json:"facts,omitempty"`
}

// BootstrapToken this key is used only once for Steward to register.
type BootstrapToken struct {
	Token               string      `json:"token,omitempty"`
	ValidUntil          metav1.Time `json:"validUntil,omitempty"`
	BootstrapTokenValid bool        `json:"bootstrapTokenValid,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// BootstrapTokenValid validity of the bootstrap token, set by the Lieutenant API.
	BootstrapToken *BootstrapToken `json:"bootstrapToken,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusters,scope=Namespaced
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
