package tenant

import (
	"fmt"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func createServiceAccount(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	sa, err := newServiceAccount(data.Client.Scheme(), tenant)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to create ServiceAccount for tenant: %w", err)}
	}

	err = data.Client.Create(data.Context, sa)
	if err != nil && !errors.IsAlreadyExists(err) {
		return pipeline.Result{Err: fmt.Errorf("create serviceaccount: %w", err)}
	}

	if data.CreateSATokenSecret {
		secret, err := newServiceAccountTokenSecret(data.Client.Scheme(), tenant)
		if err != nil {
			return pipeline.Result{Err: fmt.Errorf("failed to create ServiceAccount token secret for tenant: %w", err)}
		}

		err = data.Client.Create(data.Context, secret)
		if err != nil && !errors.IsAlreadyExists(err) {
			return pipeline.Result{Err: fmt.Errorf("create serviceaccount token secret: %w", err)}
		}
	}

	return pipeline.Result{}
}

func newServiceAccount(scheme *runtime.Scheme, tenant *synv1alpha1.Tenant) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: tenant.Namespace,
		},
	}
	setManagedByLabel(sa)
	if err := controllerutil.SetControllerReference(tenant, sa, scheme); err != nil {
		return nil, err
	}

	return sa, nil
}

func newServiceAccountTokenSecret(scheme *runtime.Scheme, tenant *synv1alpha1.Tenant) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: tenant.Namespace,
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: tenant.Name,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	setManagedByLabel(secret)
	if err := controllerutil.SetControllerReference(tenant, secret, scheme); err != nil {
		return nil, err
	}

	return secret, nil
}
