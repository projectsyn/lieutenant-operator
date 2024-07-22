package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
			Name:      "c-cluster",
			Namespace: "lieutenant",
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
			Name:      "c-cluster",
			Namespace: "lieutenant",
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
			Name:      "c-cluster",
			Namespace: "lieutenant",
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
			Name:      "c-cluster",
			Namespace: "lieutenant",
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

func preparePipelineTestClient(t *testing.T, initObjs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(synv1alpha1.AddToScheme(scheme))

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjs...).
		WithStatusSubresource(&synv1alpha1.Tenant{}).
		Build()

	return client
}

func requestFor(obj client.Object) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func clusterCompilePipelineReconciler(c client.Client) *ClusterCompilePipelineReconciler {
	r := ClusterCompilePipelineReconciler{
		Client: c,
		Scheme: c.Scheme(),
	}
	return &r
}
