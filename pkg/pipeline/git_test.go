package pipeline

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type repoMock struct {
	failRead bool
}

func (r *repoMock) Type() string          { return "mock" }
func (r *repoMock) FullURL() *url.URL     { return &url.URL{} }
func (r *repoMock) Create() error         { return nil }
func (r *repoMock) Update() (bool, error) { return false, nil }
func (r *repoMock) Read() error {
	if r.failRead {
		return fmt.Errorf("this should fail")
	}
	return nil
}
func (r *repoMock) Connect() error             { return nil }
func (r *repoMock) Remove() error              { return nil }
func (r *repoMock) CommitTemplateFiles() error { return nil }

func Test_repoExists(t *testing.T) {
	type args struct {
		repo manager.Repo
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "repo exists",
			want: true,
			args: args{
				repo: &repoMock{},
			},
		},
		{
			name: "repo doesn't exist",
			want: false,
			args: args{
				repo: &repoMock{
					failRead: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repoExists(tt.args.repo); got != tt.want {
				t.Errorf("repoExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleRepoError(t *testing.T) {
	type args struct {
		err      error
		instance *synv1alpha1.GitRepo
		repo     manager.Repo
		fail     bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add error",
			args: args{
				err:      fmt.Errorf("lol nope"),
				instance: &synv1alpha1.GitRepo{},
				repo:     &repoMock{},
			},
		},
		{
			name: "add error failure",
			args: args{
				err:      fmt.Errorf("lol nope"),
				instance: &synv1alpha1.GitRepo{},
				repo:     &repoMock{},
				fail:     true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var client client.Client

			if tt.args.fail {
				client, _ = testSetupClient([]runtime.Object{})
			} else {
				client, _ = testSetupClient([]runtime.Object{tt.args.instance})
			}

			err := handleRepoError(tt.args.err, tt.args.instance, tt.args.repo, client)
			assert.Error(t, err)
			failedPhase := synv1alpha1.Failed
			assert.Equal(t, &failedPhase, tt.args.instance.Status.Phase)

			if tt.args.fail {
				assert.Contains(t, err.Error(), "could not set status")
			}

		})
	}
}

func Test_setGitRepoURLAndHostKeys(t *testing.T) {
	type args struct {
		obj     *synv1alpha1.Cluster
		gitRepo *synv1alpha1.GitRepo
		data    *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "set url and keys",
			wantErr: false,
			args: args{
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
				data: &ExecutionContext{},
				gitRepo: &synv1alpha1.GitRepo{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Status: synv1alpha1.GitRepoStatus{
						URL:      "someURL",
						HostKeys: "someKeys",
					},
				},
			},
		},
		{
			name:    "set url and keys not found",
			wantErr: false,
			args: args{
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "invalid",
					},
				},
				data:    &ExecutionContext{},
				gitRepo: &synv1alpha1.GitRepo{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.obj,
				tt.args.gitRepo,
			})

			if got := setGitRepoURLAndHostKeys(tt.args.obj, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("setGitRepoURLAndHostKeys() = had error: %v", got.Err)
			}

			assert.Equal(t, tt.args.gitRepo.Status.URL, tt.args.obj.Spec.GitRepoURL)
			assert.Equal(t, tt.args.gitRepo.Status.HostKeys, tt.args.obj.Spec.GitHostKeys)
		})
	}
}
