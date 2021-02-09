package tenant

import (
	"context"
	"testing"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func testSetupClient(objs []runtime.Object) (client.Client, *runtime.Scheme) {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClientWithScheme(s, objs...), s
}

func fetchObject(t *testing.T, c client.Client, ns types.NamespacedName, obj runtime.Object) {
	err := c.Get(context.Background(), ns, obj)
	require.NoError(t, err)
}

func reconcileTenant(t *testing.T, c client.Client, s *runtime.Scheme, name types.NamespacedName) reconcile.Result {
	r := &ReconcileTenant{client: c, scheme: s}

	req := reconcile.Request{
		NamespacedName: name,
	}

	result, err := r.Reconcile(req)
	require.NoError(t, err)

	return result
}

func TestHandleNilGitRepoTemplate(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: "t-some-name",
		},
		Spec: synv1alpha1.TenantSpec{
			DisplayName: "Display Name",
		},
	}

	cl, s := testSetupClient([]runtime.Object{
		tenant,
		&synv1alpha1.ClusterList{},
		&synv1alpha1.GitRepo{},
	})

	r := &ReconcileTenant{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: tenant.Name,
		},
	}

	_, err := r.Reconcile(req)
	assert.NoError(t, err)
	updatedTenant := &synv1alpha1.Tenant{}
	fetchObject(t, cl, req.NamespacedName, updatedTenant)
	assert.Contains(t, updatedTenant.Spec.GitRepoTemplate.TemplateFiles, "common.yml")
}

func TestCreateGitRepo(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tenant",
			Namespace: "my-namespace",
		},
		Spec: synv1alpha1.TenantSpec{
			DisplayName: "My Tenant",
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				RepoName: "repo-name",
			},
		},
	}

	objs := []runtime.Object{
		tenant,
		&synv1alpha1.GitRepo{},
		&synv1alpha1.ClusterList{},
	}

	cl, s := testSetupClient(objs)

	name := types.NamespacedName{
		Name:      tenant.Name,
		Namespace: tenant.Namespace,
	}

	reconcileTenant(t, cl, s, name)

	gitRepo := &synv1alpha1.GitRepo{}
	fetchObject(t, cl, name, gitRepo)

	assert.Equal(t, tenant.Spec.DisplayName, gitRepo.Spec.GitRepoTemplate.DisplayName)
	fileContent, found := gitRepo.Spec.GitRepoTemplate.TemplateFiles[CommonClassName+".yml"]
	assert.True(t, found)
	assert.Equal(t, "", fileContent)
}

func TestCreateClusterClass(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tenant-a",
		},
		Spec: synv1alpha1.TenantSpec{
			DisplayName: "Display Name",
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				RepoName: "repo-name",
			},
		},
	}
	clusterA := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-a",
			Labels: map[string]string{
				apis.LabelNameTenant: tenant.Name,
			},
		},
		Spec: synv1alpha1.ClusterSpec{},
	}
	clusterB := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-b",
			Labels: map[string]string{
				apis.LabelNameTenant: tenant.Name,
			},
		},
		Spec: synv1alpha1.ClusterSpec{},
	}

	objs := []runtime.Object{
		tenant,
		clusterA,
		clusterB,
		&synv1alpha1.GitRepo{},
		&synv1alpha1.ClusterList{},
	}

	cl, s := testSetupClient(objs)

	name := types.NamespacedName{
		Name: tenant.Name,
	}

	reconcileTenant(t, cl, s, name)

	updatedTenant := &synv1alpha1.Tenant{}
	fetchObject(t, cl, name, updatedTenant)

	assert.Contains(t, updatedTenant.Spec.GitRepoTemplate.TemplateFiles[clusterA.Name+".yml"], tenant.Name)
	assert.Contains(t, updatedTenant.Spec.GitRepoTemplate.TemplateFiles[clusterB.Name+".yml"], tenant.Name)
}
