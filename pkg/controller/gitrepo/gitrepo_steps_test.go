package gitrepo

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type gitRepoArgs struct {
	repo        *synv1alpha1.GitRepo
	cluster     *synv1alpha1.Cluster
	template    *synv1alpha1.GitRepoTemplate
	tenantRef   corev1.LocalObjectReference
	templateObj pipeline.Object
	data        *pipeline.Context
}

var createOrUpdateGitRepoCases = map[string]struct {
	args    gitRepoArgs
	wantErr bool
}{
	"create git repo": {
		args: gitRepoArgs{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			},
			template: &synv1alpha1.GitRepoTemplate{
				APISecretRef: corev1.SecretReference{Name: "testSecret"},
				DeployKeys:   nil,
				Path:         "testPath",
				RepoName:     "testRepo",
			},
			tenantRef: corev1.LocalObjectReference{
				Name: "testTenant",
			},
		},
	},
	"empty template": {
		args: gitRepoArgs{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			},
			template: nil,
			tenantRef: corev1.LocalObjectReference{
				Name: "testTenant",
			},
		},
	},
}

func TestCreateOrUpdateGitRepo(t *testing.T) {
	for name, tt := range createOrUpdateGitRepoCases {
		objs := []runtime.Object{
			&synv1alpha1.GitRepo{},
		}

		cl, _ := testSetupClient(objs)

		tt.args.cluster.Spec.GitRepoTemplate = tt.args.template
		tt.args.cluster.Spec.TenantRef = tt.args.tenantRef

		t.Run(name, func(t *testing.T) {
			if res := CreateGitRepo(tt.args.cluster, &pipeline.Context{Client: cl}); (res.Err != nil) != tt.wantErr {
				t.Errorf("CreateGitRepo() error = %v, wantErr %v", res.Err, tt.wantErr)
			}

			if tt.args.template != nil {
				namespacedName := types.NamespacedName{
					Name:      tt.args.cluster.GetName(),
					Namespace: tt.args.cluster.GetNamespace(),
				}

				checkRepo := &synv1alpha1.GitRepo{}
				assert.NoError(t, cl.Get(context.TODO(), namespacedName, checkRepo))
				assert.Equal(t, tt.args.template, &checkRepo.Spec.GitRepoTemplate)
			}
		})
	}
}

var fetchGitRepoTemplateCases = map[string]struct {
	args    gitRepoArgs
	wantErr bool
}{
	"fetch tenant changes": {
		args: gitRepoArgs{
			data: &pipeline.Context{},
			repo: &synv1alpha1.GitRepo{
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
	"fetch cluster changes": {
		args: gitRepoArgs{
			data: &pipeline.Context{},
			repo: &synv1alpha1.GitRepo{
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

func Test_fetchGitRepoTemplate(t *testing.T) {
	for name, tt := range fetchGitRepoTemplateCases {
		t.Run(name, func(t *testing.T) {
			rtObj := tt.args.templateObj.(runtime.Object)

			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.repo,
				&synv1alpha1.Tenant{},
				&synv1alpha1.Cluster{},
				rtObj,
			})

			if err := fetchGitRepoTemplate(tt.args.repo, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("fetchGitRepoTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.args.templateObj.GetGitTemplate(), &tt.args.repo.Spec.GitRepoTemplate)
			tt.args.templateObj.GetGitTemplate().RepoName = "another test"
			rtObj = tt.args.templateObj.(runtime.Object)
			assert.NoError(t, tt.args.data.Client.Update(context.TODO(), rtObj))
			assert.NoError(t, fetchGitRepoTemplate(tt.args.repo, tt.args.data))
			assert.Equal(t, tt.args.templateObj.GetGitTemplate(), &tt.args.repo.Spec.GitRepoTemplate)
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

var repoExistsCases = map[string]struct {
	args manager.Repo
	want bool
}{
	"repo exists": {
		want: true,
		args: &repoMock{},
	},
	"repo doesn't exist": {
		want: false,
		args: &repoMock{
			failRead: true,
		},
	}}

func Test_repoExists(t *testing.T) {
	for name, tt := range repoExistsCases {
		t.Run(name, func(t *testing.T) {
			if got := repoExists(tt.args); got != tt.want {
				t.Errorf("repoExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

type handleRepoErrorArgs struct {
	err      error
	instance *synv1alpha1.GitRepo
	repo     manager.Repo
	fail     bool
}

var handleRepoErrorCases = map[string]struct {
	args handleRepoErrorArgs
}{
	"add error": {
		args: handleRepoErrorArgs{
			err:      fmt.Errorf("lol nope"),
			instance: &synv1alpha1.GitRepo{},
			repo:     &repoMock{},
		},
	},
	"add error failure": {
		args: handleRepoErrorArgs{
			err:      fmt.Errorf("lol nope"),
			instance: &synv1alpha1.GitRepo{},
			repo:     &repoMock{},
			fail:     true,
		},
	},
}

func Test_handleRepoError(t *testing.T) {
	for name, tt := range handleRepoErrorCases {
		t.Run(name, func(t *testing.T) {

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

var setGitRepoURLAndHostKeysCases = map[string]struct {
	args    gitRepoArgs
	wantErr bool
}{
	"set url and keys": {
		wantErr: false,
		args: gitRepoArgs{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			data: &pipeline.Context{},
			repo: &synv1alpha1.GitRepo{
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
	"set url and keys not found": {
		wantErr: false,
		args: gitRepoArgs{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid",
				},
			},
			data: &pipeline.Context{},
			repo: &synv1alpha1.GitRepo{},
		},
	},
}

func Test_setGitRepoURLAndHostKeys(t *testing.T) {
	for name, tt := range setGitRepoURLAndHostKeysCases {
		t.Run(name, func(t *testing.T) {
			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.cluster,
				tt.args.repo,
			})

			if got := SetGitRepoURLAndHostKeys(tt.args.cluster, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("SetGitRepoURLAndHostKeys() = had error: %v", got.Err)
			}

			assert.Equal(t, tt.args.repo.Status.URL, tt.args.cluster.Spec.GitRepoURL)
			assert.Equal(t, tt.args.repo.Status.HostKeys, tt.args.cluster.Spec.GitHostKeys)
		})
	}
}
