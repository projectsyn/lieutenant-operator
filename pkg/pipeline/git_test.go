package pipeline

import (
	"context"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) client.Client {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClientWithScheme(s, objs...)
}

func TestCreateOrUpdateGitRepo(t *testing.T) {
	type args struct {
		obj       *synv1alpha1.Cluster
		template  *synv1alpha1.GitRepoTemplate
		tenantRef v1.LocalObjectReference
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "create git repo",
			args: args{
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
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
		{
			name: "empty template",
			args: args{
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
				template: nil,
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

		tt.args.obj.Spec.GitRepoTemplate = tt.args.template
		tt.args.obj.Spec.TenantRef = tt.args.tenantRef

		t.Run(tt.name, func(t *testing.T) {
			if res := createGitRepo(tt.args.obj, &ExecutionContext{Client: cl}); (res.Err != nil) != tt.wantErr {
				t.Errorf("CreateGitRepo() error = %v, wantErr %v", res.Err, tt.wantErr)
			}
		})

		if tt.args.template == nil {
			continue
		}

		namespacedName := types.NamespacedName{
			Name:      tt.args.obj.GetName(),
			Namespace: tt.args.obj.GetNamespace(),
		}

		checkRepo := &synv1alpha1.GitRepo{}
		assert.NoError(t, cl.Get(context.TODO(), namespacedName, checkRepo))
		assert.Equal(t, tt.args.template, &checkRepo.Spec.GitRepoTemplate)

	}
}
