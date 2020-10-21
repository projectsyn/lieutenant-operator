package pipeline

import (
	"context"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func Test_createClusterRBAC(t *testing.T) {
	type args struct {
		obj  *synv1alpha1.Cluster
		data *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "create cluster RBAC",
			wantErr: false,
			args: args{
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "testnamespace",
					},
				},
			},
		},
	}
	for _, tt := range tests {

		client, _ := testSetupClient([]runtime.Object{
			tt.args.obj,
		})

		tt.args.data = &ExecutionContext{Client: client}

		t.Run(tt.name, func(t *testing.T) {
			if got := createClusterRBAC(tt.args.obj, tt.args.data); got.Err != nil != tt.wantErr {
				t.Errorf("createClusterRBAC() = had error: %v", got.Err)
			}
		})

		roleBinding := &rbacv1.RoleBinding{}
		serviceAccount := &corev1.ServiceAccount{}

		namespacedName := types.NamespacedName{Name: tt.args.obj.Name, Namespace: tt.args.obj.Namespace}

		assert.NoError(t, client.Get(context.TODO(), namespacedName, roleBinding))
		assert.NoError(t, client.Get(context.TODO(), namespacedName, serviceAccount))

		assert.Equal(t, serviceAccount.Name, roleBinding.Subjects[len(roleBinding.Subjects)-1].Name)
		assert.Equal(t, serviceAccount.Namespace, roleBinding.Subjects[len(roleBinding.Subjects)-1].Namespace)

	}
}

func Test_setBootstrapToken(t *testing.T) {
	type args struct {
		obj  *synv1alpha1.Cluster
		data *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Set bootstrap token",
			args: args{
				obj: &synv1alpha1.Cluster{},
				data: &ExecutionContext{
					Log: zap.New(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setBootstrapToken(tt.args.obj, tt.args.data); got.Err != nil != tt.wantErr {
				t.Errorf("setBootstrapToken() = had error: %v", got.Err)
			}
		})

		assert.NotNil(t, tt.args.obj.Status.BootstrapToken)

	}
}

func Test_newClusterStatus(t *testing.T) {
	type args struct {
		cluster *synv1alpha1.Cluster
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "new cluster status",
			args: args{
				cluster: &synv1alpha1.Cluster{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := newClusterStatus(tt.args.cluster); (err != nil) != tt.wantErr {
				t.Errorf("newClusterStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})

		assert.NotNil(t, tt.args.cluster.Status.BootstrapToken)

	}
}

func Test_setTenantOwner(t *testing.T) {
	type args struct {
		obj    *synv1alpha1.Cluster
		tenant *synv1alpha1.Tenant
		data   *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "set tenant owner",
			args: args{
				obj: &synv1alpha1.Cluster{
					Spec: synv1alpha1.ClusterSpec{
						TenantRef: corev1.LocalObjectReference{Name: "testTenant"},
					},
				},
				tenant: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testTenant",
					},
				},
				data: &ExecutionContext{},
			},
		},
		{
			name:    "tenant does not exist",
			wantErr: true,
			args: args{
				obj: &synv1alpha1.Cluster{
					Spec: synv1alpha1.ClusterSpec{
						TenantRef: corev1.LocalObjectReference{Name: "wrongTenant"},
					},
				},
				tenant: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testTenant",
					},
				},
				data: &ExecutionContext{},
			},
		},
	}
	for _, tt := range tests {

		tt.args.data.Client, _ = testSetupClient([]runtime.Object{
			tt.args.obj,
			tt.args.tenant,
		})

		t.Run(tt.name, func(t *testing.T) {
			if got := setTenantOwner(tt.args.obj, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("setTenantOwner() = had error: %v", got.Err)
			}
		})

		if !tt.wantErr {
			assert.NotEmpty(t, tt.args.obj.GetOwnerReferences())
		}

	}
}

func Test_applyTenantTemplate(t *testing.T) {
	type args struct {
		obj    *synv1alpha1.Cluster
		tenant *synv1alpha1.Tenant
		data   *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "apply tenant template",
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c-some-test",
					},
				},
				tenant: &synv1alpha1.Tenant{
					Spec: synv1alpha1.TenantSpec{
						ClusterTemplate: &synv1alpha1.ClusterSpec{
							DisplayName: "test",
							GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
								RepoName: "{{ .Name }}",
							},
						},
					},
				},
			},
		},
		{
			name:    "wrong syntax",
			wantErr: true,
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "c-some-test",
					},
				},
				tenant: &synv1alpha1.Tenant{
					Spec: synv1alpha1.TenantSpec{
						ClusterTemplate: &synv1alpha1.ClusterSpec{
							DisplayName: "test",
							GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
								RepoName: "{{ .Name }",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {

		tt.args.data.Client, _ = testSetupClient([]runtime.Object{
			tt.args.obj,
			tt.args.tenant,
		})

		t.Run(tt.name, func(t *testing.T) {
			if got := applyTenantTemplate(tt.args.obj, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("applyTenantTemplate() = had error: %v", got.Err)
			}
		})

		if !tt.wantErr {
			assert.Equal(t, "c-some-test", tt.args.obj.Spec.GitRepoTemplate.RepoName)
		}

	}
}
