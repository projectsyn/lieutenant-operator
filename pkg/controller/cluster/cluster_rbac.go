package cluster

import (
	"context"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (r *ReconcileCluster) createClusterRBAC(cluster synv1alpha1.Cluster) error {
	objMeta := metav1.ObjectMeta{
		Name:            cluster.Name,
		Namespace:       cluster.Namespace,
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&cluster, synv1alpha1.SchemeBuilder.GroupVersion.WithKind("Cluster"))},
	}
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: objMeta}
	role := &rbacv1.Role{
		ObjectMeta: objMeta,
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{synv1alpha1.SchemeGroupVersion.Group},
			Resources:     []string{"clusters"},
			Verbs:         []string{"get", "update"},
			ResourceNames: []string{cluster.Name},
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
	for _, item := range []runtime.Object{serviceAccount, role, roleBinding} {
		if err := r.client.Create(context.TODO(), item); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}
