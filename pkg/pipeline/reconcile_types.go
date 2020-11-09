package pipeline

import (
	"github.com/go-logr/logr"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PipelineObject defines an interface to extract necessary information from the CRs
type PipelineObject interface {
	GetObjectMeta() metav1.Object
	GetGitTemplate() *synv1alpha1.GitRepoTemplate
	GroupVersionKind() schema.GroupVersionKind
	GetTenantRef() corev1.LocalObjectReference
	GetDeletionPolicy() synv1alpha1.DeletionPolicy
	GetDisplayName() string
	SetGitRepoURLAndHostKeys(URL, hostKeys string)
	GetSpec() interface{}
	GetStatus() interface{}
}

// ExecutionContext contains additional data about the CRD bein processed.
type ExecutionContext struct {
	FinalizerName  string
	Client         client.Client
	Log            logr.Logger
	Deleted        bool
	originalObject PipelineObject
}

// ExecutionResult indicates wether the current execution should be aborted and
// if there was an error.
type ExecutionResult struct {
	Abort bool
	Err   error
}
