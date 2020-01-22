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

// GitPhase enum values
const (
	Creating     = GitPhase("creating")
	Created      = GitPhase("created")
	Updating     = GitPhase("update")
	Deleting     = GitPhase("deleteing")
	Deleted      = GitPhase("deleted")
	PhaseUnknown = GitPhase("")
)

// GitPhase is the enum for the git phase status
type GitPhase string

// GitType as the enum for git types
type GitType string

// IsValid checks if the GitPhase enum is in the valid range
func (g GitType) IsValid() bool {
	return g > GitLab && g < TypeUnknown
}

// GitRepoSpec defines the desired state of GitRepo
type GitRepoSpec struct {
	// APISecretRef reference to secret containing connection information
	APISecretRef *corev1.SecretReference `json:"apiSecretRef,omitempty"`
	// DeployKeys optional list of SSH deploy keys. If not set, not deploy keys will be configured
	DeployKeys []DeployKey `json:"deployKeys,omitempty"`
	// Path to Git repository
	Path string `json:"path,omitempty"`
	// RepoName ame of Git repository
	RepoName string `json:"repoName,omitempty"`
	// TenantRef references the tenant this repo belongs to
	TenantRef *TenantRef `json:"tenantRef,omitempty"`
}

// DeployKey defines an SSH key to be used for git operations.
type DeployKey struct {
	Type        string `json:"type,omitempty"`
	Key         string `json:"key,omitempty"`
	WriteAccess bool   `json:"writeAccess,omitempty"`
}

// GitRepoStatus defines the observed state of GitRepo
type GitRepoStatus struct {
	// Conditions updated by Operator with current conditions
	Conditions []GitRepoConditions `json:"conditions,omitempty"`
	// Updated by Operator with current phase. The GitPhase enum will be used for application logic
	// as using it directly would only print an integer.
	Phase GitPhase `json:"phase,omitempty"`
	// Type autodiscovered Git repo type. Same behaviour for the enum as with the Phase.
	Type GitType `json:"type,omitempty"`
	// URL computed Git repository URL
	URL string `json:"url,omitempty"`
}

// GitRepoConditions contains condition elements for the GitRep CRD
type GitRepoConditions struct {
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Status             string      `json:"status,omitempty"`
	Type               string      `json:"type,omitempty"`
}

// GitRepoTemplate contains a GitRepoSpec for use in other CRDs
type GitRepoTemplate struct {
	Spec GitRepoSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitRepo is the Schema for the gitrepos API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=gitrepos,scope=Namespaced
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
