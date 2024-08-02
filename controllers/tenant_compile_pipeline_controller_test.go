package controllers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers"
)

func Test_TenantCompilePipelineReconciler_AddVariablesStatusAndFinalizer(t *testing.T) {
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
	}
	c1 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster1",
			Namespace: "lieutenant",
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}
	c2 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster2",
			Namespace: "lieutenant",
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}

	c := preparePipelineTestClient(t, tenant, c1, c2)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(tenant))
	require.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	require.NoError(t, err)

	require.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	v, found := findEnvVar("COMMODORE_API_URL", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "https://api-url", v.Value)
	v, found = findEnvVar("COMMODORE_API_TOKEN", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "t-tenant", v.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", v.ValueFrom.SecretKeyRef.Key)
	v, found = findEnvVar("CLUSTERS", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "c-cluster1 c-cluster2", v.Value)

	require.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Equal(t, []string{"c-cluster1", "c-cluster2"}, mod_tenant.GetCompilePipelineStatus().Clusters)

	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: "c-cluster1", Namespace: "lieutenant"}, c1))
	require.Contains(t, c1.GetFinalizers(), synv1alpha1.PipelineFinalizerName)
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: "c-cluster2", Namespace: "lieutenant"}, c2))
	require.Contains(t, c2.GetFinalizers(), synv1alpha1.PipelineFinalizerName)
}

func Test_TenantCompilePipelineReconciler_DisableCompilePipeline_RemoveVariablesStatus(t *testing.T) {
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
					{
						Name:  "ACCESS_TOKEN_c_cluster1",
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
	c1 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster1",
			Namespace: "lieutenant",
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, c1)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(tenant))
	require.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	require.NoError(t, err)

	require.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	_, found := findEnvVar("COMMODORE_API_URL", mod_tenant.GetGitTemplate().CIVariables)
	assert.False(t, found)
	_, found = findEnvVar("COMMODORE_API_TOKEN", mod_tenant.GetGitTemplate().CIVariables)
	assert.False(t, found)
	_, found = findEnvVar("CLUSTERS", mod_tenant.GetGitTemplate().CIVariables)
	assert.False(t, found)
	_, found = findEnvVar("ACCESS_TOKEN_c-cluster1", mod_tenant.GetGitTemplate().CIVariables)
	assert.False(t, found)

	require.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Empty(t, mod_tenant.GetCompilePipelineStatus().Clusters)
}

func Test_TenantCompilePipelineReconciler_UpdateBasicPipelineStatus(t *testing.T) {
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
	}
	c1 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster1",
			Namespace: "lieutenant",
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}
	c2 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster2",
			Namespace: "lieutenant",
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, c1, c2)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(tenant))
	require.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	require.NoError(t, err)

	require.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	v, found := findEnvVar("COMMODORE_API_URL", mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, found)
	assert.Equal(t, "https://api-url", v.Value)
	v, found = findEnvVar("COMMODORE_API_TOKEN", mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, found)
	assert.Equal(t, "t-tenant", v.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", v.ValueFrom.SecretKeyRef.Key)
	v, found = findEnvVar("CLUSTERS", mod_tenant.GetGitTemplate().CIVariables)
	assert.True(t, found)
	assert.Equal(t, "c-cluster1 c-cluster2", v.Value)
}

func Test_TenantCompilePipelineReconciler_UpdateBasicPipelineStatus_NoUpdate_IfNoChanges(t *testing.T) {
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
	c1 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster1",
			Namespace: "lieutenant",
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}
	c2 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster2",
			Namespace: "lieutenant",
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}

	c := preparePipelineTestClient(t, tenant, c1, c2)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	orig_tenant := &synv1alpha1.Tenant{}
	err := c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, orig_tenant)
	require.NoError(t, err)

	_, err = r.Reconcile(ctx, requestFor(tenant))
	require.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	require.NoError(t, err)

	assert.Equal(t, orig_tenant.ResourceVersion, mod_tenant.ResourceVersion)
}

