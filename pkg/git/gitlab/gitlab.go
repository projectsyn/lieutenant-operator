package gitlab

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/projectsyn/lieutenant-operator/pkg/git/helpers"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"

	"github.com/go-logr/logr"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"

	"github.com/xanzy/go-gitlab"
)

func init() {
	manager.Register(&Gitlab{})
}

// Gitlab holds the necessary information to communincate with a Gitlab server
type Gitlab struct {
	url         *url.URL
	client      *gitlab.Client
	credentials manager.Credentials
	project     *gitlab.Project
	deployKeys  map[string]synv1alpha1.DeployKey
	log         logr.Logger
}

// Create will create a new Gitlab project
func (g *Gitlab) Create() error {

	namespace, pname := filepath.Split(g.url.Path)

	nsID, err := g.getNamespaceID(namespace)
	if errResp, ok := err.(*gitlab.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
		return g.createNamespace(namespace)
	} else if err != nil {
		return err
	}

	projectOptions := &gitlab.CreateProjectOptions{
		Path:        &pname,
		NamespaceID: nsID,
	}

	project, _, err := g.client.Projects.CreateProject(projectOptions)
	if err != nil {
		return err
	}

	g.project = project
	return g.setDeployKeys(g.deployKeys, false)
}

// Read reads the repository from the gitlab server and sets the object's state accordingly
func (g *Gitlab) Read() error {
	return g.getProject()
}

// Update will overwrite the deployment keys on the endpoint that differ from the local ones. Currently it will not
// touch any additional keys that may have been added to the repository.
func (g *Gitlab) Update() (bool, error) {
	remoteKeys, err := g.getDeployKeys()
	if err != nil {
		return false, err
	}

	deltaKeys := helpers.CompareKeys(g.deployKeys, remoteKeys)

	if len(deltaKeys) > 0 {
		err = g.setDeployKeys(deltaKeys, true)
		if err != nil {
			return false, err
		}
	}

	deleteKeys := helpers.CompareKeys(remoteKeys, g.deployKeys)

	if len(deleteKeys) > 0 {
		err = g.removeDeployKeys(deleteKeys)
		if err != nil {
			return false, err
		}
	}

	return len(deltaKeys) > 0, nil
}

func (g *Gitlab) removeDeployKeys(deleteKeys map[string]synv1alpha1.DeployKey) error {
	existingKeys, _, err := g.client.DeployKeys.ListProjectDeployKeys(g.project.ID, &gitlab.ListProjectDeployKeysOptions{})

	if err != nil {
		return err
	}

	for _, key := range existingKeys {
		if _, ok := deleteKeys[key.Title]; ok {
			g.log.Info(fmt.Sprintf("removing key %v; existing on repo but not in CRDs", key.Title))
			g.deleteKey(key)
		}
	}

	return err
}

// Delete deletes the project handled by the gitlab instance
func (g *Gitlab) Delete() error {
	// make sure to have the latest version of the project
	err := g.getProject()
	if err != nil {
		return err
	}

	if g.project == nil {
		return fmt.Errorf("no project %v found, can't delete", g.url.Path)
	}

	_, err = g.client.Projects.DeleteProject(g.project.ID)

	return err
}

// getProject fetches all project information from the gitlab instance
func (g *Gitlab) getProject() error {

	// we need to remove the leading slash from the path or else the HTML encode
	// for the getproject call will fail...
	path := strings.Replace(g.url.Path, "/", "", 1)
	project, _, err := g.client.Projects.GetProject(path, &gitlab.GetProjectOptions{})
	if err != nil {
		return err
	}

	g.project = project

	return nil
}

// Connect creates the Gitlab client
func (g *Gitlab) Connect() error {
	g.client = gitlab.NewClient(nil, g.credentials.Token)
	return g.client.SetBaseURL(g.url.Scheme + "://" + g.url.Host)
}

// FullURL returns the complete url of this git repository
func (g *Gitlab) FullURL() *url.URL {
	return g.url
}

