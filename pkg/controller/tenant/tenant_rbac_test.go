package tenant

import (
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func assertManagedByLabel(t *testing.T, obj metav1.ObjectMeta, value string) {
	v, ok := obj.Labels["app.kubernetes.io/managed-by"]
	assert.True(t, ok, "the label `app.kubernetes.io/managed-by` is not set")
	assert.Equal(t, value, v)
}

func assertOwnerReference(t *testing.T, obj metav1.Object, owner *synv1alpha1.Tenant) {
	refs := obj.GetOwnerReferences()
	require.Len(t, refs, 1)
	assert.Equal(t, owner.GetName(), refs[0].Name)
}

func newReconcileTenant() *ReconcileTenant {
	s := scheme.Scheme
	objs := []runtime.Object{
		&synv1alpha1.Tenant{},
	}
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	c := fake.NewFakeClientWithScheme(s, objs...)
	return &ReconcileTenant{client: c, scheme: s}
}

func TestReconcileTenant_NewServiceAccount(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tenant",
			Namespace: "my-namespace",
		},
	}

	r := newReconcileTenant()
	sa, err := r.NewServiceAccount(tenant)
	require.NoError(t, err)

	assert.Equal(t, tenant.Name, sa.Name)
	assert.Equal(t, tenant.Namespace, sa.Namespace)
	assertManagedByLabel(t, sa.ObjectMeta, "lieutenant")
	assertOwnerReference(t, sa, tenant)
}

func TestReconcileTenant_NewRole(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tenant",
			Namespace: "my-namespace",
		},
	}

	r := newReconcileTenant()
	role, err := r.NewRole(tenant)
	require.NoError(t, err)

	assert.Equal(t, tenant.Name, role.Name)
	assert.Equal(t, tenant.Namespace, role.Namespace)
	assertManagedByLabel(t, role.ObjectMeta, "lieutenant")
	assertOwnerReference(t, role, tenant)
	require.Len(t, role.Rules, 1)

	assert.Equal(t, []string{synv1alpha1.SchemeGroupVersion.Group}, role.Rules[0].APIGroups)
	assert.Len(t, role.Rules[0].Resources, 2)
	assert.Contains(t, role.Rules[0].Resources, "tenants")
	assert.Contains(t, role.Rules[0].Resources, "clusters")
	assert.Len(t, role.Rules[0].Verbs, 1)
	assert.Contains(t, role.Rules[0].Verbs, "get")
}

func TestReconcileTenant_NewRoleBinding(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tenant",
			Namespace: "my-namespace",
		},
	}

	r := newReconcileTenant()
	binding, err := r.NewRoleBinding(tenant)
	require.NoError(t, err)

	assert.Equal(t, tenant.Name, binding.Name)
	assert.Equal(t, tenant.Namespace, binding.Namespace)
	assertManagedByLabel(t, binding.ObjectMeta, "lieutenant")
	assertOwnerReference(t, binding, tenant)
	assert.Equal(t, "Role", binding.RoleRef.Kind)
	assert.Equal(t, tenant.Name, binding.RoleRef.Name)
	require.Len(t, binding.Subjects, 1)
	assert.Equal(t, "ServiceAccount", binding.Subjects[0].Kind)
	assert.Equal(t, tenant.Name, binding.Subjects[0].Name)
}
