package cluster

import (
	"context"
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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// testSetupClient returns a client containing all objects in objs
func testSetupClient(objs map[schema.GroupVersion][]runtime.Object) (client.Client, *runtime.Scheme) {
	s := scheme.Scheme
	var initObjs []runtime.Object
	for group, groupObjs := range objs {
		s.AddKnownTypes(group, groupObjs...)
		initObjs = append(initObjs, groupObjs...)
	}
	return fake.NewFakeClientWithScheme(s, initObjs...), s
}

func readObject(t *testing.T, c client.Client, ns types.NamespacedName, obj runtime.Object) {
	err := c.Get(context.Background(), ns, obj)
	require.NoError(t, err)
}

func reconcileCluster(c client.Client, s *runtime.Scheme, name types.NamespacedName) (reconcile.Result, error) {
	r := &ReconcileCluster{client: c, scheme: s}

	req := reconcile.Request{
		NamespacedName: name,
	}

	return r.Reconcile(req)
}

func TestReconcileCluster_NoCluster(t *testing.T) {
	cl, s := testSetupClient(map[schema.GroupVersion][]runtime.Object{
		synv1alpha1.SchemeGroupVersion: {&synv1alpha1.Cluster{}},
	})

	_, err := reconcileCluster(cl, s, types.NamespacedName{
		Name: "c-not-found",
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

	objs := map[schema.GroupVersion][]runtime.Object{
		synv1alpha1.SchemeGroupVersion: {
			cluster,
			&synv1alpha1.GitRepo{},
			&synv1alpha1.Tenant{},
		},
	}

	cl, s := testSetupClient(objs)

	_, err := reconcileCluster(cl, s, types.NamespacedName{
		Name: cluster.Name,
	})

	assert.Contains(t, err.Error(), "no matching secrets found")
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

	objs := map[schema.GroupVersion][]runtime.Object{
		synv1alpha1.SchemeGroupVersion: {
			tenant,
			cluster,
			&synv1alpha1.GitRepo{},
		},
	}

	cl, s := testSetupClient(objs)

	os.Setenv("SKIP_VAULT_SETUP", "true")

	name := types.NamespacedName{
		Name: cluster.Name,
	}
	_, err := reconcileCluster(cl, s, name)
	assert.NoError(t, err)

	updatedCluster := &synv1alpha1.Cluster{}
	err = cl.Get(context.TODO(), name, updatedCluster)
	assert.NoError(t, err)
	assert.Nil(t, updatedCluster.Spec.GitRepoTemplate)
}

var reconcileCluster_ReconcileTests = map[string]struct {
	want         reconcile.Result
	wantErr      bool
	skipVault    bool
	tenantName   string
	objName      string
	objNamespace string
}{
	"Check cluster state after creation": {
		want:         reconcile.Result{},
		wantErr:      false,
		tenantName:   "test-tenant",
		objName:      "test-object",
		objNamespace: "tenant",
	},
	"Check skip Vault": {
		want:         reconcile.Result{},
		skipVault:    true,
		wantErr:      false,
		tenantName:   "test-tenant",
		objName:      "test-object",
		objNamespace: "tenant",
	},
}

func TestReconcileCluster_Reconcile(t *testing.T) {
	for name, tt := range reconcileCluster_ReconcileTests {
		t.Run(name, func(t *testing.T) {

			cluster := &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.objName,
					Namespace: tt.objNamespace,
				},
				Spec: synv1alpha1.ClusterSpec{
					DisplayName: "desc",
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoName: "test",
						Path:     "test",
					},
					TenantRef: corev1.LocalObjectReference{
						Name: tt.tenantName,
					},
				},
			}

			objs := map[schema.GroupVersion][]runtime.Object{
				synv1alpha1.SchemeGroupVersion: {
					cluster,
					&synv1alpha1.GitRepo{},
					&synv1alpha1.Tenant{
						ObjectMeta: metav1.ObjectMeta{
							Name:      tt.tenantName,
							Namespace: tt.objNamespace,
						},
						Spec: synv1alpha1.TenantSpec{
							DisplayName:     tt.tenantName,
							GitRepoTemplate: &synv1alpha1.GitRepoTemplate{},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "somesecret",
							Annotations: map[string]string{
								corev1.ServiceAccountNameKey: tt.objName,
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
								corev1.ServiceAccountNameKey: tt.objName,
							},
							CreationTimestamp: metav1.Time{Time: time.Now()},
						},
						Type: corev1.SecretTypeServiceAccountToken,
						Data: map[string][]byte{
							"token": []byte("mysecret1"),
						},
					},
				},
			}

			cl, s := testSetupClient(objs)

			testMockClient := &TestMockClient{}
			vault.SetCustomClient(testMockClient)

			os.Setenv("SKIP_VAULT_SETUP", strconv.FormatBool(tt.skipVault))

			name := types.NamespacedName{
				Name:      tt.objName,
				Namespace: tt.objNamespace,
			}
			got, err := reconcileCluster(cl, s, name)

			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}

			// BootstrapToken is now only populated after the second reconcile.
			_, err = reconcileCluster(cl, s, name)
			assert.NoError(t, err)

			gitRepoNamespacedName := types.NamespacedName{
				Name:      tt.objName,
				Namespace: tt.objNamespace,
			}

			gitRepo := &synv1alpha1.GitRepo{}
			err = cl.Get(context.TODO(), gitRepoNamespacedName, gitRepo)
			assert.NoError(t, err)
			assert.Equal(t, cluster.Spec.DisplayName, gitRepo.Spec.GitRepoTemplate.DisplayName)

			newCluster := &synv1alpha1.Cluster{}
			err = cl.Get(context.TODO(), name, newCluster)
			assert.NoError(t, err)

			assert.Equal(t, tt.tenantName, newCluster.Labels[apis.LabelNameTenant])

			assert.NotNil(t, newCluster.Status.BootstrapToken)
			assert.NotEmpty(t, newCluster.Status.BootstrapToken.Token)

			sa := &corev1.ServiceAccount{}
			err = cl.Get(context.TODO(), name, sa)
			assert.NoError(t, err)

			if tt.skipVault {
				assert.Empty(t, testMockClient.secrets)
			} else {
				saToken, err := pipeline.GetServiceAccountToken(newCluster, &pipeline.ExecutionContext{Client: cl})
				saSecrets := []vault.VaultSecret{{Value: saToken, Path: path.Join(tt.tenantName, tt.objName, "steward")}}
				assert.NoError(t, err)
				assert.Equal(t, testMockClient.secrets, saSecrets)
			}
			role := &rbacv1.Role{}
			err = cl.Get(context.TODO(), name, role)
			assert.NoError(t, err)
			assert.Contains(t, role.Rules[0].ResourceNames, name.Name)

			roleBinding := &rbacv1.RoleBinding{}
			err = cl.Get(context.TODO(), name, roleBinding)
			assert.NoError(t, err)
			assert.Equal(t, roleBinding.RoleRef.Name, role.Name)
			assert.Equal(t, roleBinding.Subjects[0].Name, sa.Name)

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

func (m *TestMockClient) SetDeletionPolicy(deletionPolicy synv1alpha1.DeletionPolicy) {}

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

			cl, _ := testSetupClient(map[schema.GroupVersion][]runtime.Object{
				synv1alpha1.SchemeGroupVersion: tt.args.objs,
			})

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

func TestReconcileCluster_unmanagedGitRepo(t *testing.T) {
	objs := map[schema.GroupVersion][]runtime.Object{
		synv1alpha1.SchemeGroupVersion: {
			&synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
				},
				Spec: synv1alpha1.ClusterSpec{
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoType: synv1alpha1.UnmanagedRepoType,
					},
					GitRepoURL: "someURL",
					TenantRef: corev1.LocalObjectReference{
						Name: "tenant-a",
					},
				},
			},
			&synv1alpha1.GitRepo{},
			&synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tenant-a",
				},
				Spec: synv1alpha1.TenantSpec{
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{},
				},
			},
		},
	}

	cl, s := testSetupClient(objs)

	vault.SetCustomClient(&TestMockClient{})
	os.Setenv("SKIP_VAULT_SETUP", "true")

	_, err := reconcileCluster(cl, s, types.NamespacedName{
		Name: "cluster-a",
	})
	require.NoError(t, err)

	updatedCluster := &synv1alpha1.Cluster{}
	err = cl.Get(context.TODO(), types.NamespacedName{Name: "cluster-a"}, updatedCluster)
	require.NoError(t, err)
	assert.NotEmpty(t, updatedCluster.Spec.GitRepoURL)
	assert.Equal(t, "someURL", updatedCluster.Spec.GitRepoURL)

}
