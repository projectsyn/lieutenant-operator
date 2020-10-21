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
)

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

		cl, _ := testSetupClient(objs)

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

func Test_fetchGitRepoTemplate(t *testing.T) {
	type args struct {
		obj         *synv1alpha1.GitRepo
		templateObj PipelineObject
		data        *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "fetch tenant changes",
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.GitRepo{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
				templateObj: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: synv1alpha1.TenantSpec{
						GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
							RepoName: "Test Repo",
							RepoType: synv1alpha1.AutoRepoType,
							Path:     "test",
						},
					},
				},
			},
		},
		{
			name: "fetch cluster changes",
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.GitRepo{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
				templateObj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: synv1alpha1.ClusterSpec{
						GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
							RepoName: "Test Repo",
							RepoType: synv1alpha1.AutoRepoType,
							Path:     "test",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			rtObj := tt.args.templateObj.(runtime.Object)

			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.obj,
				&synv1alpha1.Tenant{},
				&synv1alpha1.Cluster{},
				rtObj,
			})

			if err := fetchGitRepoTemplate(tt.args.obj, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("fetchGitRepoTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.args.templateObj.GetGitTemplate(), &tt.args.obj.Spec.GitRepoTemplate)
			tt.args.templateObj.GetGitTemplate().RepoName = "another test"
			rtObj = tt.args.templateObj.(runtime.Object)
			assert.NoError(t, tt.args.data.Client.Update(context.TODO(), rtObj))
			assert.NoError(t, fetchGitRepoTemplate(tt.args.obj, tt.args.data))
			assert.Equal(t, tt.args.templateObj.GetGitTemplate(), &tt.args.obj.Spec.GitRepoTemplate)

		})
	}
}
