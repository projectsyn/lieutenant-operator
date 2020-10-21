package pipeline

import (
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

func Test_getVaultClient(t *testing.T) {
	type args struct {
		obj  *synv1alpha1.Cluster
		data *ExecutionContext
	}
	tests := []struct {
		name    string
		args    args
		want    synv1alpha1.DeletionPolicy
		wantErr bool
	}{
		{
			name: "without specific deletion policy",
			want: getDefaultDeletionPolicy(),
			args: args{
				obj: &synv1alpha1.Cluster{},
				data: &ExecutionContext{
					Log: zap.New(),
				},
			},
		},
		{
			name: "specific deletion policy",
			want: synv1alpha1.DeletePolicy,
			args: args{
				obj: &synv1alpha1.Cluster{
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

	mockClient := &testMockClient{}

	vault.SetCustomClient(mockClient)

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			_, err := getVaultClient(tt.args.obj, tt.args.data)
			assert.NoError(t, err)

			assert.Equal(t, tt.want, mockClient.deletionPolicy)

		})
	}
}

func Test_handleVaultDeletion(t *testing.T) {
	type args struct {
		obj  *synv1alpha1.Cluster
		data *ExecutionContext
	}
	tests := []struct {
		name string
		args args
		want synv1alpha1.DeletionPolicy
	}{
		{
			name: "noop",
			want: getDefaultDeletionPolicy(),
			args: args{
				obj: &synv1alpha1.Cluster{
					Spec: synv1alpha1.ClusterSpec{
						DeletionPolicy: getDefaultDeletionPolicy(),
					},
				},
				data: &ExecutionContext{},
			},
		},
		{
			name: "archive",
			want: synv1alpha1.ArchivePolicy,
			args: args{
				obj: &synv1alpha1.Cluster{
					Spec: synv1alpha1.ClusterSpec{
						DeletionPolicy: synv1alpha1.ArchivePolicy,
					},
				},
				data: &ExecutionContext{
					Deleted: true,
				},
			},
		},
		{
			name: "delete",
			want: synv1alpha1.DeletePolicy,
			args: args{
				obj: &synv1alpha1.Cluster{
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

	mockClient := &testMockClient{
		deletionPolicy: getDefaultDeletionPolicy(),
	}

	vault.SetCustomClient(mockClient)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.obj,
			})

			got := handleVaultDeletion(tt.args.obj, tt.args.data)
			assert.NoError(t, got.Err)
			assert.Equal(t, tt.want, mockClient.deletionPolicy)
		})
	}
}
