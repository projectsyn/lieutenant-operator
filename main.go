package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/kouhin/envflag"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers"
	operatorMetrics "github.com/projectsyn/lieutenant-operator/metrics"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(synv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	ctx := ctrl.SetupSignalHandler()

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var gitRepoMaxReconcileInterval time.Duration

	var skipVaultSetup bool
	var defaultDeletionPolicy string
	var defaultCreationPolicy string
	var useDeleteProtection bool
	var defGlobalGitRepoUrl string
	var watchNamespace string
	var createSaTokenSecret bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.DurationVar(&gitRepoMaxReconcileInterval, "git-repo-max-reconcile-interval", 3*time.Hour, "The maximum time between reconciliations of GitRepos.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&skipVaultSetup, "skip-vault-setup", false, "Set to `true` in order to skip vault setup.")
	flag.StringVar(&defaultDeletionPolicy, "default-deletion-policy", "Archive", "Default deletion policy for git repos. Can be `Delete`, `Retain` or `Archive`.")
	flag.StringVar(&defaultCreationPolicy, "default-creation-policy", "Create", "Default creation policy for git repos. Can be `Create` or `Adopt`.")
	flag.BoolVar(&useDeleteProtection, "lieutenant-delete-protection", false, "Whether to enable deletion protection.")
	flag.StringVar(&defGlobalGitRepoUrl, "default-global-git-repo-url", "", "Default URL for global git repo; used if global git repo isn't explicitly configured.")
	flag.StringVar(&watchNamespace, "watch-namespace", "default", "The namespace which should be watched by the operator")
	flag.BoolVar(&createSaTokenSecret, "lieutenant-create-serviceaccount-token-secret", false, "Whether Lieutenant should create ServiceAccount token secrets")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	err := envflag.Parse()
	if err != nil {
		// The setupLog is not working yet at this point; resort to fmt.Printf
		fmt.Printf("unable to parse flags and environment variables: %s", err.Error())
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	creationPolicy := getDefaultCreationPolicy(defaultCreationPolicy)
	deletionPolicy := getDefaultDeletionPolicy(defaultDeletionPolicy)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9b76ddd4.syn.tools",
		// Limit the manager to only watch the given namespace
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			opts.DefaultNamespaces = map[string]cache.Config{
				watchNamespace: {},
			}
			return cache.New(config, opts)
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err := mgr.GetFieldIndexer().IndexField(ctx, &synv1alpha1.Cluster{}, "spec.tenantRef.name", func(o client.Object) []string {
		cluster := o.(*synv1alpha1.Cluster)
		return []string{cluster.Spec.TenantRef.Name}
	}); err != nil {
		setupLog.Error(err, "unable to create tenantRef field indexer for Cluster")
		os.Exit(1)
	}

	metrics.Registry.MustRegister(&operatorMetrics.CompileMetaCollector{
		Client:    mgr.GetClient(),
		Namespace: watchNamespace,
	})
	metrics.Registry.MustRegister(&operatorMetrics.ClusterInfoCollector{
		Client:    mgr.GetClient(),
		Namespace: watchNamespace,
	})
	metrics.Registry.MustRegister(&operatorMetrics.TenantInfoCollector{
		Client:    mgr.GetClient(),
		Namespace: watchNamespace,
	})

	if err = (&controllers.ClusterReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		CreateSATokenSecret:   createSaTokenSecret,
		DefaultCreationPolicy: creationPolicy,
		DefaultDeletionPolicy: deletionPolicy,
		DeleteProtection:      useDeleteProtection,
		UseVault:              !skipVaultSetup,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
	if err = (&controllers.GitRepoReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		DefaultCreationPolicy: creationPolicy,
		DeleteProtection:      useDeleteProtection,
		MaxReconcileInterval:  gitRepoMaxReconcileInterval,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GitRepo")
		os.Exit(1)
	}
	if err = (&controllers.TenantReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		CreateSATokenSecret:     createSaTokenSecret,
		DefaultCreationPolicy:   creationPolicy,
		DefaultDeletionPolicy:   deletionPolicy,
		DefaultGlobalGitRepoUrl: defGlobalGitRepoUrl,
		DeleteProtection:        useDeleteProtection,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Tenant")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// getDefaultCreationPolicy returns to fallback creation policy for git repositories
func getDefaultCreationPolicy(stringArg string) synv1alpha1.CreationPolicy {
	cp := synv1alpha1.CreationPolicy(stringArg)
	switch cp {
	case synv1alpha1.CreatePolicy, synv1alpha1.AdoptPolicy:
		return cp
	default:
		return synv1alpha1.CreatePolicy
	}
}

func getDefaultDeletionPolicy(stringArg string) synv1alpha1.DeletionPolicy {
	cp := synv1alpha1.DeletionPolicy(stringArg)
	switch cp {
	case synv1alpha1.DeletePolicy, synv1alpha1.RetainPolicy, synv1alpha1.ArchivePolicy:
		return cp
	default:
		return synv1alpha1.ArchivePolicy
	}
}
