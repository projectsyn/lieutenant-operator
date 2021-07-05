package cluster

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	roleUtil "github.com/projectsyn/lieutenant-operator/role"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func clusterUpdateRole(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	cluster, ok := obj.(*synv1alpha1.Cluster)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a cluster")}
	}

	name := types.NamespacedName{Name: cluster.Spec.TenantRef.Name, Namespace: cluster.Namespace}
	role := &rbacv1.Role{}
	if err := data.Client.Get(context.TODO(), name, role); err != nil {
		if errors.IsNotFound(err) || runtime.IsNotRegisteredError(err) {
			// The absence of a role is not an error.
			// The role might not yet be created. It gets update on a future reconciliation.
			data.Log.Info("No role found to update.")
			return pipeline.Result{}
		}
		return pipeline.Result{Err: fmt.Errorf("failed to get role for cluster: %v", err)}
	}

	updated := false

	if data.Deleted {
		updated = roleUtil.RemoveResourceNames(role, cluster.Name)
	} else {
		updated = roleUtil.AddResourceNames(role, cluster.Name)
	}

	if !updated {
		return pipeline.Result{}
	}

	if err := data.Client.Update(context.TODO(), role); err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to update role for cluster: %v", err)}
	}

	return pipeline.Result{}
}
