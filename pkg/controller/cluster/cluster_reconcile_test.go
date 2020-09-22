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
	"github.com/projectsyn/lieutenant-operator/pkg/controller/tenant"
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
	return fake.NewFakeClient(objs...), s
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
				saToken, err := r.getServiceAccountToken(newCluster)
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
			assert.Equal(t, fileContent, fmt.Sprintf(clusterClassContent, tt.fields.tenantName, tenant.CommonClassName))
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

			cl, s := testSetupClient(tt.args.objs)

			r := &ReconcileCluster{
				client: cl,
				scheme: s,
			}
			got, err := r.getServiceAccountToken(tt.args.instance)
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

func TestClusterCatalogTemplate(t *testing.T) {
	clusterName := "c-test-1234"
	tenantName := "t-acme-corp"
	tenant := &synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: tenantName,
		},
		Spec: synv1alpha1.TenantSpec{
			DisplayName: "Some tenant",
			GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
				Path:     "tenants",
				RepoName: tenantName,
			},
			ClusterCatalog: synv1alpha1.ClusterCatalog{
				GitRepoTemplate: synv1alpha1.TenantClusterCatalogTemplate{
					Path:     `clusters/{{ index .Facts "cloud" }}/` + tenantName,
					RepoName: "cluster-{{ .ClusterID }}",
					APISecretRef: corev1.SecretReference{
						Name: "git-{{ .TenantID }}",
					},
				},
			},
		},
	}
	cluster := &synv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: synv1alpha1.ClusterSpec{
			DisplayName: "This is a test cluster",
			TenantRef: corev1.LocalObjectReference{
				Name: tenant.Name,
			},
			Facts: &synv1alpha1.Facts{
				"cloud": "cloudscale",
			},
		},
	}

	objs := []runtime.Object{
		cluster,
		tenant,
		&synv1alpha1.GitRepo{},
	}

	cl, s := testSetupClient(objs)

	r := &ReconcileCluster{client: cl, scheme: s}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: cluster.Name,
		},
	}
	testMockClient := &TestMockClient{}
	vault.SetCustomClient(testMockClient)

	os.Setenv("SKIP_VAULT_SETUP", "true")

	_, err := r.Reconcile(req)
	assert.NoError(t, err)

	newCluster := &synv1alpha1.Cluster{}
	err = cl.Get(context.TODO(), req.NamespacedName, newCluster)
	assert.NoError(t, err)

	assert.Equal(t, "cluster-"+clusterName, newCluster.Spec.GitRepoTemplate.RepoName)
	assert.Equal(t, "clusters/cloudscale/"+tenantName, newCluster.Spec.GitRepoTemplate.Path)
	assert.Equal(t, "git-"+tenantName, newCluster.Spec.GitRepoTemplate.APISecretRef.Name)
	assert.Empty(t, newCluster.Spec.GitRepoTemplate.APISecretRef.Namespace)
}
