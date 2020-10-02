package gitlab

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/projectsyn/lieutenant-operator/pkg/git/helpers"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"k8s.io/utils/pointer"

	"github.com/go-logr/logr"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"

	"github.com/xanzy/go-gitlab"
)

func init() {
	manager.Register(&Gitlab{})
}

// Gitlab holds the necessary information to communincate with a Gitlab server.
// Each Gitlab instance will handle exactly one project.
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

// Remove removes the project according to the recycle policy.
// Delete -> project gets deleted
// Archive -> project gets archived
// Retain -> nothing happens
func (g *Gitlab) Remove() error {
	switch g.ops.DeletionPolicy {
	case synv1alpha1.DeletePolicy:
		g.log.Info("deleting", "project", g.project.Name)
		return g.delete()
	case synv1alpha1.ArchivePolicy:
		g.log.Info("archiving", "project", g.project.Name)
		return g.archive()
	default:
		g.log.Info("retaining", "project", g.project.Name)
		return nil
	}
}

// archive archives the project handled by this gitlab instance
func (g *Gitlab) archive() error {
	err := g.getProject()
	if err != nil {
		return err
	}

	if g.project == nil {
		return fmt.Errorf("no project %v found, can't archive", g.ops.Path)
	}

	_, _, err = g.client.Projects.ArchiveProject(g.project.ID)

	return err
}

// delete deletes the project handled by the gitlab instance
func (g *Gitlab) delete() error {
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
	c, err := gitlab.NewClient(g.credentials.Token,
		gitlab.WithBaseURL(g.ops.URL.Scheme+"://"+g.ops.URL.Host))
	g.client = c
	return err
}

// FullURL returns the complete url of this git repository
func (g *Gitlab) FullURL() *url.URL {

	sshURL := g.ops.URL

	sshURL.Scheme = "ssh"
	sshURL.User = url.User("git")
	sshURL.Path = sshURL.Path + ".git"

	return sshURL
}

// TODO: this will be deprecated in favour of a fixed type definition in the
// CRD. As there's currently only the GitLab implementation this is a workaround
// for the brittle detection.
func (g *Gitlab) IsType(URL *url.URL) (bool, error) {

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

	filesToCommit, err := g.compareFiles()
	if err != nil {
		return err
	}

	if len(filesToCommit) == 0 {
		// we're done here
		return nil
	}

	g.log.Info("populating repository with template files")

	co := g.getCommitOptions()

	for _, file := range filesToCommit {
		action := &gitlab.CommitAction{
			FilePath: file.FileName,
			Content:  file.Content,
		}

		if file.Delete {
			g.log.Info("deleting file from repository", "file", action.FilePath, "repository", g.project.Name)
			action.Action = gitlab.FileDelete
		} else {
			g.log.Info("writing file to repository", "file", action.FilePath, "repository", g.project.Name)
			action.Action = gitlab.FileCreate
		}

		co.Actions = append(co.Actions, action)
	}

	_, _, err = g.client.Commits.CreateCommit(g.project.ID, co, nil)

	return err
}

// compareFiles will compare the files of the repositories root with the
// files that should be created. If there are existing files they will be
// dropped.
func (g *Gitlab) compareFiles() ([]manager.CommitFile, error) {

	files := []manager.CommitFile{}

	trees, _, err := g.client.Repositories.ListTree(g.project.ID, nil, nil)
	if err != nil {
		// if the tree is not found it's probably just because there are no files at all currently...
		// So we have to apply all pending ones.
		if strings.Contains(err.Error(), "Tree Not Found") {

			for name, content := range g.ops.TemplateFiles {
				files = append(files, manager.CommitFile{
					FileName: name,
					Content:  content,
				})
			}

			return files, nil
		}
		return files, fmt.Errorf("cannot list files in repository: %s", err)
	}

	compareMap := map[string]bool{}
	for _, tree := range trees {
		compareMap[tree.Path] = true
	}

	for name, content := range g.ops.TemplateFiles {
		if _, ok := compareMap[name]; ok && content == manager.DeletionMagicString {
			files = append(files, manager.CommitFile{
				FileName: name,
				Content:  content,
				Delete:   true,
			})
		} else if !ok && content != manager.DeletionMagicString {
			files = append(files, manager.CommitFile{
				FileName: name,
				Content:  content,
			})
		}

	}

	return files, nil
}

func (g *Gitlab) getCommitOptions() *gitlab.CreateCommitOptions {

	co := &gitlab.CreateCommitOptions{
		AuthorEmail:   pointer.StringPtr("lieutenant-operator@syn.local"),
		AuthorName:    pointer.StringPtr("Lieutenant Operator"),
		Branch:        pointer.StringPtr("master"),
		CommitMessage: pointer.StringPtr("Update cluster files"),
	}

	co.Actions = []*gitlab.CommitAction{}

	return co
}
