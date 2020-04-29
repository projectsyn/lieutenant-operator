package vault

import (
	"os"
	"path"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	"github.com/go-logr/logr"
	"github.com/hashicorp/vault/api"
)

const (
	tokenName    = "token"
	k8sTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var (
	// we're keeping a global client
	instanceClient VaultClient
)

type VaultClient interface {
	SetToken(secretPath string, token string, log logr.Logger) error
}

type BankVaultClient struct {
	client       *vault.Client
	secretEngine string
}

// SetCustomClient is used if a custom client needs to be used. Currently only
// used for testing.
func SetCustomClient(c VaultClient) {
	instanceClient = c
}

// NewClient returns the default VaultClient implementation, ready to be used.
// It automatically detects, if there was a Vault token provided or if it's
// running withing kubernetes.
func NewClient() (VaultClient, error) {

	if instanceClient != nil {
		return instanceClient, nil
	}

	rawClient, err := api.NewClient(&api.Config{
		Address: os.Getenv(api.EnvVaultAddress),
	})
	if err != nil {
		return nil, err
	}

	// if we're not in a k8s pod we'll use provided TOKEN env var
	if _, err = os.Stat(k8sTokenPath); os.IsNotExist(err) {
		rawClient.SetToken(os.Getenv(api.EnvVaultToken))
	}

	client, err := vault.NewClientFromRawClient(rawClient, vault.ClientRole("lieutenant-operator"))
	if err != nil {
		return nil, err
	}

	instanceClient = &BankVaultClient{
		client:       client,
		secretEngine: "kv",
	}

	return instanceClient, nil
}

// SetToken saves the token in Vault, the path should have the form
// tenant/cluster to work properly. It will check if the token exists and
// re-apply it if not.
func (b *BankVaultClient) SetToken(secretPath, token string, log logr.Logger) error {

	queryPath := path.Join(b.secretEngine, "data", secretPath)

	secret, err := b.client.RawClient().Logical().Read(queryPath)
	if err != nil {
		return err
	}

	if secret == nil {
		log.WithName("vault").Info("does not yet exist, creating", "name", secretPath)
		secret = &api.Secret{}
		secret.Data = vault.NewData(0, map[string]interface{}{
			tokenName: token,
		})
		_, err = b.client.RawClient().Logical().Write(queryPath, secret.Data)
		return err
	}

	secretData := secret.Data["data"].(map[string]interface{})

	if secretData[tokenName] != token {

		log.WithName("vault").Info("secrets don't match, re-applying")

		secretData[tokenName] = token

		_, err = b.client.RawClient().Logical().Write(queryPath, secret.Data)
	}

	return err
}
