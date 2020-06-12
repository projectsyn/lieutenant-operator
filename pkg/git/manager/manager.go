package manager

import (
	"context"
	"fmt"
	"net/url"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
)

const (
	// SecretTokenName is the name of the secret entry containing the token
	SecretTokenName = "token"
	// SecretHostKeysName is the name of the secret entry containing the SSH host keys
	SecretHostKeysName = "hostKeys"
	// SecretEndpointName is the name of the secret entry containing the api endpoint
	SecretEndpointName = "endpoint"
	// DeletionMagicString defines when a file should be deleted from the repository
	//TODO it will be replaced with somethin better in the futur TODO
	DeletionMagicString = "{delete}"
)

var (
	// implementations holds each a copy of the registered Git implementation
	implementations []Implementation
)

// Register adds a type to the list of supported Git implementations.
func Register(i Implementation) {
	implementations = append(implementations, i)
}

// NewRepo returns a Repo object that can handle the specific URL
func NewRepo(opts RepoOptions) (Repo, error) {

	for _, imp := range implementations {
		if exists, err := imp.IsType(opts.URL); exists {
			newImp, err := imp.New(opts)
			if err != nil {
				return nil, err
			}
			return newImp, nil
		} else {
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("no git implementation found for given url: %v ", opts.URL.String())
}

// RepoOptions hold the options for creating a repository. The credentials are required to work. The deploykeys are
// optional but desired.
// If not provided DeletionPolicy will default to archive.
type RepoOptions struct {
	Credentials    Credentials
	DeployKeys     map[string]synv1alpha1.DeployKey
	Logger         logr.Logger
	URL            *url.URL
	Path           string
	RepoName       string
	DisplayName    string
	TemplateFiles  map[string]string
	DeletionPolicy synv1alpha1.DeletionPolicy
}

// Credentials holds the authentication information for the API. Most of the times this
// is just a token.
type Credentials struct {
	Token string
}

// Repo represents a repository that lives on some git server
type Repo interface {
	// Type returns the type of the repo
	Type() string
	// FullURL returns the full url to the repository for ssh pulling
	FullURL() *url.URL
	Create() error
	// Update will enforce the defined keys to be deployed to the repository, it will return true if an actual change
	// happened
	Update() (bool, error)
	// Read will read the repository and populate it with the deployed keys. It will throw an
	// error if the repo is not found on the server.
	Read() error
	// Remove will remove the git project according to the recycle policy
	Remove() error
	Connect() error
	// CommitTemplateFiles uploads given files to the repository.
	// files that contain exactly the deletion magic string should be removed
	// when calling this function. TODO: will be replaced with something better in the future.
	CommitTemplateFiles() error
}

// Implementation is a set of functions needed to get the right git implementation
// for the given URL.
type Implementation interface {
	// IsType returns true, if the given URL is handleable by the given implementation (Github,Gitlab, etc.)
	IsType(URL *url.URL) (bool, error)
	// New returns a clean new Repo implementation with the given URL
	New(options RepoOptions) (Repo, error)
}

// CommitFile contains all information about a file that should be committed to git
// TODO migrate to the CRDs in the future.
type CommitFile struct {
	FileName string
	Content  string
	Delete   bool
}

// GetGitClient will return a git client from a provided template. This does a lot more
// plumbing than the simple NewClient() call. If you're needing a git client from a
// reconcile function, this is the way to go.
func GetGitClient(instance *synv1alpha1.GitRepoTemplate, namespace string, reqLogger logr.Logger, client client.Client) (Repo, string, error) {
	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{
		Name:      instance.APISecretRef.Name,
		Namespace: namespace,
	}

	if len(instance.APISecretRef.Namespace) > 0 {
		namespacedName.Namespace = instance.APISecretRef.Namespace
	}

	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, "", fmt.Errorf("error getting git secret: %v", err)
	}

	hostKeysString := ""
	if hostKeys, ok := secret.Data[SecretHostKeysName]; ok {
		hostKeysString = string(hostKeys)
	}

	if _, ok := secret.Data[SecretEndpointName]; !ok {
		return nil, "", fmt.Errorf("secret %s does not contain endpoint data", secret.GetName())
	}

	if _, ok := secret.Data[SecretTokenName]; !ok {
		return nil, "", fmt.Errorf("secret %s does not contain token", secret.GetName())
	}

	repoURL, err := url.Parse(string(secret.Data[SecretEndpointName]) + "/" + instance.Path + "/" + instance.RepoName)

	if err != nil {
		return nil, "", err
	}

	repoOptions := RepoOptions{
		Credentials: Credentials{
			Token: string(secret.Data[SecretTokenName]),
		},
		DeployKeys:     instance.DeployKeys,
		Logger:         reqLogger,
		Path:           instance.Path,
		RepoName:       instance.RepoName,
		DisplayName:    instance.DisplayName,
		URL:            repoURL,
		TemplateFiles:  instance.TemplateFiles,
		DeletionPolicy: instance.DeletionPolicy,
	}

	repo, err := NewRepo(repoOptions)
	if err != nil {
		return nil, "", err
	}

	err = repo.Connect()

	return repo, hostKeysString, err

}
