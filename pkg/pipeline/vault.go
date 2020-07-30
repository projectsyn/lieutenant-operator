package pipeline

import (
	"context"
	"fmt"
	"path"
	"sort"

	"github.com/projectsyn/lieutenant-operator/pkg/helpers"
	"github.com/projectsyn/lieutenant-operator/pkg/vault"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func getVaultClient(obj PipelineObject, data *ExecutionContext) (vault.VaultClient, error) {
	deletionPolicy := obj.GetDeletionPolicy()
	if deletionPolicy == "" {
		deletionPolicy = getDefaultDeletionPolicy()
	}

	return vault.NewClient(deletionPolicy, data.Log)
}

func createOrUpdateVault(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	secretPath := path.Join(obj.GetTenantRef().Name, obj.GetObjectMeta().GetName(), "steward")

	token, err := GetServiceAccountToken(obj.GetObjectMeta(), data)
	if err != nil {
		return ExecutionResult{Err: err}
	}

	vaultClient, err := getVaultClient(obj, data)

	err = vaultClient.AddSecrets([]vault.VaultSecret{{Path: secretPath, Value: token}})
	if err != nil {
		return ExecutionResult{Err: err}
	}

	return ExecutionResult{}
}

func GetServiceAccountToken(instance metav1.Object, data *ExecutionContext) (string, error) {
	secrets := &corev1.SecretList{}

	err := data.Client.List(context.TODO(), secrets)
	if err != nil {
		return "", err
	}

	sortSecrets := helpers.SecretSortList(*secrets)

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

func handleVaultDeletion(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	repoName := types.NamespacedName{
		Name:      obj.GetTenantRef().Name,
		Namespace: obj.GetObjectMeta().GetNamespace(),
	}

	secretPath := path.Join(repoName.Name, obj.GetObjectMeta().GetName(), "steward")

	if data.Deleted {
		vaultClient, err := getVaultClient(obj, data)
		if err != nil {
			return ExecutionResult{Err: err}
		}
		err = vaultClient.RemoveSecrets([]vault.VaultSecret{{Path: path.Dir(secretPath), Value: ""}})
		if err != nil {
			return ExecutionResult{Err: err}
		}
	}
	return ExecutionResult{}
}
