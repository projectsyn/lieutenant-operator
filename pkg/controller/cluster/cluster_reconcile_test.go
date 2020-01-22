package cluster

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"testing"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"

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

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) (client.Client, *runtime.Scheme) {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClient(objs...), s
}

func TestReconcileCluster_Reconcile(t *testing.T) {
	type fields struct {
		tenantName      string
		objName         string
		objNamespace    string
	}
	tests := []struct {
		name    string
		want    reconcile.Result
		wantErr bool
		fields  fields
	}{
		{
			name:    "Check cluster state after creation",
			want:    reconcile.Result{},
			wantErr: false,
			fields: fields{
				tenantName:      "test-tenant",
				objName:         "test-object",
				objNamespace:    "tenant",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tenant := &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.fields.objName,
					Namespace: tt.fields.objNamespace,
				},
				Spec: synv1alpha1.ClusterSpec{
					DisplayName: "test",
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						Spec: synv1alpha1.GitRepoSpec{
							RepoName: "test",
						},
					},
					TenantRef: &corev1.LocalObjectReference{
						Name:      tt.fields.tenantName,
					},
				},
			}

			objs := []runtime.Object{
				tenant,
				&synv1alpha1.GitRepo{},
			}

			cl, s := testSetupClient(objs)

			r := &ReconcileCluster{client: cl, scheme: s}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.fields.objName,
					Namespace: tt.fields.objNamespace,
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
				Name:      tt.fields.tenantName + "-cluster",
				Namespace: tt.fields.objNamespace,
			}

			gitRepo := &synv1alpha1.GitRepo{}
			err = cl.Get(context.TODO(), gitRepoNamespacedName, gitRepo)
			assert.NoError(t, err)

			newCluster := &synv1alpha1.Cluster{}
			err = cl.Get(context.TODO(), req.NamespacedName, newCluster)
			assert.NoError(t, err)

			assert.Equal(t, tt.fields.tenantName, newCluster.Labels[apis.LabelNameTenant])

			assert.NotEmpty(t, newCluster.Status.BootstrapToken.Token)

		})
	}
}