// IsType determines if the given url can be handled by this concrete implementation.
// This is done by a simple http query to the login page of gitlab. If any errors occur anywhere
// it will return false.
func (g *Gitlab) IsType(URL string) (bool, error) {

	parsedURL, err := url.Parse(URL)
	if err != nil {
		return false, err
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	gitlabURL := parsedURL.Scheme + "://" + parsedURL.Host + "/users/sign_in'"

	resp, err := httpClient.Get(gitlabURL)
	if err != nil {
		return false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("status code was %d", resp.StatusCode)
	}

	return true, nil
}

// Type returns the type of this repo instance
func (g *Gitlab) Type() string {
	return string(synv1alpha1.GitLab)
}

// New returns a new and empty Gitlab implmentation
func (g *Gitlab) New(URL string, options manager.RepoOptions) (manager.Repo, error) {

	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	return &Gitlab{
		credentials: options.Credentials,
		url:         parsedURL,
		deployKeys:  options.DeployKeys,
		log:         options.Logger,
	}, nil
}

func (g *Gitlab) getNamespaceID(namespace string) (*int, error) {

	namespace = strings.Replace(namespace, "/", "", -1)
	fetchedNamespace, _, err := g.client.Namespaces.GetNamespace(namespace)
	if err != nil {
		return nil, err
	}

	return &fetchedNamespace.ID, nil
}

func (g *Gitlab) createNamespace(namespace string) error {
	// remove all slashes
	namespace = strings.Replace(namespace, "/", "", -1)
	group := &gitlab.CreateGroupOptions{
		Name: &namespace,
		Path: &namespace,
	}
	_, _, err := g.client.Groups.CreateGroup(group)
	return err
}

// setDeployKeys will update the keys on the gitlab instance. If force is set the key will be deleted beforehand.
func (g *Gitlab) setDeployKeys(localKeys map[string]synv1alpha1.DeployKey, force bool) error {
	errorCount := 0
	existingKeys, _, err := g.client.DeployKeys.ListProjectDeployKeys(g.project.ID, &gitlab.ListProjectDeployKeysOptions{})
	if err != nil {
		return err
	}
	for k, v := range localKeys {
		mergedKey := v.Type + " " + v.Key
		keyOpts := &gitlab.AddDeployKeyOptions{
			Title:   &k,
			Key:     &mergedKey,
			CanPush: &v.WriteAccess,
		}

		_, _, err := g.client.DeployKeys.AddDeployKey(g.project.ID, keyOpts)
		if err != nil {
			g.log.Error(err, "failed adding key to repository "+g.project.Name)
			errorCount++
		} else if force {
			g.overwriteKey(existingKeys, k)
		}
	}
	if errorCount > 0 {
		return fmt.Errorf("%v keys failed to be added", errorCount)
	}
	return nil
}

func (g *Gitlab) overwriteKey(existingKeys []*gitlab.DeployKey, key string) {
	// unfortunately there's no way via the API to update a key, so we have to delete and recreate it, when it differs from
	// the yaml in the k8s cluster.
	for _, deleteKey := range existingKeys {
		if deleteKey.Title == key {
			g.log.Info("forcing re-creation of key " + key)
			g.deleteKey(deleteKey)
		}
	}
}

func (g *Gitlab) deleteKey(deleteKey *gitlab.DeployKey) {
	_, err := g.client.DeployKeys.DeleteDeployKey(g.project.ID, deleteKey.ID)
	if err != nil {
		g.log.Error(err, "could not delete existing deploy key "+deleteKey.Title)
	}
}

func (g *Gitlab) getDeployKeys() (map[string]synv1alpha1.DeployKey, error) {
	remoteKeys, _, err := g.client.DeployKeys.ListProjectDeployKeys(g.project.ID, &gitlab.ListProjectDeployKeysOptions{})
	if err != nil {
		return nil, err
	}

	deployKeys := make(map[string]synv1alpha1.DeployKey)
	for _, key := range remoteKeys {
		splittedKey := strings.Split(key.Key, " ")

		deployKeys[key.Title] = synv1alpha1.DeployKey{
			Type:        splittedKey[0],
			Key:         splittedKey[1],
			WriteAccess: *key.CanPush,
		}
	}

	return deployKeys, nil
}
