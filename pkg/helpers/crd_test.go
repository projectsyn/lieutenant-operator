package helpers

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestAddTenantLabel(t *testing.T) {
	type args struct {
		meta   *metav1.ObjectMeta
		tenant string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add labels",
			args: args{
				meta:   &metav1.ObjectMeta{},
				tenant: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddTenantLabel(tt.args.meta, tt.args.tenant)

			if tt.args.meta.Labels[apis.LabelNameTenant] != tt.args.tenant {
				t.Error("labels do not match")
			}

		})
	}
}

func TestCreateOrUpdateGitRepo(t *testing.T) {
	type args struct {
		obj       metav1.Object
		gvk       schema.GroupVersionKind
		template  *synv1alpha1.GitRepoTemplate
		tenantRef v1.LocalObjectReference
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "create and update git repo",
			args: args{
				obj: &synv1alpha1.GitRepo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
				gvk: schema.GroupVersionKind{
					Version: "testVersion",
					Kind:    "testKind",
				},
				template: &synv1alpha1.GitRepoTemplate{
					APISecretRef: v1.SecretReference{Name: "testSecret"},
					DeployKeys:   nil,
					Path:         "testPath",
					RepoName:     "testRepo",
				},
				tenantRef: v1.LocalObjectReference{
					Name: "testTenant",
				},
			},
		},
	}
	for _, tt := range tests {

		objs := []runtime.Object{
			&synv1alpha1.GitRepo{},
		}

		cl := testSetupClient(objs)

		t.Run(tt.name, func(t *testing.T) {
			if err := CreateOrUpdateGitRepo(tt.args.obj, tt.args.gvk, tt.args.template, cl, tt.args.tenantRef); (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateGitRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})

		namespacedName := types.NamespacedName{
			Name:      tt.args.obj.GetName(),
			Namespace: tt.args.obj.GetNamespace(),
		}

		checkRepo := &synv1alpha1.GitRepo{}
		assert.NoError(t, cl.Get(context.TODO(), namespacedName, checkRepo))
		assert.Equal(t, tt.args.template, &checkRepo.Spec.GitRepoTemplate)
		tt.args.template.RepoName = "changedName"
		assert.NoError(t, CreateOrUpdateGitRepo(tt.args.obj, tt.args.gvk, tt.args.template, cl, tt.args.tenantRef))
		assert.NoError(t, cl.Get(context.TODO(), namespacedName, checkRepo))
		assert.Equal(t, tt.args.template, &checkRepo.Spec.GitRepoTemplate)

		checkRepo.Spec.RepoType = synv1alpha1.AutoRepoType
		assert.NoError(t, cl.Update(context.TODO(), checkRepo))
		assert.NoError(t, CreateOrUpdateGitRepo(tt.args.obj, tt.args.gvk, tt.args.template, cl, tt.args.tenantRef))
		finalRepo := &synv1alpha1.GitRepo{}
		assert.NoError(t, cl.Get(context.TODO(), namespacedName, finalRepo))
		assert.Equal(t, checkRepo.Spec.GitRepoTemplate, finalRepo.Spec.GitRepoTemplate)
	}
}

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) client.Client {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClient(objs...)
}
