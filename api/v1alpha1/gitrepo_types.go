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
	Created       GitPhase       = "created"
	Creating      GitPhase       = "creating"
	Failed        GitPhase       = "failed"
	PhaseUnknown  GitPhase       = ""
	ArchivePolicy DeletionPolicy = "Archive"
	DeletePolicy  DeletionPolicy = "Delete"
	RetainPolicy  DeletionPolicy = "Retain"
	CreatePolicy  CreationPolicy = "Create"
	AdoptPolicy   CreationPolicy = "Adopt"
)

// GitPhase is the enum for the git phase status
type GitPhase string

// GitType as the enum for git types
type GitType string

// RepoType specifies the type of the repo
type RepoType string

// DeletionPolicy defines the type deletion policy
type DeletionPolicy string

// CreationPolicy defines the type creation policy
type CreationPolicy string

// GitRepoSpec defines the desired state of GitRepo
type GitRepoSpec struct {
	GitRepoTemplate `json:",inline"`
	// TenantRef references the tenant this repo belongs to
	TenantRef corev1.LocalObjectReference `json:"tenantRef,omitempty"`
}

// GitRepoTemplate is used for templating git repos, it does not contain the tenantRef as it will be added by the
// controller creating the template instance.
type GitRepoTemplate struct {
	// APISecretRef reference to secret containing connection information
	APISecretRef corev1.SecretReference `json:"apiSecretRef,omitempty"`
	// DeployKeys optional list of SSH deploy keys. If not set, not deploy keys will be configured
	DeployKeys map[string]DeployKey `json:"deployKeys,omitempty"`
	// Path to Git repository
	Path string `json:"path,omitempty"`
	// RepoName name of Git repository
	RepoName string `json:"repoName,omitempty"`
	// RepoType specifies if a repo should be managed by the git controller. A value of 'unmanaged' means it's not manged by the controller
	// +kubebuilder:validation:Enum=auto;unmanaged
	RepoType RepoType `json:"repoType,omitempty"`
	// DisplayName of Git repository
	DisplayName string `json:"displayName,omitempty"`
	// TemplateFiles is a list of files that should be pushed to the repository
	// after its creation.
	TemplateFiles map[string]string `json:"templateFiles,omitempty"`
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
	// AccessToken contains configuration for storing an access token in a secret.
	// If set, the Lieutenant operator will store an access token into this secret, which can be used to access the Git repository.
	// The token is stored under the key "token".
	// In the case of GitLab, this would be a Project Access Token with read-write access to the repository.
	AccessToken AccessToken `json:"accessToken,omitempty"`
	// CIVariables is a list of key-value pairs that will be set as CI variables in the Git repository.
	//
	// The variables are not expanded like PodSpec environment variables.
	CIVariables []EnvVar `json:"ciVariables,omitempty"`
}

type AccessToken struct {
	// SecretRef references the secret the access token is stored in
	SecretRef string `json:"secretRef,omitempty"`
}

// EnvVar represents an environment added to the CI system of the Git repository.
type EnvVar struct {
	// Name of the environment variable
	// +required
	Name string `json:"name"`
	// Value of the environment variable
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom is a reference to an object that contains the value of the environment variable
	// +optional
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`

	// GitlabOptions contains additional options for GitLab CI variables
	// +optional
	GitlabOptions EnvVarGitlabOptions `json:"gitlabOptions,omitempty"`
}

type EnvVarGitlabOptions struct {
	// Description is a description of the CI variable.
	// +optional
	Description string `json:"description,omitempty"`
	// Protected will expose the variable only in protected branches and tags.
	// +optional
	Protected bool `json:"protected,omitempty"`
	// Masked will mask the variable in the job logs.
	// +optional
	Masked bool `json:"masked,omitempty"`
	// Raw will prevent the variable from being expanded.
	// +optional
	Raw bool `json:"raw,omitempty"`
}

// EnvVarSource represents a source for the value of an EnvVar.
type EnvVarSource struct {
	// Selects a key of a secret in the pod's namespace
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// DeployKey defines an SSH key to be used for git operations.
type DeployKey struct {
	// Type defines what type the key is (rsa, ed25519, etc...)
	Type string `json:"type,omitempty"`
	// Key is the actual key
	Key string `json:"key,omitempty"`
	// WriteAccess if the key has RW access or not
	WriteAccess bool `json:"writeAccess,omitempty"`
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
	// LastAppliedCIVariables contains the last applied CI variables as a json string
	LastAppliedCIVariables string `json:"lastAppliedCIVariables,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GitRepo is the Schema for the gitrepos API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=gitrepos,scope=Namespaced
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName"
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

// GetGitTemplate returns the git repository template
func (g *GitRepo) GetGitTemplate() *GitRepoTemplate {
	return &g.Spec.GitRepoTemplate
}

// GetTenantRef returns the tenant of this CR
func (g *GitRepo) GetTenantRef() corev1.LocalObjectReference {
	return g.Spec.TenantRef
}

// GetDeletionPolicy returns the object's deletion policy
func (g *GitRepo) GetDeletionPolicy() DeletionPolicy {
	return g.Spec.DeletionPolicy
}

// GetCreationPolicy returns the object's creation policy
func (g *GitRepo) GetCreationPolicy() CreationPolicy {
	return g.Spec.CreationPolicy
}

// GetDisplayName returns the display name of the object
func (g *GitRepo) GetDisplayName() string {
	return g.Spec.DisplayName
}

// SetGitRepoURLAndHostKeys is currenlty a noop for gitrepo
func (g *GitRepo) SetGitRepoURLAndHostKeys(URL, hostKeys string) {
	//NOOP
}

func (g *GitRepo) GetSpec() interface{} {
	return g.Spec
}

func (g *GitRepo) GetMeta() metav1.ObjectMeta {
	return g.ObjectMeta
}

func (g *GitRepo) GetStatus() interface{} {
	return g.Status
}
