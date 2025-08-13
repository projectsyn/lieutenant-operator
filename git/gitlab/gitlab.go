package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/xanzy/go-gitlab"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/git/helpers"
	"github.com/projectsyn/lieutenant-operator/git/manager"
)

func init() {
	manager.Register(&Gitlab{})
}

var (
	ListItemsPerPage = 100
)

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
		if errors.Is(err, gitlab.ErrNotFound) {
			return manager.ErrRepoNotFound
		}
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

	sshURL := *g.ops.URL

	sshURL.Scheme = "ssh"
	sshURL.User = url.User("git")
	sshURL.Path = sshURL.Path + ".git"

	return &sshURL
}

// TODO: this will be deprecated in favour of a fixed type definition in the
// CRD. As there's currently only the GitLab implementation this is a workaround
// for the brittle detection.
func (g *Gitlab) IsType(_ *url.URL) (bool, error) {
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

	fetchedNamespace, _, err := g.client.Namespaces.GetNamespace(g.ops.Path)
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

func (g *Gitlab) overwriteKey(existingKeys []*gitlab.ProjectDeployKey, key string) {
	// unfortunately there's no way via the API to update a key, so we have to delete and recreate it, when it differs from
	// the yaml in the k8s cluster.
	for _, deleteKey := range existingKeys {
		if deleteKey.Title == key {
			g.log.Info("forcing re-creation of key " + key)
			g.deleteKey(deleteKey)
		}
	}
}

func (g *Gitlab) deleteKey(deleteKey *gitlab.ProjectDeployKey) {
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
			WriteAccess: key.CanPush,
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
		file := file // Make a copy so we can savely reference the file
		var fileAction gitlab.FileActionValue
		if file.Delete {
			g.log.Info("deleting file from repository", "file", file.FileName, "repository", g.project.Name)
			fileAction = gitlab.FileDelete
		} else {
			g.log.Info("writing file to repository", "file", file.FileName, "repository", g.project.Name)
			fileAction = gitlab.FileCreate
		}

		commitAction := &gitlab.CommitActionOptions{
			FilePath: &file.FileName,
			Content:  &file.Content,
			Action:   &fileAction,
		}

		co.Actions = append(co.Actions, commitAction)
	}

	_, _, err = g.client.Commits.CreateCommit(g.project.ID, co, nil)

	return err
}

// compareFiles will compare the files of the repositories root with the
// files that should be created. If there are existing files they will be
// dropped.
func (g *Gitlab) compareFiles() ([]manager.CommitFile, error) {

	files := make([]manager.CommitFile, 0)
	resp := &gitlab.Response{NextPage: 1}
	var trees []*gitlab.TreeNode
	var err error
	compareMap := map[string]bool{}

	// The NextPage header is empty/zero in the last page.
	for resp.NextPage > 0 {
		trees, resp, err = g.client.Repositories.ListTree(g.project.ID, &gitlab.ListTreeOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: ListItemsPerPage,
				Page:    resp.NextPage,
			},
		}, nil)
		if err != nil {
			// if the tree is not found it's probably just because there are no files at all currently...
			// So we have to apply all pending ones.
			if errors.Is(err, gitlab.ErrNotFound) {
				g.log.Info("ListTree got 404; most likely no files found in repository, applying all pending files")

				for name, content := range g.ops.TemplateFiles {
					if content != manager.DeletionMagicString {
						files = append(files, manager.CommitFile{
							FileName: name,
							Content:  content,
						})
					}
				}

				return files, nil
			}
			return files, fmt.Errorf("cannot list files in repository: %s", err)
		}
		for _, tree := range trees {
			compareMap[tree.Path] = true
		}
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
		AuthorEmail:   ptr.To("lieutenant-operator@syn.local"),
		AuthorName:    ptr.To("Lieutenant Operator"),
		Branch:        ptr.To("master"),
		CommitMessage: ptr.To("Update cluster files"),
	}

	co.Actions = make([]*gitlab.CommitActionOptions, 0)

	return co
}

