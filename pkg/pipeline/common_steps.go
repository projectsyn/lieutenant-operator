package pipeline

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func getDefaultDeletionPolicy() synv1alpha1.DeletionPolicy {
	policy := synv1alpha1.DeletionPolicy(os.Getenv("DEFAULT_DELETION_POLICY"))
	switch policy {
	case synv1alpha1.ArchivePolicy, synv1alpha1.DeletePolicy, synv1alpha1.RetainPolicy:
		return policy
	default:
		return synv1alpha1.ArchivePolicy
	}
}

func addDeletionProtection(instance PipelineObject, data *ExecutionContext) ExecutionResult {

	if data.Deleted {
		return ExecutionResult{}
	}

	config := os.Getenv(protectionSettingEnvVar)

	protected, err := strconv.ParseBool(config)
	if err != nil {
		protected = true
	}

	if protected {
		annotations := instance.GetObjectMeta().GetAnnotations()

		if annotations == nil {
			annotations = make(map[string]string)
		}

		if _, ok := annotations[DeleteProtectionAnnotation]; !ok {
			annotations[DeleteProtectionAnnotation] = "true"
		}

		instance.GetObjectMeta().SetAnnotations(annotations)
	}

	return ExecutionResult{}

}

// addTenantLabel adds the tenant label to an object.
func addTenantLabel(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	labels := obj.GetObjectMeta().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if labels[apis.LabelNameTenant] != obj.GetTenantRef().Name {
		labels[apis.LabelNameTenant] = obj.GetTenantRef().Name
	}

	obj.GetObjectMeta().SetLabels(labels)

	return ExecutionResult{}
}

func updateObject(obj PipelineObject, data *ExecutionContext) ExecutionResult {

	resourceVersion := obj.GetObjectMeta().GetResourceVersion()

	rtObj, ok := obj.(runtime.Object)
	if !ok {
		return ExecutionResult{
			Abort: true,
			Err:   fmt.Errorf("object ist not a valid runtime.object: %v", obj.GetObjectMeta().GetName()),
		}
	}

	err := data.Client.Update(context.TODO(), rtObj)
	if err != nil {
		return ExecutionResult{Err: err}
	}

	// Updating the status if either there were changes or the object is deleted will
	// lead to some race conditions. By checking first we can avoid them.
	if resourceVersion == obj.GetObjectMeta().GetResourceVersion() && !data.Deleted {
		err = data.Client.Status().Update(context.TODO(), rtObj)
	}
	return ExecutionResult{Abort: true, Err: err}
}

func wrapError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("step %s failed: %w", name, err)
}

func resultNotOK(result ExecutionResult) bool {
	return result.Abort || result.Err != nil
}

// handleDeletion will handle the finalizers if the object was deleted.
// It will only trigger if data.Deleted is true.
func handleDeletion(instance metav1.Object, data *ExecutionContext) ExecutionResult {
	if !data.Deleted {
		return ExecutionResult{}
	}

	annotationValue, exists := instance.GetAnnotations()[DeleteProtectionAnnotation]

	var protected bool
	var err error
	if exists {
		protected, err = strconv.ParseBool(annotationValue)
		// Assume true if it can't be parsed
		if err != nil {
			protected = true
			// We need to reset the error again, so we don't trigger any unwanted side effects...
			err = nil
		}
	} else {
		protected = false
	}

	if sliceContainsString(instance.GetFinalizers(), data.FinalizerName) && !protected {

		data.Deleted = true
		return ExecutionResult{}
	}

	return ExecutionResult{Err: fmt.Errorf("finalzier was not removed")}
}

// Checks if the slice of strings contains a specific string
func sliceContainsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func checkIfDeleted(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	if !obj.GetObjectMeta().GetDeletionTimestamp().IsZero() {
		data.Deleted = true

	}
	return ExecutionResult{}

}

func handleFinalizer(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	if data.FinalizerName != "" && !data.Deleted {
		controllerutil.AddFinalizer(obj.GetObjectMeta(), data.FinalizerName)
	} else {
		controllerutil.RemoveFinalizer(obj.GetObjectMeta(), data.FinalizerName)
	}
	return ExecutionResult{}
}
