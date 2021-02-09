package role_test

import (
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	roleUtil "github.com/projectsyn/lieutenant-operator/pkg/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
)

var updateRoleTests = map[string]struct {
	name    string
	role    *rbacv1.Role
	updated bool
	assert  func(*testing.T, *rbacv1.Role)
}{
	"tenant name is added to ResourceName": {
		name:    "my-tenant",
		role:    &rbacv1.Role{},
		updated: true,
		assert: func(t *testing.T, role *rbacv1.Role) {
			require.Len(t, role.Rules, 1)
			assert.Len(t, role.Rules[0].ResourceNames, 1)
			assert.Contains(t, role.Rules[0].ResourceNames, "my-tenant")
		},
	},
	"no changes": {
		name: "my-tenant",
		role: &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{synv1alpha1.SchemeGroupVersion.Group},
					Verbs:         []string{"get"},
					Resources:     []string{"tenants", "clusters"},
					ResourceNames: []string{"my-tenant"},
				},
			},
		},
		updated: false,
		assert: func(t *testing.T, role *rbacv1.Role) {
			require.Len(t, role.Rules, 1)
			assert.Len(t, role.Rules[0].ResourceNames, 1)
			assert.Contains(t, role.Rules[0].ResourceNames, "my-tenant")
		},
	},
	"other resource names remain untouched": {
		name: "my-tenant",
		role: &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{synv1alpha1.SchemeGroupVersion.Group},
					Verbs:         []string{"get"},
					Resources:     []string{"tenants", "clusters"},
					ResourceNames: []string{"other-resource"},
				},
			},
		},
		updated: true,
		assert: func(t *testing.T, role *rbacv1.Role) {
			require.Len(t, role.Rules, 1)
			assert.Len(t, role.Rules[0].ResourceNames, 2)
			assert.Contains(t, role.Rules[0].ResourceNames, "my-tenant")
			assert.Contains(t, role.Rules[0].ResourceNames, "other-resource")
		},
	},
	"add new rule when no rule matches": {
		name: "my-tenant",
		role: &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"some-group"},
					Verbs:         []string{"update"},
					Resources:     []string{"some-resource"},
					ResourceNames: []string{"before"},
				},
			},
		},
		updated: true,
		assert: func(t *testing.T, role *rbacv1.Role) {
			require.Len(t, role.Rules, 2)
			assert.Len(t, role.Rules[1].ResourceNames, 1)
			assert.Contains(t, role.Rules[1].ResourceNames, "my-tenant")
		},
	},
	"the correct rule gets updated": {
		name: "my-tenant",
		role: &rbacv1.Role{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"some-group"},
					Verbs:         []string{"update"},
					Resources:     []string{"some-resource"},
					ResourceNames: []string{"before"},
				},
				{
					APIGroups:     []string{synv1alpha1.SchemeGroupVersion.Group},
					Verbs:         []string{"get"},
					Resources:     []string{"tenants", "clusters"},
					ResourceNames: []string{"other-resource"},
				},
				{
					APIGroups:     []string{"some-other-group"},
					Verbs:         []string{"delete"},
					Resources:     []string{"some-other-resource"},
					ResourceNames: []string{"after"},
				},
			},
		},
		updated: true,
		assert: func(t *testing.T, role *rbacv1.Role) {
			require.Len(t, role.Rules, 3)
			assert.Len(t, role.Rules[1].ResourceNames, 2)
			assert.Contains(t, role.Rules[1].ResourceNames, "my-tenant")
			assert.Contains(t, role.Rules[1].ResourceNames, "other-resource")
		},
	},
}

func TestSynchronizeResourceNames(t *testing.T) {
	for name, tt := range updateRoleTests {
		t.Run(name, func(t *testing.T) {
			updated := roleUtil.SynchronizeResourceNames(tt.role, tt.name)
			assert.Equal(t, tt.updated, updated)
			tt.assert(t, tt.role)
		})
	}
}
