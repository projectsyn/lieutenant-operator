package pipeline

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type genericCases map[string]struct {
	args    args
	wantErr bool
}

type args struct {
	cluster       *synv1alpha1.Cluster
	tenant        *synv1alpha1.Tenant
	template      *synv1alpha1.TenantTemplate
	data          *Context
	finalizerName string
}

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) (client.Client, *runtime.Scheme) {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClientWithScheme(s, objs...), s
}

func TestGetDeletionPolicyDefault(t *testing.T) {
	policy := GetDefaultDeletionPolicy()
	assert.Equal(t, synv1alpha1.ArchivePolicy, policy)
}

func TestGetDeletionPolicyNonDefault(t *testing.T) {
	os.Setenv("DEFAULT_DELETION_POLICY", "Retain")
	policy := GetDefaultDeletionPolicy()
	assert.Equal(t, synv1alpha1.RetainPolicy, policy)
}

var addTenantLabelCases = genericCases{
	"add labels": {
		args: args{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					TenantRef: corev1.LocalObjectReference{Name: "test"},
				},
			},
		},
	},
}

func TestAddTenantLabel(t *testing.T) {
	for name, tt := range addTenantLabelCases {
		t.Run(name, func(t *testing.T) {
			AddTenantLabel(tt.args.cluster, &Context{})

			if tt.args.cluster.GetLabels()[apis.LabelNameTenant] != tt.args.cluster.Spec.TenantRef.Name {
				t.Error("labels do not match")
			}
		})
	}
}

var handleDeletionCases = genericCases{
	"Normal deletion": {
		args: args{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers: []string{
						"test",
					},
				},
			},
			finalizerName: "test",
		},
	},
	"Deletion protection set": {
		wantErr: true,
		args: args{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteProtectionAnnotation: "true",
					},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers: []string{
						"test",
					},
				},
			},
			finalizerName: "test",
		},
	},
	"Nonsense annotation value": {
		wantErr: true,
		args: args{
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						DeleteProtectionAnnotation: "trugadse",
					},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers: []string{
						"test",
					},
				},
			},
			finalizerName: "test",
		},
	},
}

func TestHandleDeletion(t *testing.T) {
	for name, tt := range handleDeletionCases {
		t.Run(name, func(t *testing.T) {
			client, _ := testSetupClient([]runtime.Object{&synv1alpha1.Cluster{}})

			data := &Context{
				Client:        client,
				Deleted:       true,
				FinalizerName: tt.args.finalizerName,
			}

			got := handleDeletion(tt.args.cluster, data)
			if got.Err != nil != tt.wantErr {
				t.Errorf("HandleDeletion() = had error: %v", got.Err)
			}

			want := []string{tt.args.finalizerName}

			assert.Equal(t, want, tt.args.cluster.GetFinalizers())
		})
	}
}

type addDeletionProtectionArgs struct {
	instance *synv1alpha1.Cluster
	enable   string
	result   string
}

var addDeletionProtectionCases = map[string]struct {
	args    addDeletionProtectionArgs
	wantErr bool
}{
	"Add deletion protection": {
		args: addDeletionProtectionArgs{
			instance: &synv1alpha1.Cluster{},
			enable:   "true",
			result:   "true",
		},
	},
	"Don't add deletion protection": {
		args: addDeletionProtectionArgs{
			instance: &synv1alpha1.Cluster{},
			enable:   "false",
			result:   "",
		},
	},
	"Invalid setting": {
		args: addDeletionProtectionArgs{
			instance: &synv1alpha1.Cluster{},
			enable:   "gaga",
			result:   "true",
		},
	},
}

func TestAddDeletionProtection(t *testing.T) {
	for name, tt := range addDeletionProtectionCases {
		t.Run(name, func(t *testing.T) {
			os.Setenv(protectionSettingEnvVar, tt.args.enable)

			addDeletionProtection(tt.args.instance, &Context{})

			result := tt.args.instance.GetAnnotations()[DeleteProtectionAnnotation]
			if result != tt.args.result {
				t.Errorf("AddDeletionProtection() value = %v, wantValue %v", result, tt.args.result)
			}
		})
	}
}

var checkIfDeletedCases = map[string]struct {
	args    args
	wantErr bool
	want    bool
}{
	"object deleted": {
		want: true,
		args: args{
			data: &Context{},
			tenant: &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
		},
	},
	"object not deleted": {
		want: false,
		args: args{
			data: &Context{},
			tenant: &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
	},
}

func Test_checkIfDeleted(t *testing.T) {
	for name, tt := range checkIfDeletedCases {
		t.Run(name, func(t *testing.T) {
			if got := CheckIfDeleted(tt.args.tenant, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("CheckIfDeleted() = had error %v", got.Err)
			}

			assert.Equal(t, tt.want, tt.args.data.Deleted)
		})
	}
}

var handleFinalizerCases = genericCases{
	"add finalizers": {
		args: args{
			data: &Context{
				FinalizerName: "test",
			},
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
	},
	"remove finalizers": {
		args: args{
			data: &Context{
				Deleted:       true,
				FinalizerName: "test",
			},
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
	},
}

func Test_handleFinalizer(t *testing.T) {
	for name, tt := range handleFinalizerCases {
		t.Run(name, func(t *testing.T) {
			if got := handleFinalizer(tt.args.cluster, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("handleFinalizer() = had error: %v", got.Err)
			}

			if tt.args.data.Deleted {
				assert.Empty(t, tt.args.cluster.GetFinalizers())
			} else {
				assert.NotEmpty(t, tt.args.cluster.GetFinalizers())
			}
		})
	}
}

var updateObjectCases = genericCases{
	"update objects": {
		args: args{
			data: &Context{
				originalObject: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
			tenant: &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
		},
	},
	"update fail": {
		args: args{
			data: &Context{
				originalObject: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
			tenant: &synv1alpha1.Tenant{},
		},
	},
}

func Test_updateObject(t *testing.T) {
	for name, tt := range updateObjectCases {
		t.Run(name, func(t *testing.T) {
			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.tenant,
			})

			if got := updateObject(tt.args.tenant, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("updateObject() = had error: %v", got.Err)
			}
		})
	}
}

func Test_updateObjectStatus(t *testing.T) {
	ex := &Context{
		originalObject: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "cluster-a",
				ResourceVersion: "1234",
			},
		},
	}
	cluster := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cluster-a",
			ResourceVersion: "1234",
		},
		Spec: synv1alpha1.ClusterSpec{
			DisplayName: "My Test cluster",
		},
	}
	ex.Client, _ = testSetupClient([]runtime.Object{cluster})

	res := updateObject(&synv1alpha1.Cluster{
		ObjectMeta: cluster.ObjectMeta,
		Spec:       cluster.Spec,
		Status: synv1alpha1.ClusterStatus{
			BootstrapToken: &synv1alpha1.BootstrapToken{
				Token: "token-1234",
			},
		},
	}, ex)
	require.NoError(t, res.Err)

	updatedCluster := &synv1alpha1.Cluster{}
	err := ex.Client.Get(context.Background(), types.NamespacedName{Name: cluster.Name}, updatedCluster)
	require.NoError(t, err)

	assert.NotNil(t, updatedCluster.Status.BootstrapToken)
	assert.Equal(t, "token-1234", updatedCluster.Status.BootstrapToken.Token)
	assert.NotEqual(t, updatedCluster.ResourceVersion, cluster.GetResourceVersion())
}
