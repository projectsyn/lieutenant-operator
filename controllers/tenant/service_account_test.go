package tenant

import (
	"context"
	"errors"
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Test_createTenantServiceAccountNoToken(t *testing.T) {
	ctx := context.Background()
	c := prepareClient(t, testCfg{})
	data := pipeline.Context{
		Context:             ctx,
		Client:              c,
		Log:                 log.FromContext(ctx),
		FinalizerName:       "",
		Reconciler:          &mockReconciler{},
		CreateSATokenSecret: false,
	}

	res := createServiceAccount(&synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tenant",
			Namespace: "lieutenant",
		},
	}, &data)

	assert.NoError(t, res.Err)

	sa := &corev1.ServiceAccount{}
	err := c.Get(ctx, types.NamespacedName{Name: "test-tenant", Namespace: "lieutenant"}, sa)
	assert.NoError(t, err)

	secret := &corev1.Secret{}
	err = c.Get(ctx, types.NamespacedName{Name: "test-tenant", Namespace: "lieutenant"}, secret)
	assert.True(t, apierrors.IsNotFound(err))
}

func Test_createTenantServiceAccountWithToken(t *testing.T) {
	ctx := context.Background()
	c := prepareClient(t, testCfg{})
	data := pipeline.Context{
		Context:             ctx,
		Client:              c,
		Log:                 log.FromContext(ctx),
		FinalizerName:       "",
		Reconciler:          &mockReconciler{},
		CreateSATokenSecret: true,
	}

	res := createServiceAccount(&synv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tenant",
			Namespace: "lieutenant",
		},
	}, &data)

	assert.NoError(t, res.Err)

	sa := &corev1.ServiceAccount{}
	err := c.Get(ctx, types.NamespacedName{Name: "test-tenant", Namespace: "lieutenant"}, sa)
	assert.NoError(t, err)

	secret := &corev1.Secret{}
	err = c.Get(ctx, types.NamespacedName{Name: "test-tenant", Namespace: "lieutenant"}, secret)
	assert.NoError(t, err)
}

type mockReconciler struct{}

func (r *mockReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, errors.New("mockReconciler.Reconcile() not implemented")
}

type testCfg struct {
	obj []client.Object
}

func prepareClient(t *testing.T, cfg testCfg) client.Client {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(synv1alpha1.AddToScheme(scheme))

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg.obj...).
		Build()

	return client
}
