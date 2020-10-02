package vault

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	"github.com/go-logr/logr"
	"github.com/hashicorp/vault/api"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
)

const (
	tokenName    = "token"
	k8sTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var (
	// we're keeping a global client
	instanceClient VaultClient
)

// TODO: similar map like the template files
type VaultSecret struct {
	Path  string
	Value string
}

type VaultClient interface {
	AddSecrets(secrets []VaultSecret) error
	// remove specific secret
	RemoveSecrets(secret []VaultSecret) error
}

type BankVaultClient struct {
	client         *vault.Client
	secretEngine   string
	deletionPolicy synv1alpha1.DeletionPolicy
	log            logr.Logger
}

// NewClient returns the default VaultClient implementation, ready to be used.
// It automatically detects, if there was a Vault token provided or if it's
// running withing kubernetes.
func NewClient(deletionPolicy synv1alpha1.DeletionPolicy, log logr.Logger) (VaultClient, error) {

	if instanceClient != nil {
		return instanceClient, nil
	}

	var err error
	instanceClient, err = newBankVaultClient(deletionPolicy, log)

	return instanceClient, err

}

// SetCustomClient is used if a custom client needs to be used. Currently only
// used for testing.
func SetCustomClient(c VaultClient) {
	instanceClient = c
}

func newBankVaultClient(deletionPolicy synv1alpha1.DeletionPolicy, log logr.Logger) (*BankVaultClient, error) {

	client, err := vault.NewClientFromConfig(&api.Config{
		Address: os.Getenv(api.EnvVaultAddress),
	}, vault.ClientRole("lieutenant-operator"))
	if err != nil {
		return nil, err
	}

	// if we're not in a k8s pod we'll use provided TOKEN env var
	if _, err = os.Stat(k8sTokenPath); os.IsNotExist(err) {
		client.RawClient().SetToken(os.Getenv(api.EnvVaultToken))
	}

	secretEngine := os.Getenv("VAULT_SECRET_ENGINE_PATH")
	if secretEngine == "" {
		secretEngine = "kv"
	}

	return &BankVaultClient{
		client:         client,
		secretEngine:   secretEngine,
		deletionPolicy: deletionPolicy,
		log:            log,
	}, nil

}

func (b *BankVaultClient) AddSecrets(secrets []VaultSecret) error {
	for _, secret := range secrets {
		err := b.addSecret(secret.Path, secret.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// addSecret saves the token in Vault, the path should have the form
// tenant/cluster to work properly. It will check if the token exists and
// re-apply it if not.
func (b *BankVaultClient) addSecret(secretPath, token string) error {

	queryPath := path.Join(b.secretEngine, "data", secretPath)

	secret, err := b.client.RawClient().Logical().Read(queryPath)
	if err != nil {
		return err
	}

	if secret == nil {
		b.log.WithName("vault").Info("does not yet exist, creating", "name", secretPath)
		secret = &api.Secret{}
		secret.Data = vault.NewData(0, map[string]interface{}{
			tokenName: token,
		})
		_, err = b.client.RawClient().Logical().Write(queryPath, secret.Data)
		return err
	}

	secretData, ok := secret.Data["data"].(map[string]interface{})

	if !ok {
		secretData = make(map[string]interface{})
	}

	if !ok || secretData[tokenName] != token {

		b.log.WithName("vault").Info("secrets don't match, re-applying")

		secretData[tokenName] = token

		secret.Data["data"] = secretData

		_, err = b.client.RawClient().Logical().Write(queryPath, secret.Data)
	}

	return err
}

// RemoveSecrets will remove all the keys bellow the given paths. It will list
// all secrets of in the path and delete them according to the deletion policy.
func (b *BankVaultClient) RemoveSecrets(secrets []VaultSecret) error {
	for _, secret := range secrets {
		err := b.removeSecret(secret)
		if err != nil {
			return err
		}
	}
	return nil
}

// removeSecret will remove the token according to the DeletetionPolicy
func (b *BankVaultClient) removeSecret(removeSecret VaultSecret) error {

	secrets, err := b.listSecrets(removeSecret.Path)
	if err != nil {
		return err
	}

	for _, secret := range secrets {

		sPath := path.Join(b.secretEngine, "metadata", removeSecret.Path, secret)

		s, err := b.client.RawClient().Logical().Read(sPath)
		if err != nil {
			return err
		}

		versions, err := b.getVersionList(s.Data)
		if err != nil {
			return err
		}

		switch b.deletionPolicy {
		case synv1alpha1.ArchivePolicy:
			b.log.Info("soft deleting secret", "secret", removeSecret.Path)
			err := b.deleteToken(path.Join(b.secretEngine, "delete", removeSecret.Path, secret), versions)
			if err != nil {
				return err
			}
		case synv1alpha1.DeletePolicy:
			b.log.Info("destroying secret", "secret", removeSecret.Path)
			err := b.destroyToken(path.Join(b.secretEngine, "metadata", removeSecret.Path, secret), versions)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown DeletionPolicy, skipping")
		}
	}

	return nil
}

func (b *BankVaultClient) getVersionList(data map[string]interface{}) (map[string]interface{}, error) {

	versionlist := make([]int, 0)

	if versions, ok := data["versions"]; ok {

		version, ok := versions.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("can't parse versions from secret")
		}

		for k := range version {
			if v, err := strconv.Atoi(k); err == nil {
				versionlist = append(versionlist, v)
			} else {
				return nil, err
			}

		}

	}

	sort.Ints(versionlist)

	return map[string]interface{}{"versions": versionlist}, nil
}

func (b *BankVaultClient) destroyToken(secretPath string, versions map[string]interface{}) error {
	_, err := b.client.RawClient().Logical().Delete(secretPath)
	return err
}

func (b *BankVaultClient) deleteToken(secretPath string, versions map[string]interface{}) error {
	_, err := b.client.RawClient().Logical().Write(secretPath, versions)
	return err
}

func (b *BankVaultClient) listSecrets(secretPath string) ([]string, error) {

	secrets, err := b.client.RawClient().Logical().List(path.Join(b.secretEngine, "metadata", secretPath))
	if err != nil {
		return nil, err
	}
	data, ok := secrets.Data["keys"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("list of secrets can't be decoded")
	}

	result := []string{}
	for _, secret := range data {
		s, ok := secret.(string)
		if !ok {
			return nil, fmt.Errorf("list of secrets can't be decoded")
		}
		result = append(result, s)
	}

	return result, nil

}
