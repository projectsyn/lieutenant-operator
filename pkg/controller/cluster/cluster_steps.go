package cluster

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	roleUtil "github.com/projectsyn/lieutenant-operator/pkg/role"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	clusterClassContent = `classes:
- %s.%s
`
)

func createClusterRBAC(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	objMeta := metav1.ObjectMeta{
		Name:            obj.GetObjectMeta().GetName(),
		Namespace:       obj.GetObjectMeta().GetNamespace(),
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(obj.GetObjectMeta(), synv1alpha1.SchemeBuilder.GroupVersion.WithKind("Cluster"))},
	}
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: objMeta}
	role := &rbacv1.Role{
		ObjectMeta: objMeta,
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{synv1alpha1.SchemeGroupVersion.Group},
			Resources:     []string{"clusters"},
			Verbs:         []string{"get", "update"},
			ResourceNames: []string{obj.GetObjectMeta().GetName()},
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
		if err := data.Client.Create(context.TODO(), item); err != nil && !errors.IsAlreadyExists(err) {
			return pipeline.Result{Err: err}
		}
	}
	return pipeline.Result{}
}

func setBootstrapToken(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	instance, ok := obj.(*synv1alpha1.Cluster)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("%s is not a cluster object", obj.GetObjectMeta().GetName())}
	}

	if instance.Status.BootstrapToken == nil {
		data.Log.Info("Adding status to Cluster object")
		err := newClusterStatus(instance)
		if err != nil {
			return pipeline.Result{Err: err}
		}
	}

	if time.Now().After(instance.Status.BootstrapToken.ValidUntil.Time) {
		instance.Status.BootstrapToken.TokenValid = false
	}

	return pipeline.Result{}
}

//newClusterStatus will create a default lifetime of 30 minutes if it wasn't set in the object.
func newClusterStatus(cluster *synv1alpha1.Cluster) error {
	parseTime := "24h"
	if cluster.Spec.TokenLifeTime != "" {
		parseTime = cluster.Spec.TokenLifeTime
	}

	duration, err := time.ParseDuration(parseTime)
	if err != nil {
		return err
	}

	validUntil := time.Now().Add(duration)

	token, err := generateToken()
	if err != nil {
		return err
	}

	cluster.Status.BootstrapToken = &synv1alpha1.BootstrapToken{
		Token:      token,
		ValidUntil: metav1.NewTime(validUntil),
		TokenValid: true,
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), err
}

func setTenantOwner(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant := &synv1alpha1.Tenant{}
	tenantName := types.NamespacedName{Name: obj.GetTenantRef().Name, Namespace: obj.GetObjectMeta().GetNamespace()}

	err := data.Client.Get(context.TODO(), tenantName, tenant)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	obj.GetObjectMeta().SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(tenant.GetObjectMeta(), tenant.GroupVersionKind()),
	})

	return pipeline.Result{}
}

func applyClusterTemplateFromTenant(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	nsName := types.NamespacedName{Name: obj.GetTenantRef().Name, Namespace: obj.GetObjectMeta().GetNamespace()}

	tenant := &synv1alpha1.Tenant{}
	if err := data.Client.Get(context.TODO(), nsName, tenant); err != nil {
		return pipeline.Result{Err: fmt.Errorf("Couldn't find tenant: %w", err)}
	}

	instance, ok := obj.(*synv1alpha1.Cluster)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a cluster")}
	}

	if err := ApplyClusterTemplate(instance, tenant); err != nil {
		return pipeline.Result{Err: err}
	}
	return pipeline.Result{}
}

func clusterUpdateRole(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	cluster, ok := obj.(*synv1alpha1.Cluster)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a cluster")}
	}

	name := types.NamespacedName{Name: cluster.Spec.TenantRef.Name, Namespace: cluster.Namespace}
	role := &rbacv1.Role{}
	if err := data.Client.Get(context.TODO(), name, role); err != nil {
		if errors.IsNotFound(err) || runtime.IsNotRegisteredError(err) {
			// The absence of a role is not an error.
			// The role might not yet be created. It gets update on a future reconciliation.
			data.Log.Info("No role found to update.")
			return pipeline.Result{}
		}
		return pipeline.Result{Err: fmt.Errorf("failed to get role for cluster: %v", err)}
	}

	updated := false

	if data.Deleted {
		updated = roleUtil.RemoveResourceNames(role, cluster.Name)
	} else {
		updated = roleUtil.AddResourceNames(role, cluster.Name)
	}

	if !updated {
		return pipeline.Result{}
	}

	if err := data.Client.Update(context.TODO(), role); err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to update role for cluster: %v", err)}
	}

	return pipeline.Result{}
}
