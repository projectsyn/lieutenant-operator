package gitrepo

import (
	"context"
	"net/url"
	"strings"
	"testing"

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

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) (client.Client, *runtime.Scheme) {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClient(objs...), s
}

func TestReconcileGitRepo_Reconcile(t *testing.T) {
	type fields struct {
		name       string
		namespace  string
		tenantName string
		secretName string
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
			},
			wantErr: true,
		},
		{
			name: "repo exists",
			fields: fields{
				name:       "anothertest",
				namespace:  "namespace",
				tenantName: "sometenant",
				secretName: "testsecret",
			},
		},
	}

	manager.Register(&testRepoImplementation{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			repo := &synv1alpha1.GitRepo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.fields.name,
					Namespace: tt.fields.namespace,
				},
				Spec: synv1alpha1.GitRepoSpec{
					GitRepoTemplate: synv1alpha1.GitRepoTemplate{
						APISecretRef: corev1.SecretReference{},
						DeployKeys:   nil,
						Path:         tt.fields.namespace,
						RepoName:     tt.fields.name,
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
					SecretEndpointName: []byte("something"),
					SecretTokenName:    []byte("secret"),
					SecretHostKeysName: []byte("somekey"),
				},
			}

			objs := []runtime.Object{
				repo,
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
			if !tt.wantErr {
				gitRepo := &synv1alpha1.GitRepo{}
				err = cl.Get(context.TODO(), req.NamespacedName, gitRepo)
				assert.NoError(t, err)
				assert.Equal(t, string(secret.Data[SecretHostKeysName]), gitRepo.Status.HostKeys)
			}
		})
	}
}

type testRepoImplementation struct {
	//meh
}

func (t testRepoImplementation) IsType(URL string) (bool, error) {
	return strings.Contains(URL, "another"), nil
}

func (t testRepoImplementation) New(URL string, options manager.RepoOptions) (manager.Repo, error) {
	return &testRepoImplementation{}, nil
}

func (t testRepoImplementation) Type() string {
	return "test"
}

func (t testRepoImplementation) FullURL() *url.URL {
	return &url.URL{}
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
