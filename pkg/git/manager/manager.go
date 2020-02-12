package manager

import (
	"fmt"
	"net/url"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"

	"github.com/go-logr/logr"
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
type RepoOptions struct {
	Credentials Credentials
	DeployKeys  map[string]synv1alpha1.DeployKey
	Logger      logr.Logger
	URL         *url.URL
	Path        string
	RepoName    string
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
	// FullURL returns the full url to the repository
	FullURL() *url.URL
	Create() error
	// Update will enforce the defined keys to be deployed to the repository, it will return true if an actual change
	// happened
	Update() (bool, error)
	// Read will read the repository and populate it with the deployed keys. It will throw an
	// error if the repo is not found on the server.
	Read() error
	Delete() error
	Connect() error
}

// Implementation is a set of functions needed to get the right git implementation
// for the given URL.
type Implementation interface {
	// IsType returns true, if the given URL is handleable by the given implementation (Github,Gitlab, etc.)
	IsType(URL *url.URL) (bool, error)
	// New returns a clean new Repo implementation with the given URL
	New(options RepoOptions) (Repo, error)
}
