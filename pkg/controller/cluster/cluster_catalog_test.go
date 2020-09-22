package cluster

import (
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterCatalogTemplating(t *testing.T) {
	tests := []struct {
		name         string
		tenant       *synv1alpha1.Tenant
		cluster      *synv1alpha1.Cluster
		expectedName string
		expectedPath string
		expectedRef  corev1.SecretReference
		expectedErr  string
	}{{
		"Raw string",
		&synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: "acme-tenant",
			},
			Spec: synv1alpha1.TenantSpec{
				ClusterCatalog: synv1alpha1.ClusterCatalog{
					GitRepoTemplate: synv1alpha1.TenantClusterCatalogTemplate{
						RepoName: "the-name",
						Path:     "the-path",
					},
				},
			},
		},
		&synv1alpha1.Cluster{},
		"the-name",
		"the-path",
		corev1.SecretReference{},
		"",
	}, {
		"Templated name",
		&synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: "acme-tenant",
			},
			Spec: synv1alpha1.TenantSpec{
				ClusterCatalog: synv1alpha1.ClusterCatalog{
					GitRepoTemplate: synv1alpha1.TenantClusterCatalogTemplate{
						RepoName: "{{ .ClusterID }}",
					},
				},
			},
		},
		&synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-one",
			},
		},
		"cluster-one",
		"",
		corev1.SecretReference{},
		"",
	}, {
		"Template syntax error",
		&synv1alpha1.Tenant{
			Spec: synv1alpha1.TenantSpec{
				ClusterCatalog: synv1alpha1.ClusterCatalog{
					GitRepoTemplate: synv1alpha1.TenantClusterCatalogTemplate{
						RepoName: "{{ .ClusterID }",
					},
				},
			},
		},
		&synv1alpha1.Cluster{},
		"",
		"",
		corev1.SecretReference{},
		"parse",
	}, {
		"Template data error",
		&synv1alpha1.Tenant{
			Spec: synv1alpha1.TenantSpec{
				ClusterCatalog: synv1alpha1.ClusterCatalog{
					GitRepoTemplate: synv1alpha1.TenantClusterCatalogTemplate{
						RepoName: "{{ .InexistentData }}",
					},
				},
			},
		},
		&synv1alpha1.Cluster{},
		"",
		"",
		corev1.SecretReference{},
		"render",
	}, {
		"Template facts",
		&synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: "t-nameless-breeze-3626",
			},
			Spec: synv1alpha1.TenantSpec{
				ClusterCatalog: synv1alpha1.ClusterCatalog{
					GitRepoTemplate: synv1alpha1.TenantClusterCatalogTemplate{
						RepoName: "{{ .ClusterID }}",
						Path:     `{{ .TenantID }}/{{ index .Facts "cloud" }}`,
					},
				},
			},
		},
		&synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "c-old-fire-5788",
			},
			Spec: synv1alpha1.ClusterSpec{
				Facts: &synv1alpha1.Facts{
					"cloud": "cloudscale",
				},
			},
		},
		"c-old-fire-5788",
		"t-nameless-breeze-3626/cloudscale",
		corev1.SecretReference{},
		"",
	}, {
		"Template secret ref",
		&synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: "t-nameless-breeze-3626",
			},
			Spec: synv1alpha1.TenantSpec{
				ClusterCatalog: synv1alpha1.ClusterCatalog{
					GitRepoTemplate: synv1alpha1.TenantClusterCatalogTemplate{
						APISecretRef: corev1.SecretReference{
							Name:      `{{ index .Facts "lieutenant-instance" }}-api`,
							Namespace: "{{ .TenantID }}-ns",
						},
					},
				},
			},
		},
		&synv1alpha1.Cluster{
			Spec: synv1alpha1.ClusterSpec{
				Facts: &synv1alpha1.Facts{
					"lieutenant-instance": "dev",
				},
			},
		},
		"",
		"",
		corev1.SecretReference{
			Name:      "dev-api",
			Namespace: "t-nameless-breeze-3626-ns",
		},
		"",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setClusterCatalog(tt.cluster, tt.tenant)
			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedName, tt.cluster.Spec.GitRepoTemplate.RepoName)
			assert.Equal(t, tt.expectedPath, tt.cluster.Spec.GitRepoTemplate.Path)
			assert.Equal(t, tt.expectedRef, tt.cluster.Spec.GitRepoTemplate.APISecretRef)
		})
	}
}
