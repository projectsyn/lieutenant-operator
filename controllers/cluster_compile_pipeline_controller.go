package controllers

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (r *ClusterCompilePipelineReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling Cluster Compile Pipeline")
	fmt.Println("RECONCILE")
	fmt.Println(request.NamespacedName)

	instance := &synv1alpha1.Cluster{}

	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	nsName := types.NamespacedName{Name: instance.GetTenantRef().Name, Namespace: instance.GetNamespace()}
	tenant := &synv1alpha1.Tenant{}
	if err := r.Client.Get(ctx, nsName, tenant); err != nil {
		return reconcile.Result{}, fmt.Errorf("couldn't find tenant: %w", err)
	}

	pipelineStatus := tenant.Status.CompilePipeline
	if pipelineStatus == nil {
		pipelineStatus = &synv1alpha1.CompilePipelineStatus{}
	}
	clusterInList := slices.Contains(pipelineStatus.Clusters, instance.Name)

	if instance.Spec.EnableCompilePipeline && !clusterInList {
		pipelineStatus.Clusters = append(pipelineStatus.Clusters, instance.Name)
	}
	if !instance.Spec.EnableCompilePipeline && clusterInList {
		ind := slices.Index(pipelineStatus.Clusters, instance.Name)
		pipelineStatus.Clusters = slices.Delete(pipelineStatus.Clusters, ind, ind+1)
	}

	if instance.Spec.EnableCompilePipeline != clusterInList {
		fmt.Println("Updating!")
		tenant.Status.CompilePipeline = pipelineStatus
		err = r.Client.Status().Update(ctx, tenant)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterCompilePipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synv1alpha1.Cluster{}).
		Complete(r)
}
