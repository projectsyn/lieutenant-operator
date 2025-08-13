package controllers

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	var clusters synv1alpha1.ClusterList
	if err := r.Client.List(ctx, &clusters,
		client.InNamespace(tenant.GetNamespace()),
		client.MatchingLabels{synv1alpha1.LabelNameTenant: tenant.Name},
	); err != nil {
		return reconcile.Result{}, fmt.Errorf("error listing clusters: %w", err)
	}
	clustersWithPipelineEnabled := make([]synv1alpha1.Cluster, 0, len(clusters.Items))
	for _, cluster := range clusters.Items {
		if cluster.GetEnableCompilePipeline() {
			clustersWithPipelineEnabled = append(clustersWithPipelineEnabled, cluster)
		}
	}

	if !tenant.GetCompilePipelineSpec().Enabled {
		changed = ensureCiVariablesAbsent(tenant)
	} else {
		changed = r.ensureCiVariables(tenant, filterDeletedClusters(clustersWithPipelineEnabled)) || changed
	}

	for _, cluster := range clusters.Items {
		if cluster.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(&cluster, synv1alpha1.PipelineFinalizerName) {
			if err := r.Client.Update(ctx, &cluster); err != nil {
				return reconcile.Result{}, fmt.Errorf("error adding finalizer to cluster %s: %w", cluster.Name, err)
			}
		}

		if cluster.GetGitTemplate().RepoType == synv1alpha1.UnmanagedRepoType {
			reqLogger.Info("Skipping cluster with unmanaged repo", "cluster", cluster.Name)
			continue
		}

		changed = ensureClusterCiVariable(tenant, cluster) || changed
	}

	if changed {
		if err := r.Client.Update(ctx, tenant); err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating tenant: %w", err)
		}
	}

	cs := clustersWithPipelineEnabled
	if !tenant.GetCompilePipelineSpec().Enabled {
		cs = []synv1alpha1.Cluster{}
	}
	if ensureStatus(tenant, cs) {
		if err := r.Client.Status().Update(ctx, tenant); err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating tenant status: %w", err)
		}
	}

	// Explicitly remove finalizers from ALL clusters that are being deleted to not block their deletion if the pipeline is disabled.
	// At this point the tenant has been updated and we can safely remove the finalizers.
	for _, cluster := range clusters.Items {
		if cluster.GetDeletionTimestamp().IsZero() {
			continue
		}
		if controllerutil.RemoveFinalizer(&cluster, synv1alpha1.PipelineFinalizerName) {
			if err := r.Client.Update(ctx, &cluster); err != nil {
				return reconcile.Result{}, fmt.Errorf("error removing finalizer from cluster %s: %w", cluster.Name, err)
			}
		}
	}

	return reconcile.Result{}, nil
}

func ensureStatus(tenant *synv1alpha1.Tenant, clustersWithPipelineEnabled []synv1alpha1.Cluster) bool {
	sc := sortedClusterNames(filterDeletedClusters(clustersWithPipelineEnabled))
	if slices.Equal(tenant.GetCompilePipelineStatus().Clusters, sc) {
		return false
	}

	if tenant.Status.CompilePipeline == nil {
		tenant.Status.CompilePipeline = &synv1alpha1.CompilePipelineStatus{}
	}
	tenant.Status.CompilePipeline.Clusters = sc
	return true
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

func (r *TenantCompilePipelineReconciler) ensureCiVariables(t *synv1alpha1.Tenant, clusters []synv1alpha1.Cluster) bool {
	template := t.GetGitTemplate()
	changed := false

	clusterList := strings.Join(sortedClusterNames(clusters), " ")

	list, ch := updateEnvVarValue(CI_VARIABLE_API_URL, r.ApiUrl, template.CIVariables)
	changed = ch
	list, ch = updateEnvVarValue(CI_VARIABLE_CLUSTERS, clusterList, list)
	changed = changed || ch
	list, ch = updateEnvVarValueFrom(CI_VARIABLE_API_TOKEN, t.Name, SECRET_KEY_API_TOKEN, list)
	changed = changed || ch

	if changed {
		template.CIVariables = list
	}

	return changed
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantCompilePipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("tenant_compile_pipeline").
		For(&synv1alpha1.Tenant{}).
		Owns(&synv1alpha1.Cluster{}).
		Complete(r)
}

func filterDeletedClusters(clusters []synv1alpha1.Cluster) []synv1alpha1.Cluster {
	filtered := make([]synv1alpha1.Cluster, 0, len(clusters))
	for _, c := range clusters {
		if c.GetDeletionTimestamp().IsZero() {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func sortedClusterNames(clusters []synv1alpha1.Cluster) []string {
	names := make([]string, len(clusters))
	for i, cluster := range clusters {
		names[i] = cluster.Name
	}
	sort.Strings(names)
	return names
}

func ensureClusterCiVariable(t *synv1alpha1.Tenant, c synv1alpha1.Cluster) bool {
	remove :=
		!c.GetDeletionTimestamp().IsZero() ||
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
		list, changed = updateEnvVarValueFrom(envVarName, c.Spec.GitRepoTemplate.AccessToken.SecretRef, SECRET_KEY_GITLAB_TOKEN, template.CIVariables)
	}

	if changed {
		template.CIVariables = list
		t.Spec.GitRepoTemplate = template
	}

	return changed
}
