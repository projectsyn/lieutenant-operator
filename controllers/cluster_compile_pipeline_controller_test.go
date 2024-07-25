package controllers_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_AddClusterToPipelineStatus(t *testing.T) {
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

func Test_RemoveClusterFromPipelineStatus(t *testing.T) {
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

func Test_RemoveClusterFromPipelineStatus_WhenDeleting(t *testing.T) {
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

func Test_FinalizerAdded(t *testing.T) {
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

func Test_RemoveClusterFromPipelineStatus_EnableUnset(t *testing.T) {
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

func Test_NoChangeIfClusterInList(t *testing.T) {
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

func Test_LeaveOtherListEntriesBe_WhenRemoving(t *testing.T) {
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

func Test_LeaveOtherListEntriesBe_WhenAdding(t *testing.T) {
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
func Test_CiVariableNotUpdated_IfNotEnabled(t *testing.T) {
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

func Test_CiVariableNotUpdated_IfNoToken(t *testing.T) {
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

func Test_CiVariableUpdated_IfEnabled(t *testing.T) {
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
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].GitlabOptions.Protected, true)
	assert.Equal(t, mod_tenant.Spec.GitRepoTemplate.CIVariables[0].GitlabOptions.Raw, true)
}
func Test_KeepListInAlphabeticalOrder(t *testing.T) {
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
