package tenant

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func applyTemplateFromTenantTemplate(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	key := types.NamespacedName{Name: "default", Namespace: obj.GetNamespace()}
	template := &synv1alpha1.TenantTemplate{}
	if err := data.Client.Get(context.TODO(), key, template); err != nil {
		if errors.IsNotFound(err) || runtime.IsNotRegisteredError(err) {
			// The absence of a template is not an error.
			// It simply means that there is nothing to do.
			data.Log.Info("No template found to apply to tenant.")
			return pipeline.Result{}
		}
		return pipeline.Result{
			Err:     err,
			Requeue: true,
		}
	}

	if err := tenant.ApplyTemplate(template); err != nil {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}
