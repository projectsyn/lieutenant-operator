package gitlab

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/projectsyn/lieutenant-operator/pkg/git/helpers"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"

	"github.com/go-logr/logr"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"

	"github.com/icza/gox/builtinx"
	"github.com/xanzy/go-gitlab"
)

func init() {
	manager.Register(&Gitlab{})
}

// Gitlab holds the necessary information to communincate with a Gitlab server
type Gitlab struct {
	client      *gitlab.Client
	credentials manager.Credentials
	project     *gitlab.Project
	deployKeys  map[string]synv1alpha1.DeployKey
	log         logr.Logger
	ops         manager.RepoOptions
}

// Create will create a new Gitlab project
func (g *Gitlab) Create() error {

	nsID, err := g.getNamespaceID()
	if err != nil {
		return err
	}

	projectOptions := &gitlab.CreateProjectOptions{
		Path:        &g.ops.RepoName,
		Name:        &g.ops.RepoName,
		Description: &g.ops.DisplayName,
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

// Update will update the Project Description and
// will overwrite the deployment keys on the endpoint that differ from the local ones. Currently it will not
// touch any additional keys that may have been added to the repository.
func (g *Gitlab) Update() (bool, error) {
	deployKeysUpdated, err := g.updateDeployKeys()
	if err != nil {
		return false, err
	}

	displayNameUpdated, err := g.updateDisplayName()
	if err != nil {
		return false, err
	}

	return deployKeysUpdated || displayNameUpdated, nil
}

func (g *Gitlab) updateDeployKeys() (bool, error) {
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

func (g *Gitlab) updateDisplayName() (bool, error) {
	err := g.Read()
	if err != nil {
		return false, err
	}

	remoteDisplayName := g.project.Description
	isUpdated := strings.Compare(remoteDisplayName, g.ops.DisplayName) != 0

	if isUpdated {
		project, _, err := g.client.Projects.EditProject(g.project.ID, &gitlab.EditProjectOptions{Description: &g.ops.DisplayName})
		if err != nil {
			return false, err
		}
		g.project = project
	}

	return isUpdated, nil
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
		return fmt.Errorf("no project %v found, can't delete", g.ops.Path)
	}

	_, err = g.client.Projects.DeleteProject(g.project.ID)

	return err
}

// getProject fetches all project information from the gitlab instance
func (g *Gitlab) getProject() error {
	project, _, err := g.client.Projects.GetProject(g.ops.Path+"/"+g.ops.RepoName, &gitlab.GetProjectOptions{})
	if err != nil {
		return err
	}

	g.project = project

	return nil
}

// Connect creates the Gitlab client
func (g *Gitlab) Connect() error {
	g.client = gitlab.NewClient(nil, g.credentials.Token)
	return g.client.SetBaseURL(g.ops.URL.Scheme + "://" + g.ops.URL.Host)
}

// FullURL returns the complete url of this git repository
func (g *Gitlab) FullURL() *url.URL {

	sshURL := g.ops.URL

	sshURL.Scheme = "ssh"
	sshURL.User = url.User("git")
	sshURL.Path = sshURL.Path + ".git"

	return sshURL
}

// IsType determines if the given url can be handled by this concrete implementation.
// This is done by a simple http query to the login page of gitlab. If any errors occur anywhere
// it will return false.
func (g *Gitlab) IsType(URL *url.URL) (bool, error) {

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	gitlabURL := URL.Scheme + "://" + URL.Host + "/users/sign_in'"

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

// New returns a new and empty Gitlab implementation
func (g *Gitlab) New(options manager.RepoOptions) (manager.Repo, error) {
	return &Gitlab{
		credentials: options.Credentials,
		deployKeys:  options.DeployKeys,
		log:         options.Logger,
		ops:         options,
	}, nil
}

func (g *Gitlab) getNamespaceID() (*int, error) {

	fetchedNamespace, _, err := g.client.Namespaces.GetNamespace(url.PathEscape(g.ops.Path))
	if err != nil {
		return nil, err
	}

	return &fetchedNamespace.ID, nil
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

// CommitTemplateFiles uploads all defined template files onto the repository.
func (g *Gitlab) CommitTemplateFiles() error {

	if len(g.ops.TemplateFiles) == 0 {
		return nil
	}

	filesToApply, err := g.compareFiles()
	if err != nil {
		return err
	}

	if len(filesToApply) == 0 {
		// we're done here
		return nil
	}

	g.log.Info("populating repository with template files")

	co := &gitlab.CreateCommitOptions{
		AuthorEmail:   builtinx.NewString("lieutenant-operator@syn.local"),
		AuthorName:    builtinx.NewString("Lieutenant Operator"),
		Branch:        builtinx.NewString("master"),
		CommitMessage: builtinx.NewString("Provision templates"),
	}

	co.Actions = []*gitlab.CommitAction{}

	for name, content := range filesToApply {

		co.Actions = append(co.Actions, &gitlab.CommitAction{
			Action:   gitlab.FileCreate,
			FilePath: name,
			Content:  content,
		})
	}

	_, _, err = g.client.Commits.CreateCommit(g.project.ID, co, nil)

	return err
}

// compareFiles will compare the files of the repositories root with the
// files that should be created. If there are existing files they will be
// dropped.
func (g *Gitlab) compareFiles() (map[string]string, error) {

	newmap := map[string]string{}

	trees, _, err := g.client.Repositories.ListTree(g.project.ID, nil, nil)
	if err != nil {
		// if the tree is not found it's probably just because there are no files at all currently...
		if strings.Contains(err.Error(), "Tree Not Found") {
			return g.ops.TemplateFiles, nil
		} else {
			return newmap, fmt.Errorf("cannot list files in repository: %s", err)
		}
	}

	treeMap := map[string]bool{}
	for _, tree := range trees {
		treeMap[tree.Path] = true
	}

	for k, v := range g.ops.TemplateFiles {
		if _, ok := treeMap[k]; !ok {
			newmap[k] = v
		}
	}

	return newmap, nil
}
