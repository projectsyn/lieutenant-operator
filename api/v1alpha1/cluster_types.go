package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Facts is a map of arbitrary facts for the cluster
type Facts map[string]string

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// DisplayName of cluster which could be different from metadata.name. Allows cluster renaming should it be needed.
	DisplayName string `json:"displayName,omitempty"`
	// GitRepoURL git repository storing the cluster configuration catalog. If this is set, no gitRepoTemplate is needed.
	GitRepoURL string `json:"gitRepoURL,omitempty"`
	// SSH GitHostKeys of the git server
	GitHostKeys string `json:"gitHostKeys,omitempty"`
	// GitRepoTemplate template for managing the GitRepo object.
	GitRepoTemplate *GitRepoTemplate `json:"gitRepoTemplate,omitempty"`
	// TenantRef reference to Tenant object the cluster belongs to.
	TenantRef corev1.LocalObjectReference `json:"tenantRef,omitempty"`
	// TenantGitRepoRevision allows to configure the revision of the tenant configuration to use. It can be any git tree-ish reference. The revision from the tenant will be inherited if left empty.
	TenantGitRepoRevision string `json:"tenantGitRepoRevision,omitempty"`
	// GlobalGitRepoRevision allows to configure the revision of the global configuration to use. It can be any git tree-ish reference. The revision from the tenant will be inherited if left empty.
	GlobalGitRepoRevision string `json:"globalGitRepoRevision,omitempty"`
	// TokenLifetime set the token lifetime
	TokenLifeTime string `json:"tokenLifeTime,omitempty"`
	// Facts are key/value pairs for statically configured facts
	Facts Facts `json:"facts,omitempty"`
	// DeletionPolicy defines how the external resources should be treated upon CR deletion.
	// Retain: will not delete any external resources
	// Delete: will delete the external resources
	// Archive: will archive the external resources, if it supports that
	// +kubebuilder:validation:Enum=Delete;Retain;Archive
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
	// CreationPolicy defines how the external resources should be treated upon CR creation.
	// Create: will only create a new external resource and will not manage already existing resources
	// Adopt:  will create a new external resource or will adopt and manage an already existing resource
	// +kubebuilder:validation:Enum=Create;Adopt
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`
	// EnableCompilePipeline determines whether the gitops compile pipeline should be set up for this cluster
	EnableCompilePipeline bool `json:"enableCompilePipeline,omitempty"`
}

// BootstrapToken this key is used only once for Steward to register.
type BootstrapToken struct {
	// Token is the actual token to register the cluster
	Token string `json:"token,omitempty"`
	// ValidUntil timespan how long the token is valid. If the token is
	// used after this timestamp it will be rejected.
	ValidUntil metav1.Time `json:"validUntil,omitempty"`
	// TokenValid indicates if the token is still valid or was already used.
	TokenValid bool `json:"tokenValid,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// BootstrapTokenValid validity of the bootstrap token, set by the Lieutenant API.
	BootstrapToken *BootstrapToken `json:"bootstrapToken,omitempty"`
	// Facts are key/value pairs for dynamically fetched facts
	Facts Facts `json:"facts,omitempty"`
	// CompileMeta contains information about the last compilation with Commodore.
	CompileMeta CompileMeta `json:"compileMeta,omitempty"`
}

// CompileMeta contains information about the last compilation with Commodore.
type CompileMeta struct {
	// LastCompile is the time of the last successful compilation.
	LastCompile metav1.Time `json:"lastCompile,omitempty"`
	// CommodoreBuildInfo is the freeform build information reported by the Commodore binary used for the last compilation.
	CommodoreBuildInfo map[string]string `json:"commodoreBuildInfo,omitempty"`
	// Global contains the information of the global configuration used for the last compilation.
	Global CompileMetaVersionInfo `json:"global,omitempty"`
	// Tenant contains the information of the tenant configuration used for the last compilation.
	Tenant CompileMetaVersionInfo `json:"tenant,omitempty"`
	// Packages contains the information of the packages used for the last compilation.
	Packages map[string]CompileMetaVersionInfo `json:"packages,omitempty"`
	// Instances contains the information of the component instances used for the last compilation.
	// The key is the name of the component instance.
	Instances map[string]CompileMetaInstanceVersionInfo `json:"instances,omitempty"`
}

// CompileMetaVersionInfo contains information about the version of a configuration repo or a package.
type CompileMetaVersionInfo struct {
	// URL is the URL of the git repository.
	URL string `json:"url,omitempty"`
	// GitSHA is the git commit SHA of the used commit.
	GitSHA string `json:"gitSha,omitempty"`
	// Version is the version of the configuration.
	// Can point to a tag, branch or any other git reference.
	Version string `json:"version,omitempty"`
	// Path is the path inside the git repository where the configuration is stored.
	Path string `json:"path,omitempty"`
}

// CompileMetaInstanceVersionInfo contains information about the version of a component instance.
type CompileMetaInstanceVersionInfo struct {
	CompileMetaVersionInfo `json:",inline"`

	// Component is the name of a component instance.
	Component string `json:"component,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusters,scope=Namespaced
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName"
// +kubebuilder:printcolumn:name="Tenant",type="string",JSONPath=".spec.tenantRef.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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

// GetGitTemplate returns the git repository template
func (c *Cluster) GetGitTemplate() *GitRepoTemplate {
	return c.Spec.GitRepoTemplate
}

// GetTenantRef returns the tenant of this CR
func (c *Cluster) GetTenantRef() corev1.LocalObjectReference {
	return c.Spec.TenantRef
}

// GetDeletionPolicy returns the object's deletion policy
func (c *Cluster) GetDeletionPolicy() DeletionPolicy {
	return c.Spec.DeletionPolicy
}

// GetCreationPolicy returns the object's creation policy
func (c *Cluster) GetCreationPolicy() CreationPolicy {
	return c.Spec.CreationPolicy
}

// GetDisplayName returns the display name of the object
func (c *Cluster) GetDisplayName() string {
	return c.Spec.DisplayName
}

// SetGitRepoURLAndHostKeys updates the GitRepoURL and the GitHostKeys in the Cluster at once
func (c *Cluster) SetGitRepoURLAndHostKeys(URL, hostKeys string) {
	c.Spec.GitRepoURL = URL
	c.Spec.GitHostKeys = hostKeys
}

func (c *Cluster) GetSpec() interface{} {
	return c.Spec
}

func (c *Cluster) GetMeta() metav1.ObjectMeta {
	return c.ObjectMeta
}

func (c *Cluster) GetStatus() interface{} {
	return c.Status
}
