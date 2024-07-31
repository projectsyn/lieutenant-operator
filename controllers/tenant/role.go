package tenant

import (
	"fmt"
	"slices"

	"github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func reconcileRole(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	cls := v1alpha1.ClusterList{}
	if err := data.Client.List(data.Context, &cls,
		client.InNamespace(tenant.Namespace),
		client.MatchingFields{"spec.tenantRef.name": tenant.Name},
	); err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to list clusters: %w", err)}
	}
	clusterNames := make([]string, 0, len(cls.Items))
	for _, c := range cls.Items {
		clusterNames = append(clusterNames, c.Name)
	}
	slices.Sort(clusterNames)

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: tenant.Namespace,
		},
	}
	op, err := controllerutil.CreateOrUpdate(data.Context, data.Client, &role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{synv1alpha1.GroupVersion.Group},
				Verbs:         []string{"get"},
				Resources:     []string{"tenants"},
				ResourceNames: []string{tenant.Name},
			},
		}
		if len(clusterNames) > 0 {
			role.Rules = append(role.Rules,
				rbacv1.PolicyRule{
					APIGroups:     []string{synv1alpha1.GroupVersion.Group},
					Verbs:         []string{"get"},
					Resources:     []string{"clusters"},
					ResourceNames: clusterNames,
				},
				rbacv1.PolicyRule{
					APIGroups:     []string{synv1alpha1.GroupVersion.Group},
					Verbs:         []string{"get", "update", "patch"},
					Resources:     []string{"clusters/status"},
					ResourceNames: clusterNames,
				})
		}
		return controllerutil.SetControllerReference(tenant, &role, data.Client.Scheme())
	})
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to create or update role (op: %s): %w", op, err)}
	}

	data.Log.Info("Role reconciled", "role", role.Name, "operation", op)
	return pipeline.Result{}
}
