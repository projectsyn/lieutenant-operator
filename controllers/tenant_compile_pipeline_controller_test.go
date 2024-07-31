package controllers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_AddBasicPipelineStatus(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.TenantSpec{
			CompilePipeline: &synv1alpha1.CompilePipelineSpec{
				Enabled: true,
			},
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster1", "c-cluster2"},
			},
		},
	}
	c := preparePipelineTestClient(t, tenant)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(tenant))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	i := envVarIndex("COMMODORE_API_URL", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "https://api-url", mod_tenant.GetGitTemplate().CIVariables[i].Value)
	i = envVarIndex("COMMODORE_API_TOKEN", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "t-tenant", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Key)
	i = envVarIndex("CLUSTERS", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "c-cluster1 c-cluster2", mod_tenant.GetGitTemplate().CIVariables[i].Value)
}
func Test_RemoveBasicPipelineStatus(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.TenantSpec{
			CompilePipeline: &synv1alpha1.CompilePipelineSpec{
				Enabled: false,
			},
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				CIVariables: []synv1alpha1.EnvVar{
					{
						Name:  "COMMODORE_API_URL",
						Value: "foo",
					},
					{
						Name:  "COMMODORE_API_TOKEN",
						Value: "foo",
					},
					{
						Name:  "CLUSTERS",
						Value: "foo",
					},
				},
			},
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster1", "c-cluster2"},
			},
		},
	}
	c := preparePipelineTestClient(t, tenant)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(tenant))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	i := envVarIndex("COMMODORE_API_URL", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i < 0)
	i = envVarIndex("COMMODORE_API_TOKEN", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i < 0)
	i = envVarIndex("CLUSTERS", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i < 0)
}

func Test_UpdateBasicPipelineStatus(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.TenantSpec{
			CompilePipeline: &synv1alpha1.CompilePipelineSpec{
				Enabled: true,
			},
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				CIVariables: []synv1alpha1.EnvVar{
					{
						Name:  "COMMODORE_API_URL",
						Value: "foo",
					},
					{
						Name: "COMMODORE_API_TOKEN",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "wrong-secret",
								},
								Key: "token",
							},
						},
					},
					{
						Name:  "CLUSTERS",
						Value: "foo",
					},
				},
			},
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster1", "c-cluster2"},
			},
		},
	}
	c := preparePipelineTestClient(t, tenant)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(tenant))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	i := envVarIndex("COMMODORE_API_URL", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "https://api-url", mod_tenant.GetGitTemplate().CIVariables[i].Value)
	i = envVarIndex("COMMODORE_API_TOKEN", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "t-tenant", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Key)
	i = envVarIndex("CLUSTERS", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "c-cluster1 c-cluster2", mod_tenant.GetGitTemplate().CIVariables[i].Value)
}

func Test_UpdateBasicPipelineStatus_NoUpdate_IfNoChanges(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.TenantSpec{
			CompilePipeline: &synv1alpha1.CompilePipelineSpec{
				Enabled: true,
			},
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				CIVariables: []synv1alpha1.EnvVar{
					{
						Name:  "COMMODORE_API_URL",
						Value: "https://api-url",
					},
					{
						Name: "COMMODORE_API_TOKEN",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "t-tenant",
								},
								Key: "token",
							},
						},
					},
					{
						Name:  "CLUSTERS",
						Value: "c-cluster1 c-cluster2",
					},
				},
			},
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster1", "c-cluster2"},
			},
		},
	}
	c := preparePipelineTestClient(t, tenant)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	orig_tenant := &synv1alpha1.Tenant{}
	err := c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, orig_tenant)
	assert.NoError(t, err)

	_, err = r.Reconcile(ctx, requestFor(tenant))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.Equal(t, orig_tenant.ResourceVersion, mod_tenant.ResourceVersion)
}

func Test_AddClusterCiVars(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.TenantSpec{
			CompilePipeline: &synv1alpha1.CompilePipelineSpec{
				Enabled: true,
			},
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster1", "c-cluster2", "c-cluster3"},
			},
		},
	}
	cluster1 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster1",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: v1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				AccessToken: synv1alpha1.AccessToken{
					SecretRef: "cluster1-token",
				},
			},
		},
	}
	cluster2 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster2",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: v1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				AccessToken: synv1alpha1.AccessToken{
					SecretRef: "cluster2-token",
				},
			},
		},
	}
	cluster3 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster3",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: v1.LocalObjectReference{
				Name: "t-tenant",
			},
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				AccessToken: synv1alpha1.AccessToken{
					SecretRef: "cluster2-token",
				},
			},
		},
	}

	c := preparePipelineTestClient(t, tenant, cluster1, cluster2, cluster3)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(tenant))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	i := envVarIndex("ACCESS_TOKEN_c_cluster1", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "cluster1-token", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Key)
	i = envVarIndex("ACCESS_TOKEN_c_cluster2", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i >= 0)
	assert.Equal(t, "cluster2-token", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", mod_tenant.GetGitTemplate().CIVariables[i].ValueFrom.SecretKeyRef.Key)
	i = envVarIndex("ACCESS_TOKEN_c_cluster3", &mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, i < 0)
}

func Test_ClusterCiVars_NoUpdate_IfNoChanges(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.TenantSpec{
			CompilePipeline: &synv1alpha1.CompilePipelineSpec{
				Enabled: true,
			},
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				CIVariables: []synv1alpha1.EnvVar{
					{
						Name:  "COMMODORE_API_URL",
						Value: "https://api-url",
					},
					{
						Name: "COMMODORE_API_TOKEN",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "t-tenant",
								},
								Key: "token",
							},
						},
					},
					{
						Name:  "CLUSTERS",
						Value: "c-cluster1",
					},
					{
						Name: "ACCESS_TOKEN_c_cluster1",
						ValueFrom: &synv1alpha1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "cluster1-token",
								},
								Key: "token",
							},
						},
					},
				},
			},
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster1"},
			},
		},
	}
	cluster1 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster1",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: v1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				AccessToken: synv1alpha1.AccessToken{
					SecretRef: "cluster1-token",
				},
			},
		},
	}
	cluster2 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster2",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: v1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: false,
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				AccessToken: synv1alpha1.AccessToken{
					SecretRef: "cluster2-token",
				},
			},
		},
	}

	c := preparePipelineTestClient(t, tenant, cluster1, cluster2)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	orig_tenant := &synv1alpha1.Tenant{}
	err := c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, orig_tenant)
	assert.NoError(t, err)

	_, err = r.Reconcile(ctx, requestFor(tenant))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.Equal(t, orig_tenant.ResourceVersion, mod_tenant.ResourceVersion)
}

func tenantCompilePipelineReconciler(c client.Client) *controllers.TenantCompilePipelineReconciler {
	r := controllers.TenantCompilePipelineReconciler{
		Client: c,
		Scheme: c.Scheme(),
		ApiUrl: "https://api-url",
	}
	return &r
}
