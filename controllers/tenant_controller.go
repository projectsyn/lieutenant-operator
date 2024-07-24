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
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
	DefaultDeletionPolicy synv1alpha1.DeletionPolicy

	DefaultGlobalGitRepoUrl string
	DeleteProtection        bool
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
		Context:                 ctx,
		Client:                  r.Client,
		Log:                     reqLogger,
		FinalizerName:           "",
		Reconciler:              r,
		CreateSATokenSecret:     r.CreateSATokenSecret,
		DefaultCreationPolicy:   r.DefaultCreationPolicy,
		DefaultDeletionPolicy:   r.DefaultDeletionPolicy,
		DefaultGlobalGitRepoUrl: r.DefaultGlobalGitRepoUrl,
		UseDeletionProtection:   r.DeleteProtection,
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
		// Reconcile all tenants when a TenantTemplate changes to ensure that all tenants are up to date
		Watches(&synv1alpha1.TenantTemplate{}, handler.EnqueueRequestsFromMapFunc(enqueueAllTenantsMapFunc(mgr.GetClient()))).
		Complete(r)
}

// enqueueAllTenantsMapFunc returns a function that lists all tenants in the namespace of the given object and returns a list of reconcile.Requests for all of them.
func enqueueAllTenantsMapFunc(cli client.Client) func(ctx context.Context, o client.Object) []reconcile.Request {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		l := log.FromContext(ctx).WithName("enqueueAllTenantsMapFunc")
		l.Info("Enqueue all tenants")

		tenants := &synv1alpha1.TenantList{}
		err := cli.List(ctx, tenants, client.InNamespace(o.GetNamespace()))
		if err != nil {
			l.Error(err, "Failed to list tenants")
			return []reconcile.Request{}
		}

		requests := make([]reconcile.Request, len(tenants.Items))
		for i, tenant := range tenants.Items {
			requests[i] = reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: tenant.Namespace,
					Name:      tenant.Name,
				},
			}
		}
		return requests
	}
}
