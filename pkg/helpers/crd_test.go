package helpers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAddTenantLabel(t *testing.T) {
	type args struct {
		meta   *metav1.ObjectMeta
		tenant string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add labels",
			args: args{
				meta:   &metav1.ObjectMeta{},
				tenant: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddTenantLabel(tt.args.meta, tt.args.tenant)

			if tt.args.meta.Labels[apis.LabelNameTenant] != tt.args.tenant {
				t.Error("labels do not match")
			}

		})
	}
}

func TestCreateOrUpdateGitRepo(t *testing.T) {
	type args struct {
		obj       metav1.Object
		gvk       schema.GroupVersionKind
		template  *synv1alpha1.GitRepoTemplate
		tenantRef v1.LocalObjectReference
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "create and update git repo",
			args: args{
				obj: &synv1alpha1.GitRepo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
				gvk: schema.GroupVersionKind{
					Version: "testVersion",
					Kind:    "testKind",
				},
				template: &synv1alpha1.GitRepoTemplate{
					APISecretRef: v1.SecretReference{Name: "testSecret"},
					DeployKeys:   nil,
					Path:         "testPath",
					RepoName:     "testRepo",
				},
				tenantRef: v1.LocalObjectReference{
					Name: "testTenant",
				},
			},
		},
	}
	for _, tt := range tests {

		objs := []runtime.Object{
			&synv1alpha1.GitRepo{},
		}

		cl := testSetupClient(objs)

		t.Run(tt.name, func(t *testing.T) {
			if err := CreateOrUpdateGitRepo(tt.args.obj, tt.args.gvk, tt.args.template, cl, tt.args.tenantRef); (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateGitRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})

		namespacedName := types.NamespacedName{
			Name:      tt.args.obj.GetName(),
			Namespace: tt.args.obj.GetNamespace(),
		}

		checkRepo := &synv1alpha1.GitRepo{}
		assert.NoError(t, cl.Get(context.TODO(), namespacedName, checkRepo))
		assert.Equal(t, tt.args.template, &checkRepo.Spec.GitRepoTemplate)
		tt.args.template.RepoName = "changedName"
		assert.NoError(t, CreateOrUpdateGitRepo(tt.args.obj, tt.args.gvk, tt.args.template, cl, tt.args.tenantRef))
		assert.NoError(t, cl.Get(context.TODO(), namespacedName, checkRepo))
		assert.Equal(t, tt.args.template, &checkRepo.Spec.GitRepoTemplate)

		checkRepo.Spec.RepoType = synv1alpha1.AutoRepoType
		assert.NoError(t, cl.Update(context.TODO(), checkRepo))
		assert.NoError(t, CreateOrUpdateGitRepo(tt.args.obj, tt.args.gvk, tt.args.template, cl, tt.args.tenantRef))
		finalRepo := &synv1alpha1.GitRepo{}
		assert.NoError(t, cl.Get(context.TODO(), namespacedName, finalRepo))
		assert.Equal(t, checkRepo.Spec.GitRepoTemplate, finalRepo.Spec.GitRepoTemplate)
	}
}

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs []runtime.Object) client.Client {
	s := scheme.Scheme
	s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
	return fake.NewFakeClient(objs...)
}

func TestHandleDeletion(t *testing.T) {
	type args struct {
		instance      metav1.Object
		finalizerName string
	}
	tests := []struct {
		name string
		args args
		want DeletionState
	}{
		{
			name: "Normal deletion",
			want: DeletionState{Deleted: true, FinalizerRemoved: true},
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
			name: "Deletion protection set",
			want: DeletionState{Deleted: true, FinalizerRemoved: false},
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
			name: "Nonsense annotation value",
			want: DeletionState{Deleted: true, FinalizerRemoved: false},
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
		{
			name: "Object not deleted",
			want: DeletionState{Deleted: false, FinalizerRemoved: false},
			args: args{
				instance:      &synv1alpha1.Cluster{},
				finalizerName: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			client := testSetupClient([]runtime.Object{&synv1alpha1.Cluster{}})

			got := HandleDeletion(tt.args.instance, tt.args.finalizerName, client)
			if got != tt.want {
				t.Errorf("HandleDeletion() = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestAddDeletionProtection(t *testing.T) {
	type args struct {
		instance metav1.Object
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

			AddDeletionProtection(tt.args.instance)

			result := tt.args.instance.GetAnnotations()[DeleteProtectionAnnotation]
			if result != tt.args.result {
				t.Errorf("AddDeletionProtection() value = %v, wantValue %v", result, tt.args.result)
			}
		})
	}
}
