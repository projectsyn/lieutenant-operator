package pipeline

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	protectionSettingEnvVar    = "LIEUTENANT_DELETE_PROTECTION"
	DeleteProtectionAnnotation = "syn.tools/protected-delete"
)

// Object defines an interface to extract necessary information from the CRs
type Object interface {
	client.Object

	GetGitTemplate() *synv1alpha1.GitRepoTemplate
	GroupVersionKind() schema.GroupVersionKind
	GetTenantRef() corev1.LocalObjectReference
	GetDeletionPolicy() synv1alpha1.DeletionPolicy
	GetDisplayName() string
	SetGitRepoURLAndHostKeys(URL, hostKeys string)
	GetMeta() metav1.ObjectMeta
	GetSpec() interface{}
	GetStatus() interface{}
}

// Context contains additional data about the CRD being processed.
type Context struct {
	Context             context.Context
	FinalizerName       string
	Client              client.Client
	Log                 logr.Logger
	Deleted             bool
	originalObject      Object
	Reconciler          reconcile.Reconciler
	CreateSATokenSecret bool
}

// Result indicates whether the current execution should be aborted and
// if there was an error.
type Result struct {
	Abort   bool
	Err     error
	Requeue bool
}

// Function defines the general form of a pipeline function.
type Function func(Object, *Context) Result

type Step struct {
	Name string
	F    Function
}

func RunPipeline(obj Object, data *Context, steps []Step) Result {
	l := data.Log.V(7).WithName("RunPipeline")
	l.Info("running steps", "steps", stepNames(steps))

	for i, step := range steps {
		r := step.F(obj, data)
		l.Info("ran step", "step", step.Name, "result", r, "step_index", i)
		if r.Abort || r.Err != nil {
			if r.Err == nil {
				return Result{Requeue: r.Requeue}
			}
			return Result{Err: fmt.Errorf("step %s failed: %w", step.Name, r.Err)}
		}
	}

	return Result{}
}

func Common(obj Object, data *Context) Result {
	steps := []Step{
		{Name: "deletion", F: handleDeletion},
		{Name: "add deletion protection", F: addDeletionProtection},
		{Name: "handle finalizer", F: handleFinalizer},
		{Name: "update object", F: updateObject},
	}

	return RunPipeline(obj, data, steps)
}

func stepNames(s []Step) []string {
	names := make([]string, len(s))
	for i := range s {
		names[i] = s[i].Name
	}
	return names
}
