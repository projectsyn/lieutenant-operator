package tenant

import (
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	err = data.Client.Create(data.Context(), binding)
	if err != nil && !errors.IsAlreadyExists(err) {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}

// NewRoleBinding returns a new RoleBinding object.
// Its intention is to link subjects to the role created with `Tenant.NewRole()`.
func NewRoleBinding(scheme *runtime.Scheme, tenant *synv1alpha1.Tenant) (*rbacv1.RoleBinding, error) {
	ns := tenant.Namespace
	name := tenant.Name
	binding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, Namespace: ns, Name: name}},
	}

	setManagedByLabel(&binding)
	if err := controllerutil.SetOwnerReference(tenant, &binding, scheme); err != nil {
		return nil, err
	}

	return &binding, nil
}
