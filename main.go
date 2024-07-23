package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
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

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/controllers"
	operatorMetrics "github.com/projectsyn/lieutenant-operator/metrics"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	watchNamespaceEnvVar = "WATCH_NAMESPACE"
	// createSAEnvVar is the constant for the env variable which indicates
	// whether to create ServiceAccount token secrets
	createSATokenEnvVar = "LIEUTENANT_CREATE_SERVICEACCOUNT_TOKEN_SECRET"
	// createSAEnvVar is the constant for the env variable which indicates
	// the default creation policy for git repositories
	defaultCreationPolicy = "DEFAULT_CREATION_POLICY"
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
	var apiUrl string
	var probeAddr string
	var gitRepoMaxReconcileInterval time.Duration
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&apiUrl, "lieutenant-api-url", "localhost", "The URL at which the Lieutenant API is available externally")
	flag.DurationVar(&gitRepoMaxReconcileInterval, "git-repo-max-reconcile-interval", 3*time.Hour, "The maximum time between reconciliations of GitRepos.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	watchNamespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get WatchNamespace, "+
			"the manager will watch and manage resources in all namespaces")
	}

	createSATokenSecret, err := getCreateSATokenSecret()
	if err != nil {
		setupLog.Error(err, "unable to get TokenSecret flag, "+
			"the operator won't manage ServiceAccount token secrets.")
	}

	creationPolicy := getDefaultCreationPolicy()

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
		CreateSATokenSecret:   createSATokenSecret,
		DefaultCreationPolicy: creationPolicy,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
	if err = (&controllers.ClusterCompilePipelineReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterCompilePipeline")
		os.Exit(1)
	}
	if err = (&controllers.GitRepoReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		DefaultCreationPolicy: creationPolicy,

		MaxReconcileInterval: gitRepoMaxReconcileInterval,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GitRepo")
		os.Exit(1)
	}
	if err = (&controllers.TenantReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		CreateSATokenSecret:   createSATokenSecret,
		DefaultCreationPolicy: creationPolicy,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Tenant")
		os.Exit(1)
	}
	if err = (&controllers.TenantCompilePipelineReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		ApiUrl: apiUrl,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TenantCompilePipeline")
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

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("environment variable '%s' not found", watchNamespaceEnvVar)
	}
	return ns, nil
}

// getCreateSATokenSecret returns a boolean indicating whether the operator should manage ServiceAccount token secrets
func getCreateSATokenSecret() (bool, error) {
	value, found := os.LookupEnv(createSATokenEnvVar)
	if !found {
		return false, fmt.Errorf("environment variable '%s' not found", createSATokenEnvVar)
	}
	create, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("unable to parse '%s': %v", value, err)
	}
	return create, nil
}

// getDefaultCreationPolicy returns to fallback creation policy for git repositories
func getDefaultCreationPolicy() synv1alpha1.CreationPolicy {
	p, found := os.LookupEnv(defaultCreationPolicy)
	if !found {
		return synv1alpha1.CreatePolicy
	}
	cp := synv1alpha1.CreationPolicy(p)
	switch cp {
	case synv1alpha1.CreatePolicy, synv1alpha1.AdoptPolicy:
		return cp
	default:
		return synv1alpha1.CreatePolicy
	}
}
