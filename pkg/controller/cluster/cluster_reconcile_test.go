package cluster

import (
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"github.com/projectsyn/lieutenant-operator/pkg/vault"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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
	return fake.NewFakeClientWithScheme(s, objs...), s
}

func TestReconcileCluster_NoCluster(t *testing.T) {
	cl, s := testSetupClient([]runtime.Object{
		&synv1alpha1.Cluster{},
	})
	r := &ReconcileCluster{client: cl, scheme: s}
	_, err := r.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: "c-not-found",
		},
	})
	assert.NoError(t, err)
}

func TestReconcileCluster_NoTenant(t *testing.T) {
	cluster := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: "inexistent-tenant",
			},
		},
	}

	objs := []runtime.Object{
		cluster,
		&synv1alpha1.GitRepo{},
		&synv1alpha1.Tenant{},
	}

	cl, s := testSetupClient(objs)

	r := &ReconcileCluster{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: cluster.Name,
		},
	}

	_, err := r.Reconcile(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find tenant")
}

func TestReconcileCluster_NoGitRepoTemplate(t *testing.T) {
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-tenant",
		},
	}
	cluster := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: synv1alpha1.ClusterSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: tenant.Name,
			},
		},
	}

	objs := []runtime.Object{
		tenant,
		cluster,
	}

	cl, s := testSetupClient(objs)

	r := &ReconcileCluster{client: cl, scheme: s}

	os.Setenv("SKIP_VAULT_SETUP", "true")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: cluster.Name,
		},
	}

	_, err := r.Reconcile(req)
	assert.NoError(t, err)

	updatedCluster := &synv1alpha1.Cluster{}
	err = cl.Get(context.TODO(), req.NamespacedName, updatedCluster)
	assert.NoError(t, err)
	assert.Nil(t, updatedCluster.Spec.GitRepoTemplate)
}

func TestReconcileCluster_Reconcile(t *testing.T) {
	type fields struct {
		tenantName   string
		objName      string
		objNamespace string
	}
	tests := []struct {
		name      string
		want      reconcile.Result
		wantErr   bool
		skipVault bool
		fields    fields
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
		{
			name:      "Check skip Vault",
			want:      reconcile.Result{},
			skipVault: true,
			wantErr:   false,
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
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "somesecret",
						Annotations: map[string]string{
							corev1.ServiceAccountNameKey: tt.fields.objName,
						},
						CreationTimestamp: metav1.Time{Time: time.Now().Add(15 * time.Second)},
					},
					Type: corev1.SecretTypeServiceAccountToken,
					Data: map[string][]byte{
						"token": []byte("mysecret"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "somesecret1",
						Annotations: map[string]string{
							corev1.ServiceAccountNameKey: tt.fields.objName,
						},
						CreationTimestamp: metav1.Time{Time: time.Now()},
					},
					Type: corev1.SecretTypeServiceAccountToken,
					Data: map[string][]byte{
						"token": []byte("mysecret1"),
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
			testMockClient := &TestMockClient{}
			vault.SetCustomClient(testMockClient)

			os.Setenv("SKIP_VAULT_SETUP", strconv.FormatBool(tt.skipVault))

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

			if tt.skipVault {
				assert.Empty(t, testMockClient.secrets)
			} else {
				saToken, err := pipeline.GetServiceAccountToken(newCluster, &pipeline.ExecutionContext{Client: cl})
				saSecrets := []vault.VaultSecret{{Value: saToken, Path: path.Join(tt.fields.tenantName, tt.fields.objName, "steward")}}
				assert.NoError(t, err)
				assert.Equal(t, testMockClient.secrets, saSecrets)
			}
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
			fileContent, found := testTenant.Spec.GitRepoTemplate.TemplateFiles[tt.fields.objName+".yml"]
			assert.True(t, found)
			assert.Equal(t, fileContent, fmt.Sprintf(clusterClassContent, tt.fields.tenantName, pipeline.CommonClassName))
		})
	}
}

type TestMockClient struct {
	secrets []vault.VaultSecret
}

func (m *TestMockClient) AddSecrets(secrets []vault.VaultSecret) error {
	m.secrets = secrets
	return nil
}

func (m *TestMockClient) RemoveSecrets(secrets []vault.VaultSecret) error {
	return nil
}

func TestReconcileCluster_getServiceAccountToken(t *testing.T) {
	type args struct {
		instance metav1.Object
		objs     []runtime.Object
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "check secret sorting",
			args: args{
				instance: &metav1.ObjectMeta{
					Name: "someCluster",
				},
				objs: []runtime.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "oldersecret",
							Annotations: map[string]string{
								corev1.ServiceAccountNameKey: "someCluster",
							},
							CreationTimestamp: metav1.Time{Time: time.Now()},
						},
						Type: corev1.SecretTypeServiceAccountToken,
						Data: map[string][]byte{
							"token": []byte("oldersecret"),
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "newersecret",
							Annotations: map[string]string{
								corev1.ServiceAccountNameKey: "someCluster",
							},
							CreationTimestamp: metav1.Time{Time: time.Now().Add(1 * time.Second)},
						},
						Type: corev1.SecretTypeServiceAccountToken,
						Data: map[string][]byte{
							"token": []byte("newerseccret"),
						},
					},
				},
			},
			want:    "newerseccret",
			wantErr: false,
		},
		{
			name: "check secret not found",
			args: args{
				instance: &metav1.ObjectMeta{
					Name: "someCluster",
				},
				objs: []runtime.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "oldersecret",
							Annotations: map[string]string{
								corev1.ServiceAccountNameKey: "not",
							},
							CreationTimestamp: metav1.Time{Time: time.Now()},
						},
						Type: corev1.SecretTypeServiceAccountToken,
						Data: map[string][]byte{
							"token": []byte("oldersecret"),
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "newersecret",
							Annotations: map[string]string{
								corev1.ServiceAccountNameKey: "not",
							},
							CreationTimestamp: metav1.Time{Time: time.Now().Add(1 * time.Second)},
						},
						Type: corev1.SecretTypeServiceAccountToken,
						Data: map[string][]byte{
							"token": []byte("newerseccret"),
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "check secret missing token",
			args: args{
				instance: &metav1.ObjectMeta{
					Name: "someCluster",
				},
				objs: []runtime.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "oldersecret",
							Annotations: map[string]string{
								corev1.ServiceAccountNameKey: "someCluster",
							},
							CreationTimestamp: metav1.Time{Time: time.Now()},
						},
						Type: corev1.SecretTypeServiceAccountToken,
						Data: map[string][]byte{
							"token": []byte("oldersecret"),
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "newersecret",
							Annotations: map[string]string{
								corev1.ServiceAccountNameKey: "someCluster",
							},
							CreationTimestamp: metav1.Time{Time: time.Now().Add(1 * time.Second)},
						},
						Type: corev1.SecretTypeServiceAccountToken,
						Data: map[string][]byte{
							"non": []byte("newerseccret"),
						},
					},
				},
			},
			want:    "oldersecret",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cl, _ := testSetupClient(tt.args.objs)

			got, err := pipeline.GetServiceAccountToken(tt.args.instance, &pipeline.ExecutionContext{Client: cl})
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileCluster.getServiceAccountToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReconcileCluster.getServiceAccountToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
