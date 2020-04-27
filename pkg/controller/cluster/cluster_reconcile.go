package cluster

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Cluster")

	instance := &synv1alpha1.Cluster{}

	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := r.createClusterRBAC(*instance); err != nil {
		return reconcile.Result{}, err
	}

	if instance.Status.BootstrapToken == nil {
		reqLogger.Info("Adding status to Cluster object")
		err := r.newStatus(instance)
		if err != nil {
			return reconcile.Result{}, err
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

	err = helpers.CreateOrUpdateGitRepo(instance, gvk, instance.Spec.GitRepoTemplate, r.client, instance.Spec.TenantRef)
	if err != nil {
		reqLogger.Error(err, "Cannot create or update git repo object")
		return reconcile.Result{}, err
	}

	repoName := request.NamespacedName
	repoName.Name = instance.Spec.TenantRef.Name

	err = r.updateTenantGitRepo(repoName, instance.GetName()+".yml")
	if err != nil {
		return reconcile.Result{}, err
	}

	helpers.AddTenantLabel(&instance.ObjectMeta, instance.Spec.TenantRef.Name)
	instance.Spec.GitRepoURL, instance.Spec.GitHostKeys, err = helpers.GetGitRepoURLAndHostKeys(instance, r.client)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, r.client.Update(context.TODO(), instance)
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

func (r *ReconcileCluster) updateTenantGitRepo(tenant types.NamespacedName, filename string) error {
	tenantCR := &synv1alpha1.Tenant{}

	err := r.client.Get(context.TODO(), tenant, tenantCR)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if tenantCR.Spec.GitRepoTemplate.TemplateFiles == nil {
		tenantCR.Spec.GitRepoTemplate.TemplateFiles = map[string]string{}
	}

	if _, ok := tenantCR.Spec.GitRepoTemplate.TemplateFiles[filename]; !ok {

		tenantCR.Spec.GitRepoTemplate.TemplateFiles[filename] = ""

		return r.client.Update(context.TODO(), tenantCR)
	}

	return nil
}
