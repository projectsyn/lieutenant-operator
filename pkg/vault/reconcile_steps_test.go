package vault

import (
	"os"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type testMockClient struct {
	deletionPolicy synv1alpha1.DeletionPolicy
}

func (m *testMockClient) AddSecrets(secrets []VaultSecret) error { return nil }

func (m *testMockClient) RemoveSecrets(secrets []VaultSecret) error { return nil }

func (m *testMockClient) SetDeletionPolicy(deletionPolicy synv1alpha1.DeletionPolicy) {
	m.deletionPolicy = deletionPolicy
}

type args struct {
	cluster       *synv1alpha1.Cluster
	tenant        *synv1alpha1.Tenant
	template      *synv1alpha1.TenantTemplate
	data          *pipeline.Context
	finalizerName string
}

var getVaultCases = map[string]struct {
	want synv1alpha1.DeletionPolicy
	args args
}{
	"without specific deletion policy": {
		want: pipeline.GetDefaultDeletionPolicy(),
		args: args{
			cluster: &synv1alpha1.Cluster{},
			data: &pipeline.Context{
				Log: zap.New(),
			},
		},
	},
	"specific deletion policy": {
		want: synv1alpha1.DeletePolicy,
		args: args{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					DeletionPolicy: synv1alpha1.DeletePolicy,
				},
			},
			data: &pipeline.Context{
				Log: zap.New(),
			},
		},
	},
}

func Test_getVaultClient(t *testing.T) {
	// ensure that it isn't set to anything from previous tests
	os.Unsetenv("DEFAULT_DELETION_POLICY")

	mockClient := &testMockClient{}

	SetCustomClient(mockClient)

	for name, tt := range getVaultCases {

		t.Run(name, func(t *testing.T) {
			_, err := getVaultClient(tt.args.cluster, tt.args.data)
			assert.NoError(t, err)

			assert.Equal(t, tt.want, mockClient.deletionPolicy)

		})
	}
}

var handleVaultDeletionCases = map[string]struct {
	want synv1alpha1.DeletionPolicy
	args args
}{
	"archive": {
		want: synv1alpha1.ArchivePolicy,
		args: args{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					DeletionPolicy: synv1alpha1.ArchivePolicy,
				},
			},
			data: &pipeline.Context{
				Deleted: true,
			},
		},
	},
	"delete": {
		want: synv1alpha1.DeletePolicy,
		args: args{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					DeletionPolicy: synv1alpha1.DeletePolicy,
				},
			},
			data: &pipeline.Context{
				Deleted: true,
			},
		},
	},
}

func Test_handleVaultDeletion(t *testing.T) {
	// ensure that it isn't set to anything from previous tests
	os.Unsetenv("DEFAULT_DELETION_POLICY")

	mockClient := &testMockClient{
		deletionPolicy: pipeline.GetDefaultDeletionPolicy(),
	}

	SetCustomClient(mockClient)

	for name, tt := range handleVaultDeletionCases {
		t.Run(name, func(t *testing.T) {

			s := scheme.Scheme
			objs := []runtime.Object{
				tt.args.cluster,
			}

			s.AddKnownTypes(synv1alpha1.SchemeGroupVersion, objs...)
			tt.args.data.Client = fake.NewFakeClientWithScheme(s, objs...)

			got := HandleVaultDeletion(tt.args.cluster, tt.args.data)
			assert.NoError(t, got.Err)
			assert.Equal(t, tt.want, mockClient.deletionPolicy)
		})
	}
}
