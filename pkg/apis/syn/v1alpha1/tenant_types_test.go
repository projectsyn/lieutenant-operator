package v1alpha1_test

import (
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testTenantTemplateSpec = synv1alpha1.TenantSpec{
	DisplayName:           "Tenant Name",
	GitRepoURL:            "ssh://git@example.com/my-tenant.git",
	GitRepoRevision:       "my-tenant-branch",
	GlobalGitRepoURL:      "ssh://git@example.com/my-config.git",
	GlobalGitRepoRevision: "my-config-branch",
	GitRepoTemplate:       &synv1alpha1.GitRepoTemplate{},
	DeletionPolicy:        synv1alpha1.DeletePolicy,
	ClusterTemplate: &synv1alpha1.ClusterSpec{
		DeletionPolicy: synv1alpha1.ArchivePolicy,
	},
}

var tenantTemplateCases = map[string]struct {
	template *synv1alpha1.TenantTemplate
	tenant   *synv1alpha1.Tenant
	expected *synv1alpha1.Tenant
}{
	"template gets applied": {
		template: &synv1alpha1.TenantTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name: "my-template",
			},
			Spec: testTenantTemplateSpec,
		},
		tenant: &synv1alpha1.Tenant{},
		expected: &synv1alpha1.Tenant{
			ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					"lieutenant.syn.tools/tenant-template": "my-template",
				},
			},
			Spec: testTenantTemplateSpec,
		},
	},
	"tenant values take precedence": {
		template: &synv1alpha1.TenantTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name: "my-template",
			},
			Spec: synv1alpha1.TenantSpec{
				DeletionPolicy: synv1alpha1.DeletePolicy,
			},
		},
		tenant: &synv1alpha1.Tenant{
			Spec: synv1alpha1.TenantSpec{
				DisplayName:    "Tenant Name",
				DeletionPolicy: synv1alpha1.ArchivePolicy,
			},
		},
		expected: &synv1alpha1.Tenant{
			ObjectMeta: v1.ObjectMeta{
				Annotations: map[string]string{
					"lieutenant.syn.tools/tenant-template": "my-template",
				},
			},
			Spec: synv1alpha1.TenantSpec{
				DisplayName:    "Tenant Name",
				DeletionPolicy: synv1alpha1.ArchivePolicy,
			},
		},
	},
}

func TestTenant_ApplyTemplate(t *testing.T) {
	for name, tt := range tenantTemplateCases {
		t.Run(name, func(t *testing.T) {
			actual := tt.tenant.DeepCopy()
			err := actual.ApplyTemplate(tt.template)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
