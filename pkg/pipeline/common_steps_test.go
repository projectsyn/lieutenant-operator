package pipeline

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) (client.Client, *runtime.Scheme) {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClientWithScheme(s, objs...), s
}

func TestGetDeletionPolicyDefault(t *testing.T) {
	policy := getDefaultDeletionPolicy()
	assert.Equal(t, synv1alpha1.ArchivePolicy, policy)
}

func TestGetDeletionPolicyNonDefault(t *testing.T) {
	os.Setenv("DEFAULT_DELETION_POLICY", "Retain")
	policy := getDefaultDeletionPolicy()
	assert.Equal(t, synv1alpha1.RetainPolicy, policy)
}

func TestAddTenantLabel(t *testing.T) {
	type args struct {
		obj *synv1alpha1.Cluster
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add labels",
			args: args{
				obj: &synv1alpha1.Cluster{
					Spec: synv1alpha1.ClusterSpec{
						TenantRef: corev1.LocalObjectReference{Name: "test"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addTenantLabel(tt.args.obj, &ExecutionContext{})

			if tt.args.obj.GetLabels()[apis.LabelNameTenant] != tt.args.obj.Spec.TenantRef.Name {
				t.Error("labels do not match")
			}

		})
	}
}

func TestHandleDeletion(t *testing.T) {
	type args struct {
		instance      *synv1alpha1.Cluster
		finalizerName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Normal deletion",
			args: args{
				instance: &synv1alpha1.Cluster{
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
		{
			name:    "Deletion protection set",
			wantErr: true,
			args: args{
				instance: &synv1alpha1.Cluster{
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
		{
			name:    "Nonsense annotation value",
			wantErr: true,
			args: args{
				instance: &synv1alpha1.Cluster{
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			client, _ := testSetupClient([]runtime.Object{&synv1alpha1.Cluster{}})

			data := &ExecutionContext{
				Client:        client,
				Deleted:       true,
				FinalizerName: tt.args.finalizerName,
			}

			got := handleDeletion(tt.args.instance, data)
			if got.Err != nil != tt.wantErr {
				t.Errorf("HandleDeletion() = had error: %v", got.Err)
			}

			want := []string{tt.args.finalizerName}

			assert.Equal(t, want, tt.args.instance.GetFinalizers())

		})
	}
}

func TestAddDeletionProtection(t *testing.T) {
	type args struct {
		instance *synv1alpha1.Cluster
		enable   string
		result   string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Add deletion protection",
			args: args{
				instance: &synv1alpha1.Cluster{},
				enable:   "true",
				result:   "true",
			},
		},
		{
			name: "Don't add deletion protection",
			args: args{
				instance: &synv1alpha1.Cluster{},
				enable:   "false",
				result:   "",
			},
		},
		{
			name: "Invalid setting",
			args: args{
				instance: &synv1alpha1.Cluster{},
				enable:   "gaga",
				result:   "true",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			os.Setenv(protectionSettingEnvVar, tt.args.enable)

			addDeletionProtection(tt.args.instance, &ExecutionContext{})

			result := tt.args.instance.GetAnnotations()[DeleteProtectionAnnotation]
			if result != tt.args.result {
				t.Errorf("AddDeletionProtection() value = %v, wantValue %v", result, tt.args.result)
			}
		})
	}
}

func Test_checkIfDeleted(t *testing.T) {
	type args struct {
		obj  *synv1alpha1.Tenant
		data *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    bool
	}{
		{
			name: "object deleted",
			want: true,
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
				},
			},
		},
		{
			name: "object not deleted",
			want: false,
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkIfDeleted(tt.args.obj, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("checkIfDeleted() = had error %v", got.Err)
			}

			assert.Equal(t, tt.want, tt.args.data.Deleted)

		})
	}
}

func Test_handleFinalizer(t *testing.T) {
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
			name: "add finalizers",
			args: args{
				data: &ExecutionContext{
					FinalizerName: "test",
				},
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
		},
		{
			name: "remove finalizers",
			args: args{
				data: &ExecutionContext{
					Deleted:       true,
					FinalizerName: "test",
				},
				obj: &synv1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleFinalizer(tt.args.obj, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("handleFinalizer() = had error: %v", got.Err)
			}

			if tt.args.data.Deleted {
				assert.Empty(t, tt.args.obj.GetFinalizers())
			} else {
				assert.NotEmpty(t, tt.args.obj.GetFinalizers())
			}

		})
	}
}

func Test_resultNotOK(t *testing.T) {
	assert.True(t, resultNotOK(ExecutionResult{Err: fmt.Errorf("test")}))
	assert.False(t, resultNotOK(ExecutionResult{}))
}

func Test_wrapError(t *testing.T) {
	assert.Equal(t, "step test failed: test", wrapError("test", fmt.Errorf("test")).Error())
	assert.Nil(t, wrapError("test", nil))
}

func Test_updateObject(t *testing.T) {
	type args struct {
		obj  *synv1alpha1.Tenant
		data *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "update objects",
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				},
			},
		},
		{
			name: "update fail",
			args: args{
				data: &ExecutionContext{},
				obj:  &synv1alpha1.Tenant{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.obj,
			})

			if got := updateObject(tt.args.obj, tt.args.data); (got.Err != nil) != tt.wantErr {
				t.Errorf("updateObject() = had error: %v", got.Err)
			}

		})
	}
}
