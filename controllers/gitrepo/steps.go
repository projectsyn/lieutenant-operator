package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/go-logr/logr"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	// Register Gitrepo implementation - DONOT REMOVE
	_ "github.com/projectsyn/lieutenant-operator/git"
	"github.com/projectsyn/lieutenant-operator/git/manager"
	"github.com/projectsyn/lieutenant-operator/pipeline"
)

const DEPLOY_KEY_NAME_INFIX = "-deploy-key-"
const DEPLOY_KEY_SECRET_PUBKEY = "publicKey"
const DEPLOY_KEY_SECRET_PRIVKEY = "privateKey"
const DEPLOY_KEY_SECRET_TYPE = "type"
const DEPLOY_KEY_GENERATED_PREFIX = "generated-"

func Steps(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	return steps(obj, data, manager.GetGitClient)
}

type gitClientFactory func(ctx context.Context, instance *synv1alpha1.GitRepo, reqLogger logr.Logger, client client.Client) (manager.Repo, string, error)

func steps(obj pipeline.Object, data *pipeline.Context, getGitClient gitClientFactory) pipeline.Result {
	instance, ok := obj.(*synv1alpha1.GitRepo)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object '%s/%s' is not of kind GitRepository", obj.GetNamespace(), obj.GetName())}
	}

	err := fetchGitRepoTemplate(instance, data)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("fetch Git repo template: %w", err)}
	}

	if instance.Spec.RepoType == synv1alpha1.UnmanagedRepoType {
		data.Log.Info("Skipping GitRepo '%s/%s' because it is unmanaged", obj.GetNamespace(), obj.GetName())
		return pipeline.Result{}
	}

	// NOTE(aa): Generate deploy keys before creating Git repo client, since the list of
	// deploy keys which the client is aware of is frozen at client-creation time.
	if err := ensureGeneratedDeployKeys(data.Context, data.Client, instance); err != nil {
		return pipeline.Result{Err: handleRepoError(data.Context, fmt.Errorf("ensure generated deploy keys: %w", err), instance, data.Client)}
	}

	repo, hostKeys, err := getGitClient(data.Context, instance, data.Log, data.Client)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("get Git client: %w", err)}
	}

	instance.Status.HostKeys = hostKeys

	exists, err := repoExists(repo)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to check if repo exists: %w", err)}
	}
	if !exists {
		data.Log.Info("creating git repo", manager.SecretEndpointName, repo.FullURL())
		instance.Status.URL = repo.FullURL().String()
		phase := synv1alpha1.Creating
		instance.Status.Phase = &phase
		if err := data.Client.Status().Update(data.Context, instance); err != nil {
			return pipeline.Result{Err: fmt.Errorf("could not set status while creating repository: %w", err)}
		}
		err := repo.Create()
		if err != nil {
			instance.Status.URL = "" // Revert status to reduce race condition likelihood
			return pipeline.Result{Err: handleRepoError(data.Context, err, instance, data.Client)}

		}
		data.Log.Info("successfully created the repository")
	}

	if instance.Status.URL != repo.FullURL().String() && instance.Spec.CreationPolicy != synv1alpha1.AdoptPolicy {
		var err error
		if !data.Deleted {
			phase := synv1alpha1.Failed
			instance.Status.Phase = &phase
			err = handleRepoError(data.Context, fmt.Errorf("Failed to adopt repository. Repository %q already exists and is not managed by %s ", repo.FullURL().String(), instance.Name), instance, data.Client)
		}
		return pipeline.Result{Err: err}
	}

	if data.Deleted {
		err := repo.Remove()
		if err != nil {
			return pipeline.Result{Err: fmt.Errorf("remove repo: %w", err)}
		}
		return pipeline.Result{}
	}

	if err := ensureAccessToken(data.Context, data.Client, instance, repo); err != nil {
		return pipeline.Result{Err: handleRepoError(data.Context, fmt.Errorf("ensure access token: %w", err), instance, data.Client)}
	}

	if err := ensureCIVariables(data.Context, data.Client, instance, repo); err != nil {
		return pipeline.Result{Err: handleRepoError(data.Context, fmt.Errorf("ensure ci variables: %w", err), instance, data.Client)}
	}

	err = repo.CommitTemplateFiles()
	if err != nil {
		return pipeline.Result{Err: handleRepoError(data.Context, err, instance, data.Client)}
	}

	changed, err := repo.Update()
	if err != nil {
		return pipeline.Result{Err: handleRepoError(data.Context, fmt.Errorf("update repo: %w", err), instance, data.Client)}
	}

	if changed {
		data.Log.Info("keys differed from CRD, keys re-applied to repository")
	}

	phase := synv1alpha1.Created
	instance.Status.Phase = &phase
	instance.Status.URL = repo.FullURL().String()
	instance.Status.Type = synv1alpha1.GitType(repo.Type())

	return pipeline.Result{}
}

