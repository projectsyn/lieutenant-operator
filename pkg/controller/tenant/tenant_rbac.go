package tenant

import (
	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/lib/kubernetes/pkg/apis/rbac/v1"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	roleUtil "github.com/projectsyn/lieutenant-operator/pkg/role"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewServiceAccount returns a new ServiceAccount object.
// The ServiceAccount has its metadata set in a way, that it is clear this ServiceAccount is related to the tenant.
func (r *ReconcileTenant) NewServiceAccount(tenant *synv1alpha1.Tenant) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: tenant.Namespace,
		},
	}
	setManagedByLabel(sa)
	if err := controllerutil.SetOwnerReference(tenant, sa, r.scheme); err != nil {
		return nil, err
	}

	return sa, nil
}

// NewRole returns a new Role object.
// The Role controls access to the tenant and its related clusters.
func (r *ReconcileTenant) NewRole(tenant *synv1alpha1.Tenant) (*rbacv1.Role, error) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: tenant.Namespace,
		},
	}
	setManagedByLabel(role)
	if err := controllerutil.SetOwnerReference(tenant, role, r.scheme); err != nil {
		return nil, err
	}
	roleUtil.EnsureRules(role)

	return role, nil
}

// NewRoleBinding returns a new RoleBinding object.
// Its intention is to link subjects to the role created with `Tenant.NewRole()`.
func (r *ReconcileTenant) NewRoleBinding(tenant *synv1alpha1.Tenant) (*rbacv1.RoleBinding, error) {
	builder := v1.NewRoleBinding(tenant.Name, tenant.Namespace)
	builder.SAs(tenant.Namespace, tenant.Name)

	binding := builder.BindingOrDie()
	setManagedByLabel(&binding)
	if err := controllerutil.SetOwnerReference(tenant, &binding, r.scheme); err != nil {
		return nil, err
	}

	return &binding, nil
}

func setManagedByLabel(obj metav1.Object) {
	obj.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "lieutenant",
	})
}
