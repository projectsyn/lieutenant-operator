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
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		exists  bool
		repoUrl string

		path      string
		name      string
		repoType  synv1alpha1.RepoType
		deleted   bool
		adopt     bool
		statusURL string

		shouldError bool

		shouldCreate     bool
		shouldUpdate     bool
		shouldDelete     bool
		updatedStatusURL string
	}{
		"should create repo": {
			exists:  false,
			repoUrl: "git.example.com/foo/bar",

			path:     "foo",
			name:     "bar",
			repoType: synv1alpha1.AutoRepoType,

			shouldCreate:     true,
			shouldUpdate:     true,
			updatedStatusURL: "git.example.com/foo/bar",
		},
		"should update repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",

			path:      "foo",
			name:      "bar",
			repoType:  synv1alpha1.AutoRepoType,
			statusURL: "git.example.com/foo/bar",

			shouldUpdate:     true,
			updatedStatusURL: "git.example.com/foo/bar",
		},
		"should delete repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",

			path:      "foo",
			name:      "bar",
			repoType:  synv1alpha1.AutoRepoType,
			statusURL: "git.example.com/foo/bar",
			deleted:   true,

			shouldDelete:     true,
			updatedStatusURL: "git.example.com/foo/bar",
		},
		"should not create unmanaged repo": {
			exists:  false,
			repoUrl: "git.example.com/foo/bar",

			path:     "foo",
			name:     "bar",
			repoType: synv1alpha1.UnmanagedRepoType,
		},
		"should not update unmanaged repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",

			path:      "foo",
			name:      "bar",
			repoType:  synv1alpha1.UnmanagedRepoType,
			statusURL: "git.example.com/foo/bar",

			updatedStatusURL: "git.example.com/foo/bar",
		},
		"should not delete unmanaged repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",

			path:     "foo",
			name:     "bar",
			repoType: synv1alpha1.UnmanagedRepoType,
			deleted:  true,
		},
		"should not adopt repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",

			shouldError: true,

			path:     "foo",
			name:     "bar",
			repoType: synv1alpha1.AutoRepoType,
		},
		"should adopt repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",
			adopt:   true,

			path:     "foo",
			name:     "bar",
			repoType: synv1alpha1.AutoRepoType,

			shouldUpdate:     true,
			updatedStatusURL: "git.example.com/foo/bar",
		},
		"should not delete unadopted repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",

			path:     "foo",
			name:     "bar",
			repoType: synv1alpha1.AutoRepoType,
			deleted:  true,
		},
		"should adopt and delete repo": {
			exists:  true,
			repoUrl: "git.example.com/foo/bar",
			adopt:   true,

			path:     "foo",
			name:     "bar",
			repoType: synv1alpha1.AutoRepoType,
			deleted:  true,

			shouldDelete: true,
		},
		"should create other repo": {
			exists:  false,
			repoUrl: "git.example.com/foo/bar",

			path:      "foo",
			name:      "bar",
			repoType:  synv1alpha1.AutoRepoType,
			statusURL: "git.example.com/foo/buzz",

			shouldCreate:     true,
			shouldUpdate:     true,
			updatedStatusURL: "git.example.com/foo/bar",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo := &synv1alpha1.GitRepo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c-bar",
					Namespace: "foo",
				},
				Spec: synv1alpha1.GitRepoSpec{
					GitRepoTemplate: synv1alpha1.GitRepoTemplate{
						Path:           tc.path,
						RepoName:       tc.name,
						RepoType:       tc.repoType,
						CreationPolicy: synv1alpha1.CreatePolicy,
					},
				},
				Status: synv1alpha1.GitRepoStatus{
					URL: tc.statusURL,
				},
			}
			if tc.adopt {
				repo.Spec.GitRepoTemplate.CreationPolicy = synv1alpha1.AdoptPolicy
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
			repoURL, err := url.Parse(tc.repoUrl)
			require.NoError(t, err)
			fr := &fakeRepo{
				exists: tc.exists,
				url:    repoURL,
			}
			res := steps(repo, ctx, fakeGitClientFactory(fr))
			if tc.shouldError {
				assert.Error(t, res.Err)
			} else {
				assert.NoError(t, res.Err)
			}

			assert.Equal(t, tc.shouldCreate, fr.created, "Should create repo")
			assert.Equal(t, tc.shouldUpdate, fr.updated, "Should update repo")
			assert.Equal(t, tc.shouldUpdate, fr.committed, "Should update repo content")
			assert.Equal(t, tc.shouldDelete, fr.removed, "Should delete repo")

			assert.Equal(t, tc.updatedStatusURL, repo.Status.URL)
		})
	}
}

func TestStepsCreationFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(synv1alpha1.AddToScheme(scheme))

	tcs := map[string]struct {
		failCreation bool
		failUpdate   bool
		failCommit   bool

		urlSet bool
	}{
		"handle creation failure": {
			failCreation: true,
		},
		"handle update failure": {
			failUpdate: true,
			urlSet:     true,
		},
		"handle commit failure": {
			failCommit: true,
			urlSet:     true,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			repo := &synv1alpha1.GitRepo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c-bar",
					Namespace: "foo",
				},
				Spec: synv1alpha1.GitRepoSpec{
					GitRepoTemplate: synv1alpha1.GitRepoTemplate{
						Path:     "foo",
						RepoName: "bar",
						RepoType: synv1alpha1.AutoRepoType,
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
			}
			repoURL, err := url.Parse("git.example.com/foo/bar")
			require.NoError(t, err)
			fr := &fakeRepo{
				url:          repoURL,
				failCreation: tc.failCreation,
				failUpdate:   tc.failUpdate,
				failCommit:   tc.failCommit,
			}
			res := steps(repo, ctx, fakeGitClientFactory(fr))
			assert.Error(t, res.Err)
			found := &synv1alpha1.GitRepo{}
			assert.NoError(t, c.Get(ctx.Context, types.NamespacedName{
				Namespace: repo.Namespace,
				Name:      repo.Name,
			}, found))

			if tc.urlSet {
				assert.Equal(t, "git.example.com/foo/bar", found.Status.URL)
			} else {
				assert.Equal(t, "", found.Status.URL)
			}
			assert.Equal(t, synv1alpha1.Failed, *found.Status.Phase)
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

	failCreation bool
	failUpdate   bool
	failCommit   bool
}

func (r fakeRepo) Type() string {
	return "fake"
}
func (r fakeRepo) FullURL() *url.URL {
	return r.url
}
func (r *fakeRepo) Create() error {
	if r.failCreation {
		return errors.New("cannot create repo")
	}
	r.created = true
	r.exists = true
	return nil
}
func (r *fakeRepo) Update() (bool, error) {
	if r.failUpdate {
		return false, errors.New("cannot update repo")
	}
	r.updated = true
	return true, nil
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
	if r.failCommit {
		return errors.New("cannot commit files")
	}
	r.committed = true
	return nil
}
