package controllers

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
)

// ClusterCompilePipelineReconciler reconciles a Cluster object, specifically the `Spec.EnableCompilePipeline` field, updating the corresponding tenant's status accordingly.
type ClusterCompilePipelineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=syn.tools,resources=clusters,verbs=get;list;watch;
//+kubebuilder:rbac:groups=syn.tools,resources=tenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=syn.tools,resources=clusters/finalizers,verbs=update

func (r *ClusterCompilePipelineReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling Cluster Compile Pipeline")

	instance := &synv1alpha1.Cluster{}

	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(instance, synv1alpha1.PipelineFinalizerName) {
		if instance.GetDeletionTimestamp().IsZero() {
			controllerutil.AddFinalizer(instance, synv1alpha1.PipelineFinalizerName)
			return reconcile.Result{}, r.Client.Update(ctx, instance)
		} else {
			return reconcile.Result{}, nil
		}
	}

	nsName := types.NamespacedName{Name: instance.GetTenantRef().Name, Namespace: instance.GetNamespace()}
	tenant := &synv1alpha1.Tenant{}
	if err := r.Client.Get(ctx, nsName, tenant); err != nil {
		return reconcile.Result{}, fmt.Errorf("couldn't find tenant: %w", err)
	}

	if ensureTenantStatus(tenant, instance) {
		err = r.Client.Status().Update(ctx, tenant)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	if ensureClusterCiVariable(tenant, instance) {
		err = r.Client.Update(ctx, tenant)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// We can only get here if the cluster list and CI variables were successfully updated on the tenant.
	// So in the case of deletion, we can clean up the finalizer here, because that update involves removing them.
	if !instance.GetDeletionTimestamp().IsZero() {
		if controllerutil.RemoveFinalizer(instance, synv1alpha1.PipelineFinalizerName) {
			return ctrl.Result{}, r.Client.Update(ctx, instance)
		}
	}

	return reconcile.Result{}, nil
}

func ensureTenantStatus(t *synv1alpha1.Tenant, c *synv1alpha1.Cluster) bool {
	deleted := !c.GetDeletionTimestamp().IsZero()
	pipelineStatus := t.GetCompilePipelineStatus()
	clusterInList := slices.Contains(pipelineStatus.Clusters, c.Name)
	if deleted && clusterInList {
		ind := slices.Index(pipelineStatus.Clusters, c.Name)
		pipelineStatus.Clusters = slices.Delete(pipelineStatus.Clusters, ind, ind+1)
		t.Status.CompilePipeline = pipelineStatus
		return true
	}
	if c.GetEnableCompilePipeline() && !clusterInList {
		pipelineStatus.Clusters = append(pipelineStatus.Clusters, c.Name)
		slices.Sort(pipelineStatus.Clusters)
		t.Status.CompilePipeline = pipelineStatus
		return true
	}
	if !c.GetEnableCompilePipeline() && clusterInList {
		ind := slices.Index(pipelineStatus.Clusters, c.Name)
		pipelineStatus.Clusters = slices.Delete(pipelineStatus.Clusters, ind, ind+1)
		t.Status.CompilePipeline = pipelineStatus
		return true
	}
	return false
}

func ensureClusterCiVariable(t *synv1alpha1.Tenant, c *synv1alpha1.Cluster) bool {
	remove := !c.GetDeletionTimestamp().IsZero() ||
		c.GetGitTemplate().AccessToken.SecretRef == "" ||
		!t.GetCompilePipelineSpec().Enabled ||
		!c.GetEnableCompilePipeline()

	template := t.GetGitTemplate()
	envVarName := fmt.Sprintf("%s%s", CI_VARIABLE_PREFIX_CLUSTER_ACCESS_TOKEN, strings.Replace(c.GetName(), "-", "_", -1))

	var list []synv1alpha1.EnvVar
	var changed bool

	if remove {
		list, changed = removeEnvVar(envVarName, template.CIVariables)
	} else {
		list, changed = updateEnvVarValueFrom(envVarName, c.Spec.GitRepoTemplate.AccessToken.SecretRef, SECRET_KEY_GITLAB_TOKEN, true, template.CIVariables)
	}

	if changed {
		template.CIVariables = list
		t.Spec.GitRepoTemplate = template
	}

	return changed

}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterCompilePipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synv1alpha1.Cluster{}).
		Complete(r)
}
