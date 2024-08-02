package controllers_test

import (
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func preparePipelineTestClient(t *testing.T, initObjs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(synv1alpha1.AddToScheme(scheme))

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjs...).
		WithStatusSubresource(&synv1alpha1.Tenant{}).
		Build()

	return client
}

func requestFor(obj client.Object) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func findEnvVar(name string, list []synv1alpha1.EnvVar) (v synv1alpha1.EnvVar, found bool) {
	for _, envvar := range list {
		if envvar.Name == name {
			return envvar, true
		}
	}
	return synv1alpha1.EnvVar{}, false
}
