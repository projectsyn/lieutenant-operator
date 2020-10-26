package pipeline

import (
	"os"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/vault"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type testMockClient struct {
	secrets        []vault.VaultSecret
	deletionPolicy synv1alpha1.DeletionPolicy
}

func (m *testMockClient) AddSecrets(secrets []vault.VaultSecret) error { return nil }

func (m *testMockClient) RemoveSecrets(secrets []vault.VaultSecret) error { return nil }

func (m *testMockClient) SetDeletionPolicy(deletionPolicy synv1alpha1.DeletionPolicy) {
	m.deletionPolicy = deletionPolicy
}

var getVaultCases = map[string]struct {
	want synv1alpha1.DeletionPolicy
	args args
}{
	"without specific deletion policy": {
		want: getDefaultDeletionPolicy(),
		args: args{
			cluster: &synv1alpha1.Cluster{},
			data: &ExecutionContext{
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
			data: &ExecutionContext{
				Log: zap.New(),
			},
		},
	},
}

func Test_getVaultClient(t *testing.T) {
	// ensure that it isn't set to anything from previous tests
	os.Unsetenv("DEFAULT_DELETION_POLICY")

	mockClient := &testMockClient{}

	vault.SetCustomClient(mockClient)

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
	"noop": {
		want: getDefaultDeletionPolicy(),
		args: args{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					DeletionPolicy: getDefaultDeletionPolicy(),
				},
			},
			data: &ExecutionContext{},
		},
	},
	"archive": {
		want: synv1alpha1.ArchivePolicy,
		args: args{
			cluster: &synv1alpha1.Cluster{
				Spec: synv1alpha1.ClusterSpec{
					DeletionPolicy: synv1alpha1.ArchivePolicy,
				},
			},
			data: &ExecutionContext{
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
			data: &ExecutionContext{
				Deleted: true,
			},
		},
	},
}

func Test_handleVaultDeletion(t *testing.T) {
	// ensure that it isn't set to anything from previous tests
	os.Unsetenv("DEFAULT_DELETION_POLICY")

	mockClient := &testMockClient{
		deletionPolicy: getDefaultDeletionPolicy(),
	}

	vault.SetCustomClient(mockClient)

	for name, tt := range handleVaultDeletionCases {
		t.Run(name, func(t *testing.T) {

			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.cluster,
			})

			got := handleVaultDeletion(tt.args.cluster, tt.args.data)
			assert.NoError(t, got.Err)
			assert.Equal(t, tt.want, mockClient.deletionPolicy)
		})
	}
}
