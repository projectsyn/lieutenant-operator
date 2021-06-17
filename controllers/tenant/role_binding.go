package tenant

import (
	"context"
	"fmt"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/lib/kubernetes/pkg/apis/rbac/v1"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func createRoleBinding(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	binding, err := NewRoleBinding(data.Client.Scheme(), tenant)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to create RoleBinding for tenant: %w", err)}
	}

	err = data.Client.Create(context.TODO(), binding)
	if err != nil && !errors.IsAlreadyExists(err) {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}

// NewRoleBinding returns a new RoleBinding object.
// Its intention is to link subjects to the role created with `Tenant.NewRole()`.
func NewRoleBinding(scheme *runtime.Scheme, tenant *synv1alpha1.Tenant) (*rbacv1.RoleBinding, error) {
	builder := v1.NewRoleBinding(tenant.Name, tenant.Namespace)
	builder.SAs(tenant.Namespace, tenant.Name)

	binding := builder.BindingOrDie()
	setManagedByLabel(&binding)
	if err := controllerutil.SetOwnerReference(tenant, &binding, scheme); err != nil {
		return nil, err
	}

	return &binding, nil
}
