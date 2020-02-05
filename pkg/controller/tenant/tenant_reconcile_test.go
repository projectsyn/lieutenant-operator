package tenant

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
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
	return fake.NewFakeClient(objs...), s
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
			objName:   "test",
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
					DisplayName: "test",
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoName: "test",
					},
				},
			}

			objs := []runtime.Object{
				tenant,
				&synv1alpha1.GitRepo{},
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
				Name:      tt.objName + "-tenant",
				Namespace: tt.namespace,
			}

			gitRepo := &synv1alpha1.GitRepo{}
			err = cl.Get(context.TODO(), gitRepoNamespacedName, gitRepo)
			assert.NoError(t, err)

		})
	}
}
