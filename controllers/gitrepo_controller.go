package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers/gitrepo"
	"github.com/projectsyn/lieutenant-operator/controllers/gitrepo/watchers"
	"github.com/projectsyn/lieutenant-operator/pipeline"
)

// GitRepoReconciler reconciles a GitRepo object
type GitRepoReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	DefaultCreationPolicy synv1alpha1.CreationPolicy

	// MaxReconcileInterval is the maximum time between two reconciliations.
	MaxReconcileInterval time.Duration

	DeleteProtection bool
}

//+kubebuilder:rbac:groups=syn.tools,resources=gitrepos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=syn.tools,resources=gitrepos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=syn.tools,resources=gitrepos/finalizers,verbs=update

// Reconcile will create or delete a git repository based on the event.
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
// ATTENTION: do not manipulate the spec here, this will lead to loops, as the specs are
// defined in the gitrepotemplates of the other CRDs (tenant, cluster).
func (r *GitRepoReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling GitRepo")

	// Fetch the GitRepo instance
	instance := &synv1alpha1.GitRepo{}

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
		DefaultCreationPolicy: r.DefaultCreationPolicy,
		UseDeletionProtection: r.DeleteProtection,
	}

	steps := []pipeline.Step{
		{Name: "copy original object", F: pipeline.DeepCopyOriginal},
		{Name: "deletion check", F: pipeline.CheckIfDeleted},
		{Name: "git repo specific steps", F: gitrepo.Steps},
		{Name: "add tenant label", F: pipeline.AddTenantLabel},
		{Name: "Common", F: pipeline.Common},
	}

	res := pipeline.RunPipeline(instance, data, steps)

	// Immediately requeue if the result says so
	if res.Requeue {
		return reconcile.Result{Requeue: true}, res.Err
	}

	// Requeue after the maximum interval
	return reconcile.Result{
		RequeueAfter: r.MaxReconcileInterval,
	}, res.Err
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitRepoReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&synv1alpha1.GitRepo{},
		watchers.GitRepoCIVariableValueFromSecretKeyRefNameIndex,
		watchers.GitRepoCIVariableValueFromSecretKeyRefNameIndexFunc,
	)
	if err != nil {
		return fmt.Errorf("unable to create index for GitRepo: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&synv1alpha1.GitRepo{}).
		Owns(&corev1.Secret{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(watchers.SecretGitRepoCIVariablesMapFunc(mgr.GetClient())),
		).
		Complete(r)
}
