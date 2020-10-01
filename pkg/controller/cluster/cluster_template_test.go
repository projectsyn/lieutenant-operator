package cluster

import (
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyClusterTemplateRaw(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		Spec: synv1alpha1.TenantSpec{
			ClusterTemplate: &synv1alpha1.ClusterSpec{
				DisplayName: "newname",
				GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
					RepoName: "repo",
					Path:     "path",
					RepoType: synv1alpha1.UnmanagedRepoType,
				},
			},
		},
	}
	cluster := &synv1alpha1.Cluster{
		Spec: synv1alpha1.ClusterSpec{
			DisplayName:   "test",
			TokenLifeTime: "1m",
		},
	}

	err := applyClusterTemplate(cluster, tenant)
	assert.NoError(t, err)
	assert.Equal(t, "test", cluster.Spec.DisplayName)
	assert.Equal(t, "repo", cluster.Spec.GitRepoTemplate.RepoName)
	assert.Equal(t, "path", cluster.Spec.GitRepoTemplate.Path)
	assert.Equal(t, "1m", cluster.Spec.TokenLifeTime)
	assert.Equal(t, synv1alpha1.UnmanagedRepoType, cluster.Spec.GitRepoTemplate.RepoType)
}

func TestApplyClusterTemplate(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		Spec: synv1alpha1.TenantSpec{
			ClusterTemplate: &synv1alpha1.ClusterSpec{
				DisplayName: "test",
				GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
					RepoName: "{{ .Name }}",
					Path:     "{{ .Spec.TenantRef.Name }}",
					APISecretRef: corev1.SecretReference{
						Name: `secret-{{ index .Annotations "syn.tools/tenant"}}`,
					},
				},
			},
		},
	}
	cluster := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "c-some-test",
			Annotations: map[string]string{
				"syn.tools/tenant": "tenant-a",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: "tenant-a",
			},
		},
	}

	err := applyClusterTemplate(cluster, tenant)
	assert.NoError(t, err)
	assert.Equal(t, "test", cluster.Spec.DisplayName)
	assert.Equal(t, "c-some-test", cluster.Spec.GitRepoTemplate.RepoName)
	assert.Equal(t, "tenant-a", cluster.Spec.GitRepoTemplate.Path)
	assert.Equal(t, "{{ .Name }}", tenant.Spec.ClusterTemplate.GitRepoTemplate.RepoName)
	assert.Equal(t, "secret-tenant-a", cluster.Spec.GitRepoTemplate.APISecretRef.Name)
}

func TestApplyClusterTemplateFail(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		Spec: synv1alpha1.TenantSpec{
			ClusterTemplate: &synv1alpha1.ClusterSpec{
				DisplayName: "test",
				GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
					// Wrong go template syntax
					RepoName: "{{ .Name }",
				},
			},
		},
	}
	cluster := &synv1alpha1.Cluster{}

	err := applyClusterTemplate(cluster, tenant)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")

	// Non existent data in template
	tenant.Spec.ClusterTemplate.GitRepoTemplate.RepoName = "{{ .nonexistent }}"
	err = applyClusterTemplate(cluster, tenant)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render")
}
