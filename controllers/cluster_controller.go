package controllers

import (
	"context"

	"github.com/projectsyn/lieutenant-operator/controllers/cluster"
	"github.com/projectsyn/lieutenant-operator/controllers/gitrepo"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	CreateSATokenSecret   bool
	DefaultCreationPolicy synv1alpha1.CreationPolicy
}

//+kubebuilder:rbac:groups=syn.tools,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=syn.tools,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=syn.tools,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;roles,verbs=get;list;watch;create;update;patch;delete

func (r *ClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling Cluster")

	instance := &synv1alpha1.Cluster{}

	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	data := &pipeline.Context{
		Context:               ctx,
		Client:                r.Client,
		Log:                   reqLogger,
		FinalizerName:         synv1alpha1.FinalizerName,
		Reconciler:            r,
		CreateSATokenSecret:   r.CreateSATokenSecret,
		DefaultCreationPolicy: r.DefaultCreationPolicy,
	}

	steps := []pipeline.Step{
		{Name: "copy original object", F: pipeline.DeepCopyOriginal},
		{Name: "cluster specific steps", F: cluster.SpecificSteps},
		{Name: "create git repo", F: gitrepo.CreateOrUpdate},
		{Name: "set gitrepo url and hostkeys", F: gitrepo.UpdateURLAndHostKeys},
		{Name: "add tenant label", F: pipeline.AddTenantLabel},
		{Name: "Common", F: pipeline.Common},
	}

	res := pipeline.RunPipeline(instance, data, steps)

	return reconcile.Result{Requeue: res.Requeue}, res.Err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synv1alpha1.Cluster{}).
		Owns(&synv1alpha1.GitRepo{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Secret{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
