package gitrepo

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/git/manager"
	"github.com/projectsyn/lieutenant-operator/pipeline"
)

func TestSteps(t *testing.T) {

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(synv1alpha1.AddToScheme(scheme))

	tcs := map[string]struct {
		exists     bool
		exitingUrl string
		path       string
		name       string
		repoType   synv1alpha1.RepoType
		deleted    bool

		shouldCreate bool
		shouldUpdate bool
		shouldDelete bool
	}{
		"should create repo": {
			exists:     false,
			exitingUrl: "git.example.com/foo/bar",
			path:       "foo",
			name:       "bar",
			repoType:   synv1alpha1.AutoRepoType,

			shouldCreate: true,
			shouldUpdate: true,
		},
		"should update repo": {
			exists:     true,
			exitingUrl: "git.example.com/foo/bar",
			path:       "foo",
			name:       "bar",
			repoType:   synv1alpha1.AutoRepoType,

			shouldUpdate: true,
		},
		"should delete repo": {
			exists:     true,
			exitingUrl: "git.example.com/foo/bar",
			path:       "foo",
			name:       "bar",
			repoType:   synv1alpha1.AutoRepoType,
			deleted:    true,

			shouldDelete: true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo := &synv1alpha1.GitRepo{
				Spec: synv1alpha1.GitRepoSpec{
					GitRepoTemplate: synv1alpha1.GitRepoTemplate{
						Path:     tc.path,
						RepoName: tc.name,
						RepoType: tc.repoType,
					},
				},
			}
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(repo).
				Build()
			ctx := &pipeline.Context{
				Context:       context.TODO(),
				FinalizerName: "foo",
				Client:        c,
				Log:           logr.Discard(),
				Deleted:       tc.deleted,
			}
			repoURL, err := url.Parse(tc.exitingUrl)
			require.NoError(t, err)
			fr := &fakeRepo{
				exists: tc.exists,
				url:    repoURL,
			}
			res := steps(repo, ctx, fakeGitClientFactory(fr))
			assert.NoError(t, res.Err)

			assert.Equal(t, tc.shouldCreate, fr.created, "Should create repo")
			assert.Equal(t, tc.shouldUpdate, fr.updated, "Should update repo")
			assert.Equal(t, tc.shouldUpdate, fr.committed, "Should update repo content")
			assert.Equal(t, tc.shouldDelete, fr.removed, "Should delete repo")
		})
	}
}

func fakeGitClientFactory(r *fakeRepo) gitClientFactory {
	return func(ctx context.Context, instance *synv1alpha1.GitRepoTemplate, namespace string, reqLogger logr.Logger, client client.Client) (manager.Repo, string, error) {
		return r, "", nil
	}
}

type fakeRepo struct {
	url *url.URL

	exists bool

	created   bool
	updated   bool
	removed   bool
	committed bool
}

func (r fakeRepo) Type() string {
	return "fake"
}
func (r fakeRepo) FullURL() *url.URL {
	return r.url
}
func (r *fakeRepo) Create() error {
	r.created = true
	r.exists = true
	return nil
}
func (r *fakeRepo) Update() (bool, error) {
	r.updated = true
	return false, nil
}
func (r fakeRepo) Read() error {
	if !r.exists {
		return errors.New("Repos does not exist")
	}
	return nil
}
func (r *fakeRepo) Remove() error {
	r.removed = true
	return nil
}
func (r fakeRepo) Connect() error {
	return nil
}
func (r *fakeRepo) CommitTemplateFiles() error {
	r.committed = true
	return nil
}
