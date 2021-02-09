package cluster

import (
	"context"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type StepTestCases map[string]struct {
	args    StepTestArgs
	wantErr bool
}

type StepTestArgs struct {
	cluster       *synv1alpha1.Cluster
	tenant        *synv1alpha1.Tenant
	template      *synv1alpha1.TenantTemplate
	data          *pipeline.Context
	finalizerName string
}

var rbacCases = StepTestCases{
	"create cluster RBAC": {
		wantErr: false,
		args: StepTestArgs{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testnamespace",
				},
			},
		},
	},
}

func Test_createClusterRBAC(t *testing.T) {
	for name, tt := range rbacCases {
		client, _ := testSetupClient(map[schema.GroupVersion][]runtime.Object{
			synv1alpha1.SchemeGroupVersion: {
				tt.args.cluster,
			},
		})

		tt.args.data = &pipeline.Context{Client: client}

		t.Run(name, func(t *testing.T) {
			if got := createClusterRBAC(tt.args.cluster, tt.args.data); got.Err != nil != tt.wantErr {
				t.Errorf("createClusterRBAC() = had error: %v", got.Err)
			}

			roleBinding := &rbacv1.RoleBinding{}
			serviceAccount := &corev1.ServiceAccount{}

			namespacedName := types.NamespacedName{Name: tt.args.cluster.Name, Namespace: tt.args.cluster.Namespace}

			assert.NoError(t, client.Get(context.TODO(), namespacedName, roleBinding))
			assert.NoError(t, client.Get(context.TODO(), namespacedName, serviceAccount))

			assert.Equal(t, serviceAccount.Name, roleBinding.Subjects[len(roleBinding.Subjects)-1].Name)
			assert.Equal(t, serviceAccount.Namespace, roleBinding.Subjects[len(roleBinding.Subjects)-1].Namespace)

		})
	}
}

var setBootstrapTokenCases = StepTestCases{
	"Set bootstrap token": {
		args: StepTestArgs{
			cluster: &synv1alpha1.Cluster{},
			data: &pipeline.Context{
				Log: zap.New(),
			},
		},
	},
}

func Test_setBootstrapToken(t *testing.T) {
	for name, tt := range setBootstrapTokenCases {
		t.Run(name, func(t *testing.T) {
			if got := setBootstrapToken(tt.args.cluster, tt.args.data); got.Err != nil != tt.wantErr {
				t.Errorf("setBootstrapToken() = had error: %v", got.Err)
			}

			assert.NotNil(t, tt.args.cluster.Status.BootstrapToken)
		})
	}
}

var clusterStatusCases = StepTestCases{
	"new cluster status": {
		args: StepTestArgs{
			cluster: &synv1alpha1.Cluster{},
		},
	},
}

func Test_newClusterStatus(t *testing.T) {
	for name, tt := range clusterStatusCases {
		t.Run(name, func(t *testing.T) {
			if err := newClusterStatus(tt.args.cluster); (err != nil) != tt.wantErr {
				t.Errorf("newClusterStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.NotNil(t, tt.args.cluster.Status.BootstrapToken)

		})
	}
}

var setTenantOwnerCases = StepTestCases{
	"set tenant owner": {
		args: StepTestArgs{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					TenantRef: corev1.LocalObjectReference{Name: "testTenant"},
				},
			},
			tenant: &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testTenant",
				},
			},
			data: &pipeline.Context{},
		},
	},
	"tenant does not exist": {
		wantErr: true,
		args: StepTestArgs{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					TenantRef: corev1.LocalObjectReference{Name: "wrongTenant"},
				},
			},
			tenant: &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testTenant",
				},
			},
			data: &pipeline.Context{},
		},
	},
}

func Test_setTenantOwner(t *testing.T) {
	for name, tt := range setTenantOwnerCases {
		tt.args.data.Client, _ = testSetupClient(map[schema.GroupVersion][]runtime.Object{
			synv1alpha1.SchemeGroupVersion: {
				tt.args.cluster,
				tt.args.tenant,
			},
		})

		t.Run(name, func(t *testing.T) {
			if got := setTenantOwner(tt.args.cluster, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("setTenantOwner() = had error: %v", got.Err)
			}

			if !tt.wantErr {
				assert.NotEmpty(t, tt.args.cluster.GetOwnerReferences())
			}
		})
	}
}

var applyTenantTemplateCases = StepTestCases{
	"apply tenant template": {
		args: StepTestArgs{
			data: &pipeline.Context{},
			cluster: &synv1alpha1.Cluster{
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
	"wrong syntax": {
		wantErr: true,
		args: StepTestArgs{
			data: &pipeline.Context{},
			cluster: &synv1alpha1.Cluster{
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

func Test_applyTenantTemplate(t *testing.T) {
	for name, tt := range applyTenantTemplateCases {
		tt.args.data.Client, _ = testSetupClient(map[schema.GroupVersion][]runtime.Object{
			synv1alpha1.SchemeGroupVersion: {
				tt.args.cluster,
				tt.args.tenant,
			},
		})

		t.Run(name, func(t *testing.T) {
			if got := applyClusterTemplateFromTenant(tt.args.cluster, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("applyClusterTemplateFromTenant() = had error: %v", got.Err)
			}

			if !tt.wantErr {
				assert.Equal(t, "c-some-test", tt.args.cluster.Spec.GitRepoTemplate.RepoName)
			}
		})
	}
}
