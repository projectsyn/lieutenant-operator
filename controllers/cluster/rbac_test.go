package cluster

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_CreateClusterSANoToken(t *testing.T) {
	c := prepareClient(t, testCfg{})

	objMeta := metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: "lieutenant",
	}

	ctx := context.Background()

	err := createClusterSA(ctx, c, objMeta, false)
	assert.NoError(t, err)

	sa := &corev1.ServiceAccount{}
	err = c.Get(ctx, types.NamespacedName{Name: "test-cluster", Namespace: "lieutenant"}, sa)
	assert.NoError(t, err)

	secret := &corev1.Secret{}
	err = c.Get(ctx, types.NamespacedName{Name: "test-cluster", Namespace: "lieutenant"}, secret)
	assert.True(t, errors.IsNotFound(err))
}

func Test_CreateClusterSACreateToken(t *testing.T) {
	c := prepareClient(t, testCfg{})

	objMeta := metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: "lieutenant",
	}

	ctx := context.Background()

	err := createClusterSA(ctx, c, objMeta, true)
	assert.NoError(t, err)

	sa := &corev1.ServiceAccount{}
	err = c.Get(ctx, types.NamespacedName{Name: "test-cluster", Namespace: "lieutenant"}, sa)
	assert.NoError(t, err)

	secret := &corev1.Secret{}
	err = c.Get(ctx, types.NamespacedName{Name: "test-cluster", Namespace: "lieutenant"}, secret)
	assert.NoError(t, err)

	assert.Equal(t, corev1.SecretTypeServiceAccountToken, secret.Type)
	assert.Equal(t, map[string]string{corev1.ServiceAccountNameKey: "test-cluster"}, secret.ObjectMeta.Annotations)
}

type testCfg struct {
	obj []client.Object
}

func prepareClient(t *testing.T, cfg testCfg) client.Client {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg.obj...).
		Build()

	return client
}
