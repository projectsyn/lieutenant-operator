package cluster

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func setTenantOwner(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant := &synv1alpha1.Tenant{}
	tenantName := types.NamespacedName{Name: obj.GetTenantRef().Name, Namespace: obj.GetNamespace()}

	err := data.Client.Get(context.TODO(), tenantName, tenant)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	obj.SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(tenant, tenant.GroupVersionKind()),
	})

	return pipeline.Result{}
}

func applyClusterTemplateFromTenant(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	nsName := types.NamespacedName{Name: obj.GetTenantRef().Name, Namespace: obj.GetNamespace()}

	tenant := &synv1alpha1.Tenant{}
	if err := data.Client.Get(context.TODO(), nsName, tenant); err != nil {
		return pipeline.Result{Err: fmt.Errorf("couldn't find tenant: %w", err)}
	}

	instance, ok := obj.(*synv1alpha1.Cluster)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a cluster")}
	}

	if err := applyClusterTemplate(instance, tenant); err != nil {
		return pipeline.Result{Err: err}
	}
	return pipeline.Result{}
}
