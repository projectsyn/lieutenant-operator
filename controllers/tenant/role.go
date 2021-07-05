package tenant

import (
	"context"
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	roleUtil "github.com/projectsyn/lieutenant-operator/role"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func createRole(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	role, err := newRole(data.Client.Scheme(), tenant)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to create Role for tenant: %w", err)}
	}

	err = data.Client.Create(context.TODO(), role)
	if err != nil && !errors.IsAlreadyExists(err) {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}

// newRole returns a new Role object.
// The Role controls access to the tenant and its related clusters.
func newRole(scheme *runtime.Scheme, tenant *synv1alpha1.Tenant) (*rbacv1.Role, error) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: tenant.Namespace,
		},
	}
	setManagedByLabel(role)
	if err := controllerutil.SetOwnerReference(tenant, role, scheme); err != nil {
		return nil, err
	}
	roleUtil.EnsureRules(role)

	return role, nil
}

func tenantUpdateRole(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	name := types.NamespacedName{Name: tenant.Name, Namespace: tenant.Namespace}
	role := &rbacv1.Role{}
	if err := data.Client.Get(context.TODO(), name, role); err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to get role for tenant: %v", err)}
	}

	if !roleUtil.AddResourceNames(role, tenant.Name) {
		return pipeline.Result{}
	}

	if err := data.Client.Update(context.TODO(), role); err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to update role for tenant: %v", err)}
	}

	return pipeline.Result{}
}
