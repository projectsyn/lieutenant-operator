package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitType enum values
const (
	GitLab      = GitType("gitlab")
	GitHub      = GitType("github")
	Gitea       = GitType("gitea")
	TypeUnknown = GitType("")
)

const (
	// AutoRepoType managed by the git controller
	AutoRepoType = RepoType("auto")
	// UnmanagedRepoType by the git controller. These objects are only used as data store
	UnmanagedRepoType = RepoType("unmanaged")
	// DefaultRepoType is auto
	DefaultRepoType = RepoType("")
)

// GitPhase enum values
const (
	Created      = GitPhase("created")
	Failed       = GitPhase("failed")
	PhaseUnknown = GitPhase("")
)

// GitPhase is the enum for the git phase status
type GitPhase string

// GitType as the enum for git types
type GitType string

// RepoType specifies the type of the repo
type RepoType string

// GitRepoSpec defines the desired state of GitRepo
type GitRepoSpec struct {
	GitRepoTemplate `json:",inline"`
	// TenantRef references the tenant this repo belongs to
	TenantRef corev1.LocalObjectReference `json:"tenantRef"`
}

// GitRepoTemplate is used for templating git repos, it does not contain the tenantRef as it will be added by the
// controller creating the template instance.
type GitRepoTemplate struct {
	// APISecretRef reference to secret containing connection information
	APISecretRef corev1.SecretReference `json:"apiSecretRef"`
	// DeployKeys optional list of SSH deploy keys. If not set, not deploy keys will be configured
	DeployKeys map[string]DeployKey `json:"deployKeys,omitempty"`
	// Path to Git repository
	Path string `json:"path"`
	// RepoName ame of Git repository
	RepoName string `json:"repoName"`
	// RepoType specifies if a repo should be managed by the git controller. A value of 'unmanaged' means it's not manged by the controller
	// +kubebuilder:default=auto
	// +kubebuilder:validation:Enum=auto;unmanaged
	RepoType RepoType `json:"repoType,omitempty"`
}

// DeployKey defines an SSH key to be used for git operations.
type DeployKey struct {
	Type        string `json:"type,omitempty"`
	Key         string `json:"key,omitempty"`
	WriteAccess bool   `json:"writeAccess,omitempty"`
}

// GitRepoStatus defines the observed state of GitRepo
type GitRepoStatus struct {
	// Updated by Operator with current phase. The GitPhase enum will be used for application logic
	// as using it directly would only print an integer.
	Phase *GitPhase `json:"phase,omitempty"`
	// Type autodiscovered Git repo type. Same behaviour for the enum as with the Phase.
	Type GitType `json:"type,omitempty"`
	// URL computed Git repository URL
	URL string `json:"url,omitempty"`
	// SSH HostKeys of the git server
	HostKeys string `json:"hostKeys,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitRepo is the Schema for the gitrepos API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=gitrepos,scope=Namespaced
// +kubebuilder:printcolumn:name="Repo Name",type="string",JSONPath=".spec.repoName"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type GitRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitRepoSpec   `json:"spec,omitempty"`
	Status GitRepoStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitRepoList contains a list of GitRepo
type GitRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitRepo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitRepo{}, &GitRepoList{})
}
