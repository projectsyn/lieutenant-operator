package vault

import (
	"fmt"
	"path"
	"sort"

	"github.com/projectsyn/lieutenant-operator/collection"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func getVaultClient(obj pipeline.Object, data *pipeline.Context) (VaultClient, error) {
	deletionPolicy := data.DefaultDeletionPolicy

	return NewClient(deletionPolicy, data.Log)
}

func CreateOrUpdateVault(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	if !data.UseVault {
		return pipeline.Result{}
	}

	secretPath := path.Join(obj.GetTenantRef().Name, obj.GetName(), "steward")

	token, err := GetServiceAccountToken(obj, data)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("get SA token: %w", err)}
	}

	vaultClient, err := getVaultClient(obj, data)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("get vault client: %w", err)}
	}

	err = vaultClient.AddSecrets([]VaultSecret{{Path: secretPath, Value: token}})
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("add vault secret '%s': %w", secretPath, err)}
	}

	return pipeline.Result{}
}

func GetServiceAccountToken(instance metav1.Object, data *pipeline.Context) (string, error) {
	secrets := &corev1.SecretList{}

	err := data.Client.List(data.Context, secrets)
	if err != nil {
		return "", err
	}

	sortSecrets := collection.SecretSortList(*secrets)

	sort.Sort(sort.Reverse(sortSecrets))

	for _, secret := range sortSecrets.Items {

		if secret.Type != corev1.SecretTypeServiceAccountToken {
			continue
		}

		if secret.Annotations[corev1.ServiceAccountNameKey] == instance.GetName() {
			if string(secret.Data["token"]) == "" {
				// We'll skip the secrets if the token is not yet populated.
				continue
			}
			return string(secret.Data["token"]), nil
		}
	}

	return "", fmt.Errorf("no matching secrets found")
}

func HandleVaultDeletion(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	if !data.UseVault {
		return pipeline.Result{}
	}

	repoName := types.NamespacedName{
		Name:      obj.GetTenantRef().Name,
		Namespace: obj.GetNamespace(),
	}

	secretPath := path.Join(repoName.Name, obj.GetName(), "steward")

	if data.Deleted {
		vaultClient, err := getVaultClient(obj, data)
		if err != nil {
			return pipeline.Result{Err: fmt.Errorf("get vault client: %w", err)}
		}
		err = vaultClient.RemoveSecrets([]VaultSecret{{Path: path.Dir(secretPath), Value: ""}})
		if err != nil {
			return pipeline.Result{Err: fmt.Errorf("remove secret: %w", err)}
		}
	}
	return pipeline.Result{}
}
