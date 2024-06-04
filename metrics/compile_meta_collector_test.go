package metrics_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/metrics"
)

func Test_ClusterUpgradingMetric(t *testing.T) {
	namespace := "testns"
	expectedMetricNames := []string{
		"syn_lieutenant_cluster_compile_meta_last_compile",
		"syn_lieutenant_cluster_compile_meta_commodore_build_info",
		"syn_lieutenant_cluster_compile_meta_package",
		"syn_lieutenant_cluster_compile_meta_instance",
		"syn_lieutenant_cluster_compile_meta_global",
		"syn_lieutenant_cluster_compile_meta_tenant",
	}

	c := prepareClient(t,
		&synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "c-empty",
			},
		},
		&synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "c2",
			},
			Spec: synv1alpha1.ClusterSpec{
				TenantRef: corev1.LocalObjectReference{
					Name: "t2",
				},
			},
			Status: synv1alpha1.ClusterStatus{
				CompileMeta: synv1alpha1.CompileMeta{
					LastCompile: metav1.Time{Time: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
					CommodoreBuildInfo: map[string]string{
						"version":    "1.0.0",
						"gitVersion": "abc123",
					},
					Global: synv1alpha1.CompileMetaVersionInfo{
						URL:     "git.example.com/global",
						GitSHA:  "4a452469d427806459f4aa7f875389ee6e325f8f",
						Version: "main",
					},
					Tenant: synv1alpha1.CompileMetaVersionInfo{
						URL:     "git.example.com/tenant",
						GitSHA:  "edc7dae55537ab49312eaf699e8af3d4d953b775",
						Version: "main",
					},
					Packages: map[string]synv1alpha1.CompileMetaVersionInfo{
						"package1": {
							URL:     "git.example.com/package1",
							GitSHA:  "76b8632b4626ba3a4786da470a3f98911da6cc5d",
							Version: "v1.0.0",
						},
						"package2": {
							URL:     "git.example.com/package2",
							GitSHA:  "5957867eaa9f5a817f63ad7661dd3f58210f96a8",
							Version: "chore/version-bump",
						},
					},
					Instances: map[string]synv1alpha1.CompileMetaInstanceVersionInfo{
						"instance1": {
							Component: "helm",
							CompileMetaVersionInfo: synv1alpha1.CompileMetaVersionInfo{
								URL:     "git.example.com/component-helm",
								GitSHA:  "a05fe3d372a8976d280c9c7e88e51b273886be4d",
								Version: "v7.2.0",
							},
						},
						"alerting": {
							Component: "alerting",
							CompileMetaVersionInfo: synv1alpha1.CompileMetaVersionInfo{
								URL:     "git.example.com/component-ops",
								GitSHA:  "428361d827a02685c132b8926fbd0a299e5bd4e9",
								Version: "main",
								Path:    "alerting",
							},
						},
					},
				},
			},
		},
	)

	subject := &metrics.CompileMetaCollector{
		Client: c,

		Namespace: namespace,
	}

	metrics := `# HELP syn_lieutenant_cluster_compile_meta_commodore_build_info Commodore reported build information.
# TYPE syn_lieutenant_cluster_compile_meta_commodore_build_info gauge
syn_lieutenant_cluster_compile_meta_commodore_build_info{cluster="c-empty",tenant=""} 1
syn_lieutenant_cluster_compile_meta_commodore_build_info{cluster="c2",gitVersion="abc123",tenant="t2",version="1.0.0"} 1
# HELP syn_lieutenant_cluster_compile_meta_global Version information of the global defaults Commodore repository.
# TYPE syn_lieutenant_cluster_compile_meta_global gauge
syn_lieutenant_cluster_compile_meta_global{cluster="c-empty",gitSha="",path="",tenant="",url="",version=""} 1
syn_lieutenant_cluster_compile_meta_global{cluster="c2",gitSha="4a452469d427806459f4aa7f875389ee6e325f8f",path="",tenant="t2",url="git.example.com/global",version="main"} 1
# HELP syn_lieutenant_cluster_compile_meta_instance Version information of the used Commodore component instance repositories.
# TYPE syn_lieutenant_cluster_compile_meta_instance gauge
syn_lieutenant_cluster_compile_meta_instance{cluster="c2",component="alerting",gitSha="428361d827a02685c132b8926fbd0a299e5bd4e9",name="alerting",path="alerting",tenant="t2",url="git.example.com/component-ops",version="main"} 1
syn_lieutenant_cluster_compile_meta_instance{cluster="c2",component="helm",gitSha="a05fe3d372a8976d280c9c7e88e51b273886be4d",name="instance1",path="",tenant="t2",url="git.example.com/component-helm",version="v7.2.0"} 1
# HELP syn_lieutenant_cluster_compile_meta_last_compile The timestamp of the last cluster compilation as unix timestamp in seconds.
# TYPE syn_lieutenant_cluster_compile_meta_last_compile gauge
syn_lieutenant_cluster_compile_meta_last_compile{cluster="c-empty",tenant=""} -6.21355968e+10
syn_lieutenant_cluster_compile_meta_last_compile{cluster="c2",tenant="t2"} 1.6094592e+09
# HELP syn_lieutenant_cluster_compile_meta_package Version information of the used Commodore package repositories.
# TYPE syn_lieutenant_cluster_compile_meta_package gauge
syn_lieutenant_cluster_compile_meta_package{cluster="c2",gitSha="5957867eaa9f5a817f63ad7661dd3f58210f96a8",name="package2",path="",tenant="t2",url="git.example.com/package2",version="chore/version-bump"} 1
syn_lieutenant_cluster_compile_meta_package{cluster="c2",gitSha="76b8632b4626ba3a4786da470a3f98911da6cc5d",name="package1",path="",tenant="t2",url="git.example.com/package1",version="v1.0.0"} 1
# HELP syn_lieutenant_cluster_compile_meta_tenant Version information of the tenant Commodore repository.
# TYPE syn_lieutenant_cluster_compile_meta_tenant gauge
syn_lieutenant_cluster_compile_meta_tenant{cluster="c-empty",gitSha="",path="",tenant="",url="",version=""} 1
syn_lieutenant_cluster_compile_meta_tenant{cluster="c2",gitSha="edc7dae55537ab49312eaf699e8af3d4d953b775",path="",tenant="t2",url="git.example.com/tenant",version="main"} 1
`
	require.NoError(t,
		testutil.CollectAndCompare(subject, strings.NewReader(metrics), expectedMetricNames...),
	)
}

