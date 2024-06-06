package metrics_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/metrics"
)

func Test_ClusterInfoCollector(t *testing.T) {
	namespace := "testns"
	expectedMetricNames := []string{
		"syn_lieutenant_cluster_info",
		"syn_lieutenant_cluster_facts",
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
				DisplayName: "Cluster 2",
				TenantRef: corev1.LocalObjectReference{
					Name: "t2",
				},
				Facts: map[string]string{
					"key":                            "value",
					"_key":                           "value",
					"0key-duplicate-after-normalize": "value",
					"1key-duplicate-after-normalize": "value",
					"2key-duplicate-after-normalize": "value",
					"key-with847_💩_â-invalid-chars":  "value",
					"cluster":                        "value",
					"tenant":                         "value",
				},
			},
		},
	)

	subject := &metrics.ClusterInfoCollector{
		Client: c,

		Namespace: namespace,
	}

	metrics := `# HELP syn_lieutenant_cluster_facts Lieutenant cluster facts. Keys are normalized to be valid Prometheus labels.
# TYPE syn_lieutenant_cluster_facts gauge
syn_lieutenant_cluster_facts{cluster="c-empty",tenant=""} 1
syn_lieutenant_cluster_facts{cluster="c2",fact__key="value",fact__key_duplicate_after_normalize="value",fact__key_duplicate_after_normalize_1="value",fact__key_duplicate_after_normalize_2="value",key="value",key_with847_____invalid_chars="value",orig_cluster="value",orig_tenant="value",tenant="t2"} 1
# HELP syn_lieutenant_cluster_info Cluster information metric.
# TYPE syn_lieutenant_cluster_info gauge
syn_lieutenant_cluster_info{cluster="c-empty",display_name="",tenant=""} 1
syn_lieutenant_cluster_info{cluster="c2",display_name="Cluster 2",tenant="t2"} 1
`
	require.NoError(t,
		testutil.CollectAndCompare(subject, strings.NewReader(metrics), expectedMetricNames...),
	)
}

func Test_ClusterInfoCollector_ListFail(t *testing.T) {
	namespace := "testns"

	listErr := errors.New("whoopsie daisy")

	c := prepareFailingClient(t, listErr)

	subject := &metrics.ClusterInfoCollector{
		Client: c,

		Namespace: namespace,
	}

	require.ErrorContains(t, testutil.CollectAndCompare(subject, strings.NewReader(``)), listErr.Error())
}
