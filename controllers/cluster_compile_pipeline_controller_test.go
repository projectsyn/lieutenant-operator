package controllers_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers"
)

func Test_ClusterCompilePipelineReconciler_AddClusterToPipelineStatus(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
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
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Contains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-cluster")
}

func Test_ClusterCompilePipelineReconciler_RemoveClusterFromPipelineStatus(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
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
			EnableCompilePipeline: false,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.NotContains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-cluster")
}

func Test_ClusterCompilePipelineReconciler_RemoveClusterFromPipelineStatus_WhenDeleting(t *testing.T) {
	now := metav1.NewTime(time.Now())
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster"},
			},
		},
	}
	cluster := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "c-cluster",
			Namespace:         "lieutenant",
			DeletionTimestamp: &now,
			Finalizers:        []string{synv1alpha1.PipelineFinalizerName},
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.NotContains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-cluster")
	assert.NotContains(t, mod_tenant.Finalizers, synv1alpha1.PipelineFinalizerName)
}

func Test_ClusterCompilePipelineReconciler_FinalizerAdded(t *testing.T) {
	cluster := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "c-cluster",
			Namespace: "lieutenant",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: false,
		},
	}
	c := preparePipelineTestClient(t, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_cluster := &synv1alpha1.Cluster{}
	err = c.Get(ctx, types.NamespacedName{Name: "c-cluster", Namespace: "lieutenant"}, mod_cluster)
	assert.NoError(t, err)

	assert.Contains(t, mod_cluster.Finalizers, synv1alpha1.PipelineFinalizerName)
}

func Test_ClusterCompilePipelineReconciler_RemoveClusterFromPipelineStatus_EnableUnset(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
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
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.NotContains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-cluster")
}

func Test_ClusterCompilePipelineReconciler_NoChangeIfClusterInList(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
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
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Contains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-cluster")
}

func Test_ClusterCompilePipelineReconciler_LeaveOtherListEntriesBe_WhenRemoving(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-cluster", "c-other-cluster"},
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
			EnableCompilePipeline: false,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.NotContains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-cluster")
	assert.Contains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-other-cluster")
}

func Test_ClusterCompilePipelineReconciler_LeaveOtherListEntriesBe_WhenAdding(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-other-cluster"},
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
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Contains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-cluster")
	assert.Contains(t, mod_tenant.Status.CompilePipeline.Clusters, "c-other-cluster")
}
func Test_ClusterCompilePipelineReconciler_CiVariableNotUpdated_IfNotEnabled(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
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
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				AccessToken: synv1alpha1.AccessToken{
					SecretRef: "my-secret",
				},
			},
			TenantRef: corev1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.Empty(t, mod_tenant.GetGitTemplate().CIVariables)
}

func Test_ClusterCompilePipelineReconciler_CiVariableNotUpdated_IfNoToken(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
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
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.Empty(t, mod_tenant.GetGitTemplate().CIVariables)
}

func Test_ClusterCompilePipelineReconciler_CiVariableUpdated_IfEnabled(t *testing.T) {
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
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				AccessToken: synv1alpha1.AccessToken{
					SecretRef: "my-secret",
				},
			},
			TenantRef: corev1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, cluster)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(cluster))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Spec.GitRepoTemplate)
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].Name, "ACCESS_TOKEN_c_cluster")
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].ValueFrom.SecretKeyRef.Name, "my-secret")
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].ValueFrom.SecretKeyRef.Key, "token")
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].GitlabOptions.Masked, true)
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].GitlabOptions.Protected, false)
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].GitlabOptions.Raw, true)
}

func Test_ClusterCompilePipelineReconciler_KeepListInAlphabeticalOrder(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-tenant",
			Namespace: "lieutenant",
		},
		Status: synv1alpha1.TenantStatus{
			CompilePipeline: &synv1alpha1.CompilePipelineStatus{
				Clusters: []string{"c-b", "c-d"},
			},
		},
	}
	clusterA := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "c-a",
			Namespace:  "lieutenant",
			Finalizers: []string{synv1alpha1.PipelineFinalizerName},
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
		},
	}
	clusterB := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "c-b",
			Namespace:  "lieutenant",
			Finalizers: []string{synv1alpha1.PipelineFinalizerName},
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: false,
		},
	}
	clusterC := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "c-c",
			Namespace:  "lieutenant",
			Finalizers: []string{synv1alpha1.PipelineFinalizerName},
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: "t-tenant",
			},
			EnableCompilePipeline: true,
		},
	}
	c := preparePipelineTestClient(t, tenant, clusterA, clusterB, clusterC)
	r := clusterCompilePipelineReconciler(c)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, requestFor(clusterA))
	assert.NoError(t, err)

	mod_tenant := &synv1alpha1.Tenant{}
	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[0], "c-a")
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[1], "c-b")
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[2], "c-d")

	_, err = r.Reconcile(ctx, requestFor(clusterB))
	assert.NoError(t, err)

	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[0], "c-a")
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[1], "c-d")

	_, err = r.Reconcile(ctx, requestFor(clusterC))
	assert.NoError(t, err)

	err = c.Get(ctx, types.NamespacedName{Name: "t-tenant", Namespace: "lieutenant"}, mod_tenant)
	assert.NoError(t, err)

	assert.NotNil(t, mod_tenant.Status.CompilePipeline)
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[0], "c-a")
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[1], "c-c")
	assert.Equal(t, mod_tenant.Status.CompilePipeline.Clusters[2], "c-d")
}

func clusterCompilePipelineReconciler(c client.Client) *controllers.ClusterCompilePipelineReconciler {
	r := controllers.ClusterCompilePipelineReconciler{
		Client: c,
		Scheme: c.Scheme(),
	}
	return &r
}

func Test_ClusterCompilePipelineReconciler_CIVariable_IgnoreUnmanagedRepo(t *testing.T) {
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
			r := clusterCompilePipelineReconciler(c)
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