// EnsureProjectAccessToken ensures that the project has an access token set.
// If the token is expired or not set, a new token will be created.
// This implementation does not use the Gitlab token rotation feature.
// Using this feature would invalidate old tokens immediately, which would break pipelines.
// Also the newly created token would immediately be revoked if an old one is used.
func (g *Gitlab) EnsureProjectAccessToken(ctx context.Context, name string, opts manager.EnsureProjectAccessTokenOptions) (manager.ProjectAccessToken, error) {
	at, _, err := g.client.ProjectAccessTokens.ListProjectAccessTokens(g.project.ID, &gitlab.ListProjectAccessTokensOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		return manager.ProjectAccessToken{}, err
	}
	validATs := make([]gitlab.ProjectAccessToken, 0, len(at))
	for _, token := range at {
		if token == nil {
			continue
		}
		if !token.Active {
			continue
		}
		if token.Revoked {
			continue
		}
		if token.ExpiresAt == nil {
			continue
		}
		if time.Time(*token.ExpiresAt).Before(g.ops.Now().Add(-10 * 24 * time.Hour)) {
			continue
		}
		validATs = append(validATs, *token)
	}

	slices.SortFunc(validATs, func(a, b gitlab.ProjectAccessToken) int {
		at := time.Time(ptr.Deref(a.ExpiresAt, gitlab.ISOTime{}))
		bt := time.Time(ptr.Deref(b.ExpiresAt, gitlab.ISOTime{}))
		if at.Before(bt) {
			return 1
		}
		if at.After(bt) {
			return -1
		}
		return 0
	})

	if opts.UID == nil {
		if len(validATs) > 0 {
			return manager.ProjectAccessToken{
				UID:       strconv.Itoa(validATs[0].ID),
				ExpiresAt: time.Time(*validATs[0].ExpiresAt),
			}, nil
		}
	} else {
		uid := *opts.UID
		for _, token := range validATs {
			if strconv.Itoa(token.ID) == uid {
				return manager.ProjectAccessToken{
					UID:       uid,
					ExpiresAt: time.Time(*token.ExpiresAt),
				}, nil
			}
		}
		if len(validATs) > 0 {
			g.log.Info("found valid access token, but no UID match", "uid", validATs[0].ID, "given_uid", opts.UID)
		}
	}

	token, _, err := g.client.ProjectAccessTokens.CreateProjectAccessToken(g.project.ID, &gitlab.CreateProjectAccessTokenOptions{
		// Gitlab allows duplicated names and we can easily identify tokens by age.
		// So we just reuse the name.
		Name:        &name,
		ExpiresAt:   ptr.To(gitlab.ISOTime(g.ops.Now().Add(30 * 24 * time.Hour))),
		Scopes:      ptr.To([]string{"write_repository"}),
		AccessLevel: ptr.To(gitlab.MaintainerPermissions),
	}, gitlab.WithContext(ctx))

	if err != nil {
		return manager.ProjectAccessToken{}, fmt.Errorf("error response from gitlab when creating ProjectAccessToken: %w", err)
	}

	return manager.ProjectAccessToken{
		UID:       strconv.Itoa(token.ID),
		Token:     token.Token,
		ExpiresAt: time.Time(ptr.Deref(token.ExpiresAt, gitlab.ISOTime{})),
	}, nil
}

// EnsureCIVariables ensures that the given variables are set in the CI/CD pipeline.
// The managedVariables is used to identify the variables that are managed by the operator.
// Variables that are not managed by the operator will be ignored.
// Variables that are managed but not in variables will be deleted.
func (g *Gitlab) EnsureCIVariables(ctx context.Context, managedVariables []string, variables []manager.EnvVar) error {
	l := log.FromContext(ctx).WithName("EnsureCIVariables")

	var errs []error
	managed := sets.New(managedVariables...)
	current := sets.New[string]()
	for _, v := range variables {
		current.Insert(v.Name)
	}

	toDelete := managed.Difference(current)
	for _, v := range sets.List(toDelete) {
		_, err := g.client.ProjectVariables.RemoveVariable(g.project.ID, v, &gitlab.RemoveProjectVariableOptions{}, gitlab.WithContext(ctx))
		if err != nil && !errors.Is(err, gitlab.ErrNotFound) {
			errs = append(errs, fmt.Errorf("error removing variable %s: %w", v, err))
		}
	}

	remote, _, err := g.client.ProjectVariables.ListVariables(g.project.ID, &gitlab.ListProjectVariablesOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error listing variables: %w", err)
	}
	remoteByName := make(map[string]gitlab.ProjectVariable, len(remote))
	for _, v := range remote {
		if v == nil {
			continue
		}
		remoteByName[v.Key] = *v
	}

	for _, v := range variables {
		if !managed.Has(v.Name) {
			continue
		}

		remote, ok := remoteByName[v.Name]
		var changed bool
		if ok {
			changed = varNeedsUpdate(remote, v)
		} else {
			changed = true
		}
		if !changed {
			continue
		}

		l.Info("updating changed variable", "name", v.Name)
		err := g.updateOrCreateCIVariable(ctx, v)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return multierr.Combine(errs...)
}

// varNeedsUpdate returns true if the remote variable needs to be updated.
// Does not check the Key, as the key is not allowed to change.
func varNeedsUpdate(remote gitlab.ProjectVariable, local manager.EnvVar) bool {
	if remote.Value != local.Value {
		return true
	}
	if local.GitlabOptions.Description != nil && remote.Description != *local.GitlabOptions.Description {
		return true
	}
	if local.GitlabOptions.Protected != nil && remote.Protected != *local.GitlabOptions.Protected {
		return true
	}
	if local.GitlabOptions.Masked != nil && remote.Masked != *local.GitlabOptions.Masked {
		return true
	}
	if local.GitlabOptions.Raw != nil && remote.Raw != *local.GitlabOptions.Raw {
		return true
	}
	return false
}

// updateOrCreateCIVariable updates or creates a CI variable in the Gitlab project.
// It tries to update the variable first and creates it if it does not exist.
func (g *Gitlab) updateOrCreateCIVariable(ctx context.Context, v manager.EnvVar) error {
	_, _, err := g.client.ProjectVariables.UpdateVariable(g.project.ID, v.Name, &gitlab.UpdateProjectVariableOptions{
		Value:     &v.Value,
		Protected: v.GitlabOptions.Protected,
		Masked:    v.GitlabOptions.Masked,
		Raw:       v.GitlabOptions.Raw,
	}, gitlab.WithContext(ctx))
	if err == nil {
		return nil
	}
	if !errors.Is(err, gitlab.ErrNotFound) {
		return fmt.Errorf("error updating variable %s: %w", v.Name, err)
	}
	_, _, err = g.client.ProjectVariables.CreateVariable(g.project.ID, &gitlab.CreateProjectVariableOptions{
		Key:       &v.Name,
		Value:     &v.Value,
		Protected: v.GitlabOptions.Protected,
		Masked:    v.GitlabOptions.Masked,
		Raw:       v.GitlabOptions.Raw,
	}, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error creating variable %s: %w", v.Name, err)
	}
	return nil
}
