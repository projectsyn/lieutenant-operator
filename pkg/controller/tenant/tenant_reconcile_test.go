package tenant

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
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
	err = cl.Get(context.TODO(), types.NamespacedName{Name: tenant.Name}, updatedTenant)
	assert.NoError(t, err)
	assert.Contains(t, updatedTenant.Spec.GitRepoTemplate.TemplateFiles, "common.yml")
}

func TestCreateGitRepo(t *testing.T) {
	tests := []struct {
		name      string
		want      reconcile.Result
		wantErr   bool
		objName   string
		namespace string
	}{
		{
			name:      "Git repo object created",
			want:      reconcile.Result{},
			wantErr:   false,
			objName:   "my-git-repo",
			namespace: "tenant",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tenant := &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.objName,
					Namespace: tt.namespace,
				},
				Spec: synv1alpha1.TenantSpec{
					DisplayName: "Display Name",
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

			r := &ReconcileTenant{client: cl, scheme: s}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.objName,
					Namespace: tt.namespace,
				},
			}

			got, err := r.Reconcile(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}

			gitRepoNamespacedName := types.NamespacedName{
				Name:      tt.objName,
				Namespace: tt.namespace,
			}

			gitRepo := &synv1alpha1.GitRepo{}
			err = cl.Get(context.TODO(), gitRepoNamespacedName, gitRepo)
			assert.NoError(t, err)
			assert.Equal(t, tenant.Spec.DisplayName, gitRepo.Spec.GitRepoTemplate.DisplayName)
			fileContent, found := gitRepo.Spec.GitRepoTemplate.TemplateFiles[pipeline.CommonClassName+".yml"]
			assert.True(t, found)
			assert.Equal(t, "", fileContent)
		})
	}
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

	r := &ReconcileTenant{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: tenant.Name,
		},
	}

	_, err := r.Reconcile(req)
	require.NoError(t, err)

	updatedTenant := &synv1alpha1.Tenant{}
	err = cl.Get(context.Background(), req.NamespacedName, updatedTenant)
	require.NoError(t, err)

	assert.Contains(t, updatedTenant.Spec.GitRepoTemplate.TemplateFiles[clusterA.Name+".yml"], tenant.Name)
	assert.Contains(t, updatedTenant.Spec.GitRepoTemplate.TemplateFiles[clusterB.Name+".yml"], tenant.Name)
}
