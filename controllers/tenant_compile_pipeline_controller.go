package controllers

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
)

// TenantCompilePipelineReconciler reconciles a Tenant object, specifically the `Spec.CompilePipeline` field, updating the corresponding tenant's git repo accordingly.
type TenantCompilePipelineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	ApiUrl string
}

const (
	CI_VARIABLE_API_URL                     = "COMMODORE_API_URL"
	CI_VARIABLE_API_TOKEN                   = "COMMODORE_API_TOKEN"
	CI_VARIABLE_CLUSTERS                    = "CLUSTERS"
	CI_VARIABLE_PREFIX_CLUSTER_ACCESS_TOKEN = "ACCESS_TOKEN_"
	SECRET_KEY_API_TOKEN                    = "token"
	SECRET_KEY_GITLAB_TOKEN                 = "token"
)

//+kubebuilder:rbac:groups=syn.tools,resources=tenants,verbs=get;list;watch;update;patch;
//+kubebuilder:rbac:groups=syn.tools,resources=clusters,verbs=get;list;watch;
//+kubebuilder:rbac:groups=syn.tools,resources=tenants/status,verbs=get;

func (r *TenantCompilePipelineReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling Tenant Compile Pipeline")

	tenant := &synv1alpha1.Tenant{}

	err := r.Client.Get(ctx, request.NamespacedName, tenant)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	changed := false

	if !tenant.GetCompilePipelineSpec().Enabled {
		changed = ensureCiVariablesAbsent(tenant)
	} else {
		changed = r.ensureCiVariables(tenant) || changed
	}

	cluster := &synv1alpha1.Cluster{}
	pipelineStatus := tenant.GetCompilePipelineStatus()
	for _, clusterName := range pipelineStatus.Clusters {

		nsName := types.NamespacedName{Name: clusterName, Namespace: tenant.GetNamespace()}
		err := r.Client.Get(ctx, nsName, cluster)
		if err != nil {
			if errors.IsNotFound(err) {
				reqLogger.Info("Could not find cluster from list in .Status.CompilePipeline.Clusters", "clusterName", clusterName)
				continue
			}
			return reconcile.Result{}, fmt.Errorf("while reconciling CI variables for clusters: %w", err)
		}

		changed = ensureClusterCiVariable(tenant, cluster) || changed
	}
	if changed {
		err = r.Client.Update(ctx, tenant)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil

}

func ensureCiVariablesAbsent(t *synv1alpha1.Tenant) bool {
	vars, changed := removeEnvVar(CI_VARIABLE_API_URL, t.GetGitTemplate().CIVariables)
	vars, ch := removeEnvVar(CI_VARIABLE_API_TOKEN, vars)
	changed = ch || changed
	vars, ch = removeEnvVar(CI_VARIABLE_CLUSTERS, vars)
	changed = ch || changed
	if changed {
		t.GetGitTemplate().CIVariables = vars
	}
	return changed
}

func (r *TenantCompilePipelineReconciler) ensureCiVariables(t *synv1alpha1.Tenant) bool {
	template := t.GetGitTemplate()
	changed := false

	pipelineStatus := t.Status.CompilePipeline
	if pipelineStatus == nil {
		pipelineStatus = &synv1alpha1.CompilePipelineStatus{}
	}
	clusterList := strings.Join(pipelineStatus.Clusters, ",")

	list, ch := updateEnvVarValue(CI_VARIABLE_API_URL, r.ApiUrl, template.CIVariables)
	changed = ch
	list, ch = updateEnvVarValue(CI_VARIABLE_CLUSTERS, clusterList, list)
	changed = changed || ch
	list, ch = updateEnvVarValueFrom(CI_VARIABLE_API_TOKEN, t.Name, SECRET_KEY_API_TOKEN, false, list)
	changed = changed || ch

	if changed {
		template.CIVariables = list
	}

	return changed
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantCompilePipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synv1alpha1.Tenant{}).
		Complete(r)
}