func Test_TenantCompilePipelineReconciler_AddClusterCiVars(t *testing.T) {
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
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
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
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
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
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
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
	require.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	require.NoError(t, err)

	require.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	v, found := findEnvVar("ACCESS_TOKEN_c_cluster1", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "cluster1-token", v.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", v.ValueFrom.SecretKeyRef.Key)
	v, found = findEnvVar("ACCESS_TOKEN_c_cluster2", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "cluster2-token", v.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", v.ValueFrom.SecretKeyRef.Key)
	_, found = findEnvVar("ACCESS_TOKEN_c_cluster3", mod_tenant.GetGitTemplate().CIVariables)
	assert.False(t, found)
}

func Test_TenantCompilePipelineReconciler_ClusterCiVars_NoUpdate_IfNoChanges(t *testing.T) {
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
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
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
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
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
	require.NoError(t, err)

	_, err = r.Reconcile(ctx, requestFor(tenant))
	require.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	require.NoError(t, err)

	assert.Equal(t, orig_tenant.ResourceVersion, mod_tenant.ResourceVersion)
}

func Test_TenantCompilePipelineReconciler_DeleteClusterRemoveVariableStatusFinalizer(t *testing.T) {
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
						Name:  "ACCESS_TOKEN_c_cluster1",
						Value: "foo",
					},
					{
						Name:  "ACCESS_TOKEN_c_cluster2",
						Value: "foo",
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
	c1 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "c-cluster1",
			Namespace:  "lieutenant",
			Finalizers: []string{synv1alpha1.PipelineFinalizerName},
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}
	c2 := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "c-cluster2",
			Namespace:  "lieutenant",
			Finalizers: []string{synv1alpha1.PipelineFinalizerName},
			Labels: map[string]string{
				synv1alpha1.LabelNameTenant: "t-tenant",
			},
		},
		Spec: synv1alpha1.ClusterSpec{
			EnableCompilePipeline: true,
		},
	}

	c := preparePipelineTestClient(t, tenant, c1, c2)
	r := tenantCompilePipelineReconciler(c)
	ctx := context.Background()

	require.NoError(t, c.Delete(ctx, c2))
	_, err := r.Reconcile(ctx, requestFor(tenant))
	require.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	require.NoError(t, err)

	require.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	v, found := findEnvVar("COMMODORE_API_URL", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "https://api-url", v.Value)
	v, found = findEnvVar("COMMODORE_API_TOKEN", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "t-tenant", v.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "token", v.ValueFrom.SecretKeyRef.Key)
	v, found = findEnvVar("CLUSTERS", mod_tenant.GetGitTemplate().CIVariables)
	require.True(t, found)
	assert.Equal(t, "c-cluster1", v.Value)

	require.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Equal(t, []string{"c-cluster1"}, mod_tenant.GetCompilePipelineStatus().Clusters)

	err = c.Get(ctx, types.NamespacedName{Name: "c-cluster2", Namespace: "lieutenant"}, c2)
	require.NotNil(t, err)
	require.True(t, apierrors.IsNotFound(err))
}

func Test_TenantCompilePipelineReconciler_IgnoreUnmanagedRepoForCIVariableUpdate(t *testing.T) {
	for _, tc := range []struct {
		preExisting            bool
		clusterPipelineEnabled bool
	}{
		{
			preExisting:            false,
			clusterPipelineEnabled: true,
		},
		{
			preExisting:            true,
			clusterPipelineEnabled: true,
		},
		{
			preExisting:            true,
			clusterPipelineEnabled: false,
		},
	} {
		t.Run(fmt.Sprintf("PreExisting=%t,ClusterPipelineEnabled=%t", tc.preExisting, tc.clusterPipelineEnabled), func(t *testing.T) {
			tenant := &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "t-tenant",
					Namespace: "lieutenant",
				},
				Spec: synv1alpha1.TenantSpec{
					CompilePipeline: &synv1alpha1.CompilePipelineSpec{
						Enabled: true,
					},
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{},
				},
				Status: synv1alpha1.TenantStatus{
					CompilePipeline: &synv1alpha1.CompilePipelineStatus{
						Clusters: []string{"c-cluster"},
					},
				},
			}
			cluster := &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "c-cluster",
					Namespace:  "lieutenant",
					Finalizers: []string{synv1alpha1.PipelineFinalizerName},
				},
				Spec: synv1alpha1.ClusterSpec{
					TenantRef: corev1.LocalObjectReference{
						Name: "t-tenant",
					},
					EnableCompilePipeline: tc.clusterPipelineEnabled,
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoType: synv1alpha1.UnmanagedRepoType,
						AccessToken: synv1alpha1.AccessToken{
							SecretRef: "my-secret",
						},
					},
				},
			}
			preExistingVar := synv1alpha1.EnvVar{
				Name:  "ACCESS_TOKEN_c_cluster",
				Value: "whatever",
			}
			if tc.preExisting {
				tenant.Spec.GitRepoTemplate.CIVariables = []synv1alpha1.EnvVar{preExistingVar}
			}

			c := preparePipelineTestClient(t, tenant, cluster)
			r := tenantCompilePipelineReconciler(c)
			ctx := context.Background()

			_, err := r.Reconcile(ctx, requestFor(cluster))
			assert.NoError(t, err)

			modTenant := &synv1alpha1.Tenant{}
			err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, modTenant)
			assert.NoError(t, err)

			findCIVar := func(name string) (ciVar synv1alpha1.EnvVar, ok bool) {
				for _, ciVar := range modTenant.GetGitTemplate().CIVariables {
					if ciVar.Name == name {
						return ciVar, true
					}
				}
				return synv1alpha1.EnvVar{}, false
			}

			if tc.clusterPipelineEnabled && tc.preExisting {
				ciVar, ok := findCIVar("ACCESS_TOKEN_c_cluster")
				require.True(t, ok)
				assert.Equal(t, preExistingVar, ciVar, "Unmanged CIVariable should not have been changed")
			} else if tc.clusterPipelineEnabled && !tc.preExisting {
				_, ok := findCIVar("ACCESS_TOKEN_c_cluster")
				assert.False(t, ok, "Unmanged CIVariable should not have been added")
			} else if !tc.clusterPipelineEnabled && tc.preExisting {
				_, ok := findCIVar("ACCESS_TOKEN_c_cluster")
				assert.True(t, ok, "Unmanged CIVariable should not have been removed")
			}
		})
	}
}

func tenantCompilePipelineReconciler(c client.Client) *controllers.TenantCompilePipelineReconciler {
	r := controllers.TenantCompilePipelineReconciler{
		Client: c,
		Scheme: c.Scheme(),
		ApiUrl: "https://api-url",
	}
	return &r
}