func repoExists(repo manager.Repo) (bool, error) {
	err := repo.Read()
	if err != nil {
		if errors.Is(err, manager.ErrRepoNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil

}

func handleRepoError(ctx context.Context, err error, instance *synv1alpha1.GitRepo, client client.Client) error {
	phase := synv1alpha1.Failed
	instance.Status.Phase = &phase
	if updateErr := client.Status().Update(ctx, instance); updateErr != nil {
		return fmt.Errorf("could not set status while handling error: %s: %s", updateErr, err)
	}
	return err
}

const (
	LieutenantAccessTokenUIDAnnotation       = "lieutenant.syn.tools/accessTokenUID"
	LieutenantAccessTokenExpiresAtAnnotation = "lieutenant.syn.tools/accessTokenExpiresAt"
)

// ensureAccessToken ensures that an up-to-date access token returned from the manager is stored in the referenced secret.
// It passes the UID of the previous access token to the manager to ensure that the same access token is returned if it has not expired.
func ensureAccessToken(ctx context.Context, cli client.Client, instance *synv1alpha1.GitRepo, repo manager.Repo) error {
	name := instance.Spec.AccessToken.SecretRef
	if name == "" {
		return nil
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, cli, secret, func() error {
		uid := secret.Annotations[LieutenantAccessTokenUIDAnnotation]

		pat, err := repo.EnsureProjectAccessToken(ctx, instance.GetName(), manager.EnsureProjectAccessTokenOptions{
			UID: &uid,
		})
		if err != nil {
			return fmt.Errorf("error ensuring project access token: %w", err)
		}

		if pat.Updated() {
			if secret.Annotations == nil {
				secret.Annotations = make(map[string]string)
			}
			secret.Annotations[LieutenantAccessTokenUIDAnnotation] = pat.UID
			secret.Annotations[LieutenantAccessTokenExpiresAtAnnotation] = pat.ExpiresAt.Format(time.RFC3339)
			secret.Data = map[string][]byte{
				"token": []byte(pat.Token),
			}
		}

		return controllerutil.SetControllerReference(instance, secret, cli.Scheme())
	})
	if err != nil {
		return fmt.Errorf("error creating or updating access token secret: %w", err)
	}
	log.FromContext(ctx).Info("Reconciled secret",
		"secret", secret.Name,
		"pat_uid", secret.Annotations[LieutenantAccessTokenUIDAnnotation],
		"pat_expires_at", secret.Annotations[LieutenantAccessTokenExpiresAtAnnotation],
		"op", op)

	return nil
}

// ensureCIVariables ensures that the CI variables are set on the repository.
// It calls the manager with the current variables from the CRD and a combination of the previous variables and the current variables as the managed variables.
func ensureCIVariables(ctx context.Context, cli client.Client, instance *synv1alpha1.GitRepo, repo manager.Repo) error {
	var prevVars []synv1alpha1.EnvVar
	if instance.Status.LastAppliedCIVariables != "" {
		if err := json.Unmarshal([]byte(instance.Status.LastAppliedCIVariables), &prevVars); err != nil {
			return fmt.Errorf("error unmarshalling previous ci variables: %w", err)
		}
	}
	managedVars := sets.New[string]()
	for _, v := range instance.Spec.CIVariables {
		managedVars.Insert(v.Name)
	}
	for _, v := range prevVars {
		managedVars.Insert(v.Name)
	}

	vars := make([]manager.EnvVar, 0, len(instance.Spec.CIVariables))
	valueFromErrs := make([]error, 0, len(instance.Spec.CIVariables))
	for _, v := range instance.Spec.CIVariables {
		val, err := valueFromEnvVar(ctx, cli, instance.Namespace, v)
		if err != nil {
			valueFromErrs = append(valueFromErrs, err)
			continue
		}
		vars = append(vars, manager.EnvVar{
			Name:  v.Name,
			Value: val,

			GitlabOptions: manager.EnvVarGitlabOptions{
				Description: ptr.To(v.GitlabOptions.Description),
				Protected:   ptr.To(v.GitlabOptions.Protected),
				Masked:      ptr.To(v.GitlabOptions.Masked),
				Raw:         ptr.To(v.GitlabOptions.Raw),
			},
		})
	}
	if err := multierr.Combine(valueFromErrs...); err != nil {
		return fmt.Errorf("error collecting values for env vars: %w", err)
	}

	if err := repo.EnsureCIVariables(ctx, sets.List(managedVars), vars); err != nil {
		return fmt.Errorf("error ensuring ci variables: %w", err)
	}

	varsJSON, err := json.Marshal(instance.Spec.CIVariables)
	if err != nil {
		return fmt.Errorf("error marshalling ci variables: %w", err)
	}

	instance.Status.LastAppliedCIVariables = string(varsJSON)
	return nil
}

// ensureGeneratedDeployKeys ensures that the repo's `generateDeployKey` entries
// all have a corresponding `deployKey` entry, generating SSH keys as required
// and storing them in individual secrets.
func ensureGeneratedDeployKeys(ctx context.Context, cli client.Client, instance *synv1alpha1.GitRepo) error {
	for genKey, settings := range instance.Spec.GeneratedDeployKeys {
		secretName := instance.Name + DEPLOY_KEY_NAME_INFIX + genKey
		secretNSName := types.NamespacedName{Name: secretName, Namespace: instance.Namespace}
		secret := &corev1.Secret{}

		err := cli.Get(ctx, secretNSName, secret)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			err = generateNewDeployKeySecret(ctx, cli, settings, secretNSName, instance, secret)
			if err != nil {
				return err
			}
		}

		pubkeyB, ok := secret.Data[DEPLOY_KEY_SECRET_PUBKEY]
		if !ok {
			return fmt.Errorf("could not retrieve deploy key from secret: missing key: %s", DEPLOY_KEY_SECRET_PUBKEY)
		}
		pubkey := string(pubkeyB)
		parts := strings.Split(pubkey, " ")

		keyName := DEPLOY_KEY_GENERATED_PREFIX + genKey

		oldKey, ok := instance.Status.GeneratedDeployKeys[keyName]
		if !ok || oldKey.Key != parts[1] {
			if instance.Status.GeneratedDeployKeys == nil {
				instance.Status.GeneratedDeployKeys = make(map[string]synv1alpha1.DeployKeyStatus)
			}
			instance.Status.GeneratedDeployKeys[keyName] = synv1alpha1.DeployKeyStatus{
				DeployKey: synv1alpha1.DeployKey{
					Type:        parts[0],
					Key:         parts[1],
					WriteAccess: settings.WriteAccess,
				},
				SecretRef: secretName,
			}
		}
	}

	return nil
}

func generateNewDeployKeySecret(ctx context.Context, cli client.Client, settings synv1alpha1.DeployKeyTemplate, secretName types.NamespacedName, owner *synv1alpha1.GitRepo, secretRef *corev1.Secret) error {
	keyType := keygen.Ed25519
	if settings.Type == "ssh-rsa" {
		keyType = keygen.RSA
	}

	kp, err := keygen.New(
		"/tmp/"+secretName.Name,
		keygen.WithKeyType(keyType),
	)
	if err != nil {
		return err
	}

	secretRef.Name = secretName.Name
	secretRef.Namespace = secretName.Namespace

	secretRef.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(owner, owner.GroupVersionKind()), 
	}

	secretRef.Data = make(map[string][]byte)

	secretRef.Data[DEPLOY_KEY_SECRET_PUBKEY] = kp.RawAuthorizedKey()
	secretRef.Data[DEPLOY_KEY_SECRET_PRIVKEY] = kp.RawPrivateKey()

	return cli.Create(ctx, secretRef)
}

