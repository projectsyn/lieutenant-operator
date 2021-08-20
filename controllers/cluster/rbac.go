package cluster

import (
	"context"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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
	if err := createClusterSA(data.Context, data.Client, objMeta); err != nil {
		return pipeline.Result{Err: err}
	}
	if err := createOrUpdateClusterRole(data.Context, data.Client, objMeta); err != nil {
		return pipeline.Result{Err: err}
	}
	if err := createOrUpdateClusterRoleBinding(data.Context, data.Client, objMeta); err != nil {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}

func createClusterSA(ctx context.Context, c client.Client, objMeta metav1.ObjectMeta) error {
	sa := &corev1.ServiceAccount{ObjectMeta: objMeta}
	if err := c.Create(ctx, sa); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createOrUpdateClusterRole(ctx context.Context, c client.Client, objMeta metav1.ObjectMeta) error {
	role := &rbacv1.Role{
		ObjectMeta: objMeta,
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{synv1alpha1.GroupVersion.Group},
			Resources:     []string{"clusters", "clusters/status"},
			Verbs:         []string{"get", "update"},
			ResourceNames: []string{objMeta.Name},
		}},
	}
	found := &rbacv1.Role{}

	err := c.Get(ctx, client.ObjectKeyFromObject(role), found)
	if err != nil {
		if errors.IsNotFound(err) {
			err = c.Create(ctx, role)
		}
		return err
	}
	if !equality.Semantic.DeepEqual(found.Rules, role.Rules) {
		found.Rules = role.Rules
		return c.Update(ctx, found)
	}
	return nil
}

func createOrUpdateClusterRoleBinding(ctx context.Context, c client.Client, objMeta metav1.ObjectMeta) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: objMeta,
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: objMeta.Name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      objMeta.Name,
			Namespace: objMeta.Namespace,
		}},
	}
	found := &rbacv1.RoleBinding{}

	err := c.Get(ctx, client.ObjectKeyFromObject(roleBinding), found)
	if err != nil {
		if errors.IsNotFound(err) {
			err = c.Create(ctx, roleBinding)
		}
		return err
	}

	if !equality.Semantic.DeepEqual(found.RoleRef, roleBinding.RoleRef) ||
		!equality.Semantic.DeepEqual(found.Subjects, roleBinding.Subjects) {
		found.RoleRef = roleBinding.RoleRef
		found.Subjects = roleBinding.Subjects
		return c.Update(ctx, found)
	}
	return nil
}