func Test_ClusterUpgradingMetric_ListFail(t *testing.T) {
	namespace := "testns"

	listErr := errors.New("whoopsie daisy")

	c := prepareFailingClient(t, listErr)

	subject := &metrics.CompileMetaCollector{
		Client: c,

		Namespace: namespace,
	}

	require.ErrorContains(t, testutil.CollectAndCompare(subject, strings.NewReader(``)), listErr.Error())
}

func Test_ClusterUpgradingMetric_InvalidMetricNoFailSkip_LabelError(t *testing.T) {
	t.Log("Invalid labels should not cause a full failure, but be skipped.")

	namespace := "testns"
	expectedMetricNames := []string{
		"syn_lieutenant_cluster_compile_meta_commodore_build_info",
	}

	c := prepareClient(t,
		&synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "c-empty",
			},
		},
		&synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "c2",
			},
			Spec: synv1alpha1.ClusterSpec{
				TenantRef: corev1.LocalObjectReference{
					Name: "t2",
				},
			},
			Status: synv1alpha1.ClusterStatus{
				CompileMeta: synv1alpha1.CompileMeta{
					CommodoreBuildInfo: map[string]string{
						"version":    "1.0.0",
						"gitVersion": "abc123",
					},
				},
			},
		},
		&synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "c3",
			},
			Spec: synv1alpha1.ClusterSpec{
				TenantRef: corev1.LocalObjectReference{
					Name: "t3",
				},
			},
			Status: synv1alpha1.ClusterStatus{
				CompileMeta: synv1alpha1.CompileMeta{
					CommodoreBuildInfo: map[string]string{
						"version":             "1.0.0",
						"_invalid-label-2345": "abc123",
					},
				},
			},
		},
	)

	subject := &metrics.CompileMetaCollector{
		Client: c,

		Namespace: namespace,
	}

	metrics := `
# HELP syn_lieutenant_cluster_compile_meta_commodore_build_info Commodore reported build information.
# TYPE syn_lieutenant_cluster_compile_meta_commodore_build_info gauge
syn_lieutenant_cluster_compile_meta_commodore_build_info{cluster="c-empty",tenant=""} 1
syn_lieutenant_cluster_compile_meta_commodore_build_info{cluster="c2",gitVersion="abc123",tenant="t2",version="1.0.0"} 1
`

	require.NoError(t,
		testutil.CollectAndCompare(subject, strings.NewReader(metrics), expectedMetricNames...),
	)
}

func prepareClient(t *testing.T, initObjs ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, synv1alpha1.AddToScheme(scheme))

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjs...).
		WithStatusSubresource(
			&synv1alpha1.Cluster{},
		).
		Build()

	return client
}

type failingClient struct {
	client.Client
	Err error
}

func (f *failingClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return f.Err
}

func prepareFailingClient(t *testing.T, err error) client.Client {
	c := prepareClient(t)
	return &failingClient{Client: c, Err: err}
}
