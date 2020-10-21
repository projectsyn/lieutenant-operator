package pipeline

import (
	"os"
	"testing"
	"time"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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

			client := testSetupClient([]runtime.Object{&synv1alpha1.Cluster{}})

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
