package cluster

import (
	"context"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createClusterRBAC(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	objMeta := metav1.ObjectMeta{
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(obj, synv1alpha1.SchemeBuilder.GroupVersion.WithKind("Cluster"))},
	}
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: objMeta}
	role := &rbacv1.Role{
		ObjectMeta: objMeta,
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{synv1alpha1.GroupVersion.Group},
			Resources:     []string{"clusters", "clusters/status"},
			Verbs:         []string{"get", "update"},
			ResourceNames: []string{obj.GetName()},
		}},
	}
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: objMeta,
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: role.Name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccount.Name,
			Namespace: serviceAccount.Namespace,
		}},
	}
	for _, item := range []client.Object{serviceAccount, role, roleBinding} {
		found := item.DeepCopyObject().(client.Object)
		err := data.Client.Get(context.TODO(), client.ObjectKeyFromObject(item), found)
		if errors.IsNotFound(err) {
			if err := data.Client.Create(context.TODO(), item); err != nil && !errors.IsAlreadyExists(err) {
				return pipeline.Result{Err: err}
			}
		} else if err != nil {
			return pipeline.Result{Err: err}
		}
		if err := data.Client.Update(context.TODO(), item); err != nil {
			return pipeline.Result{Err: err}
		}
	}
	return pipeline.Result{}
}
