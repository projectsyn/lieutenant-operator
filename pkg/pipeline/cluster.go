package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
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

func clusterSpecificSteps(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	result := createClusterRBAC(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("create cluster RBAC", result.Err)
		return result
	}

	result = checkIfDeleted(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("deletion check", result.Err)
		return result
	}

	result = setBootstrapToken(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("set bootstrap token", result.Err)
		return result
	}

	if strings.ToLower(os.Getenv("SKIP_VAULT_SETUP")) != "true" {
		result = createOrUpdateVault(obj, data)
		if resultNotOK(result) {
			result.Err = wrapError("create or update vault", result.Err)
			return result
		}

		result = handleVaultDeletion(obj, data)
		if resultNotOK(result) {
			result.Err = wrapError("delete vault entries", result.Err)
			return result
		}

	}

	result = setTenantOwner(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("set tenant owner", result.Err)
		return result
	}

	return ExecutionResult{}
}

func createClusterRBAC(obj PipelineObject, data *ExecutionContext) ExecutionResult {
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
			return ExecutionResult{Err: err}
		}
	}
	return ExecutionResult{}
}

func setBootstrapToken(obj PipelineObject, data *ExecutionContext) ExecutionResult {

	instance, ok := obj.(*synv1alpha1.Cluster)
	if !ok {
		return ExecutionResult{Err: fmt.Errorf("%s is not a cluster object", obj.GetObjectMeta().GetName())}
	}

	if instance.Status.BootstrapToken == nil {
		data.Log.Info("Adding status to Cluster object")
		err := newClusterStatus(instance)
		if err != nil {
			return ExecutionResult{Err: err}
		}
	}

	if time.Now().After(instance.Status.BootstrapToken.ValidUntil.Time) {
		instance.Status.BootstrapToken.TokenValid = false
	}

	return ExecutionResult{}

}

//newClusterStatus will create a default lifetime of 30 minutes if it wasn't set in the object.
func newClusterStatus(cluster *synv1alpha1.Cluster) error {

	parseTime := "30m"
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

func setTenantOwner(obj PipelineObject, data *ExecutionContext) ExecutionResult {

	tenant := &synv1alpha1.Tenant{}

	tenantName := types.NamespacedName{Name: obj.GetTenantRef().Name, Namespace: obj.GetObjectMeta().GetNamespace()}

	err := data.Client.Get(context.TODO(), tenantName, tenant)
	if err != nil {
		return ExecutionResult{Err: err}
	}

	obj.GetObjectMeta().SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(tenant.GetObjectMeta(), tenant.GroupVersionKind()),
	})

	return ExecutionResult{}
}
