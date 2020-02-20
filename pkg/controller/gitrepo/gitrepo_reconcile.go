package gitrepo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// SecretTokenName is the name of the secret entry containing the token
	SecretTokenName = "token"
	// SecretHostKeysName is the name of the secret entry containing the SSH host keys
	SecretHostKeysName = "hostKeys"
	// SecretEndpointName is the name of the secret entry containing the api endpoint
	SecretEndpointName = "endpoint"
)

// Reconcile will create or delete a git repository based on the event.
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGitRepo) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GitRepo")

	// Fetch the GitRepo instance
	instance := &synv1alpha1.GitRepo{}

	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	helpers.AddTenantLabel(&instance.ObjectMeta, instance.Spec.TenantRef.Name)
	if instance.Spec.RepoType == synv1alpha1.DefaultRepoType {
		instance.Spec.RepoType = synv1alpha1.AutoRepoType
	}
	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{
		Name:      instance.Spec.APISecretRef.Name,
		Namespace: instance.Namespace,
	}

	if len(instance.Spec.APISecretRef.Namespace) > 0 {
		namespacedName.Namespace = instance.Spec.APISecretRef.Namespace
	}

	err = r.client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error getting git secret: %v", err)
	}

	if hostKeys, ok := secret.Data[SecretHostKeysName]; ok {
		instance.Status.HostKeys = string(hostKeys)
	}

	if _, ok := secret.Data[SecretEndpointName]; !ok {
		return reconcile.Result{}, fmt.Errorf("secret %s does not contain endpoint data", secret.GetName())
	}

	if _, ok := secret.Data[SecretTokenName]; !ok {
		return reconcile.Result{}, fmt.Errorf("secret %s does not contain token", secret.GetName())
	}

	repoURL, err := url.Parse(string(secret.Data[SecretEndpointName]) + "/" + instance.Spec.Path + "/" + instance.Spec.RepoName)

	if err != nil {
		return reconcile.Result{}, err
	}

	repoOptions := manager.RepoOptions{
		Credentials: manager.Credentials{
			Token: string(secret.Data[SecretTokenName]),
		},
		DeployKeys: instance.Spec.DeployKeys,
		Logger:     reqLogger,
		Path:       instance.Spec.Path,
		RepoName:   instance.Spec.RepoName,
		URL:        repoURL,
	}

	repo, err := manager.NewRepo(repoOptions)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = repo.Connect()
	if err != nil {
		return reconcile.Result{}, err
	}

	if !r.repoExists(repo) {
		reqLogger.Info("creating git repo", SecretEndpointName, repoOptions.URL)
		err = repo.Create()
		if err != nil {
			phase := synv1alpha1.Failed
			instance.Status.Phase = &phase
			instance.Status.URL = repo.FullURL().String()
			if updateErr := r.client.Status().Update(context.TODO(), instance); updateErr != nil {
				return reconcile.Result{}, fmt.Errorf("could not set status while handling error: %s: %s", updateErr, err)
			}
			return reconcile.Result{}, err
		}

		reqLogger.Info("successfully created the repository")
		phase := synv1alpha1.Created
		instance.Status.Phase = &phase
		instance.Status.URL = repo.FullURL().String()
		return reconcile.Result{}, r.client.Status().Update(context.TODO(), instance)
	}
	changed, err := repo.Update()
	if err != nil {
		return reconcile.Result{}, err
	}

	if changed {
		reqLogger.Info("keys differed from CRD, keys re-applied to repository")
	}

	helpers.AddTenantLabel(&instance.ObjectMeta, instance.Spec.TenantRef.Name)

	return reconcile.Result{}, r.client.Status().Update(context.TODO(), instance)
}

func (r *ReconcileGitRepo) repoExists(repo manager.Repo) bool {
	if err := repo.Read(); err == nil {
		return true
	}

	return false
}
