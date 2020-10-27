package gitrepo

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var savedGitRepoOpt manager.RepoOptions

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) (client.Client, *runtime.Scheme) {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClientWithScheme(s, objs...), s
}

func TestReconcileGitRepo_Reconcile(t *testing.T) {
	type fields struct {
		name        string
		namespace   string
		displayName string
		tenantName  string
		secretName  string
		URL         string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "test gitrepos not found",
			fields: fields{
				name:       "testrep",
				namespace:  "testnamespace",
				secretName: "testsecret",
				URL:        "some.example.com",
			},
			wantErr: true,
		},
		{
			name: "repo exists",
			fields: fields{
				name:        "anothertest",
				namespace:   "namespace",
				displayName: "This is the name",
				tenantName:  "sometenant",
				secretName:  "testsecret",
				URL:         "git.corp.net",
			},
		},
	}

	testRepo := &testRepoImplementation{}
	manager.Register(testRepo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &synv1alpha1.GitRepo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.fields.name,
					Namespace: tt.fields.namespace,
				},
				Spec: synv1alpha1.GitRepoSpec{
					GitRepoTemplate: synv1alpha1.GitRepoTemplate{
						APISecretRef: corev1.SecretReference{
							Name:      tt.fields.secretName,
							Namespace: tt.fields.namespace,
						},
						DeployKeys:  nil,
						Path:        tt.fields.namespace,
						RepoName:    tt.fields.name,
						DisplayName: tt.fields.displayName,
					},
					TenantRef: corev1.LocalObjectReference{
						Name: tt.fields.tenantName,
					},
				},
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.fields.secretName,
					Namespace: tt.fields.namespace,
				},
				Data: map[string][]byte{
					manager.SecretEndpointName: []byte(tt.fields.URL),
					manager.SecretTokenName:    []byte("secret"),
					manager.SecretHostKeysName: []byte("somekey"),
				},
			}

			objs := []runtime.Object{
				repo,
				&synv1alpha1.Tenant{},
				&synv1alpha1.Cluster{},
			}

			cl, s := testSetupClient(objs)

			assert.NoError(t, cl.Create(context.TODO(), secret))

			r := &ReconcileGitRepo{client: cl, scheme: s}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.fields.name,
					Namespace: tt.fields.namespace,
				},
			}

			_, err := r.Reconcile(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gitRepo := &synv1alpha1.GitRepo{}
			err = cl.Get(context.TODO(), req.NamespacedName, gitRepo)
			assert.NoError(t, err)
			if !tt.wantErr {
				assert.Equal(t, string(secret.Data[manager.SecretHostKeysName]), gitRepo.Status.HostKeys)
				assert.Equal(t, synv1alpha1.DefaultRepoType, gitRepo.Spec.RepoType)
				assert.Equal(t, tt.fields.displayName, gitRepo.Spec.DisplayName)
				assert.Equal(t, tt.fields.displayName, savedGitRepoOpt.DisplayName)
				assert.Equal(t, tt.fields.tenantName, gitRepo.Labels[apis.LabelNameTenant])
				assert.Equal(t, synv1alpha1.Created, *gitRepo.Status.Phase)
				assert.Equal(t, tt.fields.URL+"/"+tt.fields.namespace+"/"+tt.fields.name, gitRepo.Status.URL)
				assert.Equal(t, synv1alpha1.GitType("test"), gitRepo.Status.Type)
			}
		})
	}
}

type testRepoImplementation struct {
	options manager.RepoOptions
}

func (t testRepoImplementation) IsType(URL *url.URL) (bool, error) {
	return strings.HasPrefix(URL.String(), "git"), nil
}

func (t testRepoImplementation) New(options manager.RepoOptions) (manager.Repo, error) {
	t.options = options
	savedGitRepoOpt = options
	return t, nil
}

func (t testRepoImplementation) RemoveFile(path string) error {
	return nil
}

func (t testRepoImplementation) Remove() error {
	return nil
}

func (t testRepoImplementation) Type() string {
	return "test"
}

func (t testRepoImplementation) FullURL() *url.URL {
	return t.options.URL
}

func (t testRepoImplementation) Create() error {
	return nil
}

func (t testRepoImplementation) Update() (bool, error) {
	return true, nil
}

func (t testRepoImplementation) Read() error {
	return nil
}

func (t testRepoImplementation) Delete() error {
	return nil
}

func (t testRepoImplementation) Connect() error {
	return nil
}

func (t testRepoImplementation) CommitTemplateFiles() error {
	return nil
}
