package controllers

import (
	"context"

	"github.com/projectsyn/lieutenant-operator/controllers/gitrepo"
	"github.com/projectsyn/lieutenant-operator/controllers/tenant"
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

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	CreateSATokenSecret   bool
	DefaultCreationPolicy synv1alpha1.CreationPolicy
}

//+kubebuilder:rbac:groups=syn.tools,resources=tenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=syn.tools,resources=tenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=syn.tools,resources=tenants/finalizers,verbs=update
//+kubebuilder:rbac:groups=syn.tools,resources=tenanttemplates,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;roles,verbs=get;list;watch;create;update;patch;delete

// Reconcile The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *TenantReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling Tenant")

	// Fetch the Tenant instance
	instance := &synv1alpha1.Tenant{}
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
		FinalizerName:         "",
		Reconciler:            r,
		CreateSATokenSecret:   r.CreateSATokenSecret,
		DefaultCreationPolicy: r.DefaultCreationPolicy,
	}

	steps := []pipeline.Step{
		{Name: "copy original object", F: pipeline.DeepCopyOriginal},
		{Name: "tenant specific steps", F: tenant.Steps},
		{Name: "create git repo", F: gitrepo.CreateOrUpdate},
		{Name: "set gitrepo url and hostkeys", F: gitrepo.UpdateURLAndHostKeys},
		{Name: "common", F: pipeline.Common},
	}
	res := pipeline.RunPipeline(instance, data, steps)

	return reconcile.Result{Requeue: res.Requeue}, res.Err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synv1alpha1.Tenant{}).
		Owns(&synv1alpha1.GitRepo{}).
		Owns(&synv1alpha1.Cluster{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Secret{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
