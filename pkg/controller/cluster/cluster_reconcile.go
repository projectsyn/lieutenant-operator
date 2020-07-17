package cluster

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	synTenant "github.com/projectsyn/lieutenant-operator/pkg/controller/tenant"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"
	"github.com/projectsyn/lieutenant-operator/pkg/vault"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusterClassContent = `classes:
- %s.%s
`
	finalizerName = "cluster.lieutenant.syn.tools"
)

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Cluster")

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		instance := &synv1alpha1.Cluster{}

		err := r.client.Get(context.TODO(), request.NamespacedName, instance)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		if err := r.createClusterRBAC(*instance); err != nil {
			return err
		}

		if instance.Status.BootstrapToken == nil {
			reqLogger.Info("Adding status to Cluster object")
			err := r.newStatus(instance)
			if err != nil {
				return err
			}
		}

		if time.Now().After(instance.Status.BootstrapToken.ValidUntil.Time) {
			instance.Status.BootstrapToken.TokenValid = false
		}

		gvk := schema.GroupVersionKind{
			Version: instance.APIVersion,
			Kind:    instance.Kind,
		}

		if len(instance.Spec.GitRepoTemplate.DisplayName) == 0 {
			instance.Spec.GitRepoTemplate.DisplayName = instance.Spec.DisplayName
		}

		instance.Spec.GitRepoTemplate.DeletionPolicy = instance.Spec.DeletionPolicy

		err = helpers.CreateOrUpdateGitRepo(instance, gvk, instance.Spec.GitRepoTemplate, r.client, instance.Spec.TenantRef)
		if err != nil {
			reqLogger.Error(err, "Cannot create or update git repo object")
			return err
		}

		repoName := request.NamespacedName
		repoName.Name = instance.Spec.TenantRef.Name

		var vaultClient vault.VaultClient = nil
		secretPath := path.Join(instance.Spec.TenantRef.Name, instance.Name, "steward")

		deletionPolicy := instance.Spec.DeletionPolicy
		if deletionPolicy == "" {
			deletionPolicy = helpers.GetDeletionPolicy()
		}

		vaultClient, err = vault.NewClient(deletionPolicy, reqLogger)
		if err != nil {
			return err
		}

		if strings.ToLower(os.Getenv("SKIP_VAULT_SETUP")) != "true" {

			token, err := r.getServiceAccountToken(instance)
			if err != nil {
				return err
			}

			err = vaultClient.AddSecrets([]vault.VaultSecret{{Path: secretPath, Value: token}})
			if err != nil {
				return err
			}

		}

		deleted := helpers.HandleDeletion(instance, finalizerName, r.client)
		if deleted.FinalizerRemoved {
			if vaultClient != nil {
				err := vaultClient.RemoveSecrets([]vault.VaultSecret{{Path: path.Dir(secretPath), Value: ""}})
				if err != nil {
					return err
				}
			}
			err = r.removeClusterFileFromTenant(instance.GetName(), repoName, reqLogger)
			if err != nil {
				return err
			}
		}
		if deleted.Deleted {
			return r.client.Update(context.TODO(), instance)
		}

		err = r.updateTenantGitRepo(repoName, instance.GetName())
		if err != nil {
			return err
		}

		helpers.AddTenantLabel(&instance.ObjectMeta, instance.Spec.TenantRef.Name)
		helpers.AddDeletionProtection(instance)
		controllerutil.AddFinalizer(instance, finalizerName)

		instance.Spec.GitRepoURL, instance.Spec.GitHostKeys, err = helpers.GetGitRepoURLAndHostKeys(instance, r.client)
		if err != nil {
			return err
		}
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			return err
		}
		return r.client.Update(context.TODO(), instance)
	})

	return reconcile.Result{}, err
}

func (r *ReconcileCluster) generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), err
}

//newStatus will create a default lifetime of 30 minutes if it wasn't set in the object.
func (r *ReconcileCluster) newStatus(cluster *synv1alpha1.Cluster) error {

	parseTime := "30m"
	if cluster.Spec.TokenLifeTime != "" {
		parseTime = cluster.Spec.TokenLifeTime
	}

	duration, err := time.ParseDuration(parseTime)
	if err != nil {
		return err
	}

	validUntil := time.Now().Add(duration)

	token, err := r.generateToken()
	if err != nil {
		return err
	}

	cluster.Status.BootstrapToken = &synv1alpha1.BootstrapToken{
		Token:      token,
		ValidUntil: metav1.NewTime(validUntil),
		TokenValid: true,
	}
	return nil
}

func (r *ReconcileCluster) getTenantCR(tenant types.NamespacedName) (*synv1alpha1.Tenant, error) {
	tenantCR := &synv1alpha1.Tenant{}
	return tenantCR, r.client.Get(context.TODO(), tenant, tenantCR)
}

func (r *ReconcileCluster) updateTenantGitRepo(tenant types.NamespacedName, clusterName string) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {

		tenantCR, err := r.getTenantCR(tenant)
		if err != nil {
			return err
		}

		if tenantCR.Spec.GitRepoTemplate.TemplateFiles == nil {
			tenantCR.Spec.GitRepoTemplate.TemplateFiles = map[string]string{}
		}

		clusterClassFile := clusterName + ".yml"
		if _, ok := tenantCR.Spec.GitRepoTemplate.TemplateFiles[clusterClassFile]; !ok {
			fileContent := fmt.Sprintf(clusterClassContent, tenant.Name, synTenant.CommonClassName)
			tenantCR.Spec.GitRepoTemplate.TemplateFiles[clusterClassFile] = fileContent
			return r.client.Update(context.TODO(), tenantCR)
		}
		return nil
	})
}

func (r *ReconcileCluster) getServiceAccountToken(instance metav1.Object) (string, error) {
	secrets := &corev1.SecretList{}

	err := r.client.List(context.TODO(), secrets)
	if err != nil {
		return "", err
	}

	sortSecrets := helpers.SecretSortList(*secrets)

	sort.Sort(sort.Reverse(sortSecrets))

	for _, secret := range sortSecrets.Items {

		if secret.Type != corev1.SecretTypeServiceAccountToken {
			continue
		}

		if secret.Annotations[corev1.ServiceAccountNameKey] == instance.GetName() {
			if string(secret.Data["token"]) == "" {
				// We'll skip the secrets if the token is not yet populated.
				continue
			}
			return string(secret.Data["token"]), nil
		}
	}

	return "", fmt.Errorf("no matching secrets found")
}

func (r *ReconcileCluster) removeClusterFileFromTenant(clusterName string, tenantInfo types.NamespacedName, reqLogger logr.Logger) error {

	tenantCR, err := r.getTenantCR(tenantInfo)
	if err != nil {
		return err
	}

	fileName := clusterName + ".yml"

	if tenantCR.Spec.GitRepoTemplate.TemplateFiles == nil {
		return nil
	}

	if _, ok := tenantCR.Spec.GitRepoTemplate.TemplateFiles[fileName]; ok {
		tenantCR.Spec.GitRepoTemplate.TemplateFiles[fileName] = manager.DeletionMagicString
		err := r.client.Update(context.TODO(), tenantCR)
		if err != nil {
			return err
		}
	}

	return nil
}