// valueFromEnvVar returns the value of an envVar. It returns an error if the envVar is invalid or the value cannot be retrieved.
// EnvVars with both value and valueFrom are invalid.
// An envVar with no value and no valueFrom returns an empty string.
// If valueFrom is set but the secret reference is not valid, an error is returned.
// If the secret does not exist and the secretKeyRef is optional, an empty string is returned. Otherwise, an error is returned.
func valueFromEnvVar(ctx context.Context, cli client.Client, namespace string, envVar synv1alpha1.EnvVar) (string, error) {
	l := log.FromContext(ctx).WithName("valueFromEnvVar")

	if envVar.Value != "" {
		if envVar.ValueFrom != nil {
			return "", fmt.Errorf("envVar %q has both value and valueFrom", envVar.Name)
		}
		return envVar.Value, nil
	}
	if envVar.ValueFrom == nil {
		return "", nil
	}
	if envVar.ValueFrom.SecretKeyRef == nil {
		return "", fmt.Errorf("envVar %q has no secretKeyRef", envVar.Name)
	}
	if envVar.ValueFrom.SecretKeyRef.Name == "" || envVar.ValueFrom.SecretKeyRef.Key == "" {
		return "", fmt.Errorf("envVar %q has incomplete secretKeyRef", envVar.Name)
	}
	optional := ptr.Deref(envVar.ValueFrom.SecretKeyRef.Optional, false)
	secret := &corev1.Secret{}
	err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: envVar.ValueFrom.SecretKeyRef.Name}, secret)
	if err != nil {
		if apierrors.IsNotFound(err) && optional {
			l.Info("secret not found but is optional, returning empty string", "secret", envVar.ValueFrom.SecretKeyRef.Name)
			return "", nil
		}
		return "", fmt.Errorf("error getting secret %q: %w", envVar.ValueFrom.SecretKeyRef.Name, err)
	}
	val, ok := secret.Data[envVar.ValueFrom.SecretKeyRef.Key]
	if !ok {
		if optional {
			l.Info("key not found but secret is optional, returning empty string", "key", envVar.ValueFrom.SecretKeyRef.Key, "secret", envVar.ValueFrom.SecretKeyRef.Name)
			return "", nil
		}
		return "", fmt.Errorf("secret %q does not contain key %q", envVar.ValueFrom.SecretKeyRef.Name, envVar.ValueFrom.SecretKeyRef.Key)
	}
	return string(val), nil
}
