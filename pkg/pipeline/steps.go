package pipeline

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"errors"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func GetDefaultDeletionPolicy() synv1alpha1.DeletionPolicy {
	policy := synv1alpha1.DeletionPolicy(os.Getenv("DEFAULT_DELETION_POLICY"))
	switch policy {
	case synv1alpha1.ArchivePolicy, synv1alpha1.DeletePolicy, synv1alpha1.RetainPolicy:
		return policy
	default:
		return synv1alpha1.ArchivePolicy
	}
}

func addDeletionProtection(instance Object, data *Context) Result {
	if data.Deleted {
		return Result{}
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

	return Result{}

}

// AddTenantLabel adds the tenant label to an object.
func AddTenantLabel(obj Object, data *Context) Result {
	labels := obj.GetObjectMeta().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if labels[apis.LabelNameTenant] != obj.GetTenantRef().Name {
		labels[apis.LabelNameTenant] = obj.GetTenantRef().Name
	}

	obj.GetObjectMeta().SetLabels(labels)

	return Result{}
}

func updateObject(obj Object, data *Context) Result {

	rtObj, ok := obj.(runtime.Object)
	if !ok {
		return Result{Err: errors.New("object is not a runtime object")}
	}

	if !specAndMetaEqual(obj, data.originalObject) {
		err := data.Client.Update(context.TODO(), rtObj)
		if err != nil {
			if k8serrors.IsConflict(err) {
				return Result{Requeue: true}
			}
			return Result{Err: err}
		}
	}

	if !equality.Semantic.DeepEqual(data.originalObject.GetStatus(), obj.GetStatus()) {

		err := data.Client.Status().Update(context.TODO(), rtObj)
		if err != nil {
			if k8serrors.IsConflict(err) {
				return Result{Requeue: true}
			}
			return Result{Err: err}
		}
	}

	return Result{Abort: true}
}

func specAndMetaEqual(a, b Object) bool {

	if !equality.Semantic.DeepEqual(a.GetObjectMeta(), b.GetObjectMeta()) {
		return false
	}

	if !equality.Semantic.DeepEqual(a.GetSpec(), b.GetSpec()) {
		return false
	}

	return true

}

// handleDeletion will handle the finalizers if the object was deleted.
// It will only trigger if data.Deleted is true.
func handleDeletion(obj Object, data *Context) Result {
	if !data.Deleted {
		return Result{}
	}

	instance := obj.GetObjectMeta()

	annotationValue, exists := instance.GetAnnotations()[DeleteProtectionAnnotation]

	var protected bool
	var err error
	if exists {
		protected, err = strconv.ParseBool(annotationValue)
		// Assume true if it can't be parsed
		if err != nil {
			protected = true
		}
	} else {
		protected = false
	}

	if sliceContainsString(instance.GetFinalizers(), data.FinalizerName) && !protected {
		return Result{}
	}

	return Result{Err: fmt.Errorf("finalzier was not removed")}
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

func CheckIfDeleted(obj Object, data *Context) Result {
	if !obj.GetObjectMeta().GetDeletionTimestamp().IsZero() {
		data.Deleted = true

	}
	return Result{}

}

func handleFinalizer(obj Object, data *Context) Result {
	if data.FinalizerName != "" && !data.Deleted {
		controllerutil.AddFinalizer(obj.GetObjectMeta(), data.FinalizerName)
	} else {
		controllerutil.RemoveFinalizer(obj.GetObjectMeta(), data.FinalizerName)
	}
	return Result{}
}

func DeepCopyOriginal(obj Object, data *Context) Result {
	rtObj, ok := obj.(runtime.Object)
	if !ok {
		return Result{Err: errors.New("object is not a runtime.Object")}
	}

	copy := rtObj.DeepCopyObject()

	data.originalObject = copy.(Object)

	return Result{}
}
