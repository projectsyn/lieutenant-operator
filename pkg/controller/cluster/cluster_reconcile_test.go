package cluster

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"

	"github.com/stretchr/testify/assert"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
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
		tenantName   string
		objName      string
		objNamespace string
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
				tenantName:   "test-tenant",
				objName:      "test-object",
				objNamespace: "tenant",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cluster := &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.fields.objName,
					Namespace: tt.fields.objNamespace,
				},
				Spec: synv1alpha1.ClusterSpec{
					DisplayName: "desc",
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoName: "test",
						Path:     "test",
					},
					TenantRef: corev1.LocalObjectReference{
						Name: tt.fields.tenantName,
					},
				},
			}

			objs := []runtime.Object{
				cluster,
				&synv1alpha1.GitRepo{},
				&synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tt.fields.tenantName,
						Namespace: tt.fields.objNamespace,
					},
					Spec: synv1alpha1.TenantSpec{
						DisplayName:     tt.fields.tenantName,
						GitRepoTemplate: &synv1alpha1.GitRepoTemplate{},
					},
				},
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
				Name:      tt.fields.objName,
				Namespace: tt.fields.objNamespace,
			}

			gitRepo := &synv1alpha1.GitRepo{}
			err = cl.Get(context.TODO(), gitRepoNamespacedName, gitRepo)
			assert.NoError(t, err)
			assert.Equal(t, cluster.Spec.DisplayName, gitRepo.Spec.GitRepoTemplate.DisplayName)

			newCluster := &synv1alpha1.Cluster{}
			err = cl.Get(context.TODO(), req.NamespacedName, newCluster)
			assert.NoError(t, err)

			assert.Equal(t, tt.fields.tenantName, newCluster.Labels[apis.LabelNameTenant])

			assert.NotEmpty(t, newCluster.Status.BootstrapToken.Token)

			sa := &corev1.ServiceAccount{}
			err = cl.Get(context.TODO(), req.NamespacedName, sa)
			assert.NoError(t, err)

			role := &rbacv1.Role{}
			err = cl.Get(context.TODO(), req.NamespacedName, role)
			assert.NoError(t, err)
			assert.Contains(t, role.Rules[0].ResourceNames, req.Name)

			roleBinding := &rbacv1.RoleBinding{}
			err = cl.Get(context.TODO(), req.NamespacedName, roleBinding)
			assert.NoError(t, err)
			assert.Equal(t, roleBinding.RoleRef.Name, role.Name)
			assert.Equal(t, roleBinding.Subjects[0].Name, sa.Name)

			testTenant := &synv1alpha1.Tenant{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: tt.fields.tenantName, Namespace: tt.fields.objNamespace}, testTenant)
			assert.NoError(t, err)
			_, found := testTenant.Spec.GitRepoTemplate.TemplateFiles[tt.fields.objName+".yml"]
			assert.True(t, found)

		})
	}
}
