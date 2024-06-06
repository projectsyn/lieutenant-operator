package metrics

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=syn.tools,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=syn.tools,resources=clusters/status,verbs=get

var clusterInfoDesc = prometheus.NewDesc(
	"syn_lieutenant_cluster_info",
	"Cluster information metric.",
	[]string{"cluster", "tenant", "display_name"},
	nil,
)

var clusterFactsDesc = prometheus.NewDesc(
	"syn_lieutenant_cluster_facts",
	"Lieutenant cluster facts.",
	[]string{"cluster", "tenant", "display_name"},
	nil,
)

// commodore build info has dynamic labels
func newClusterFactsDesc(lbls ...string) *prometheus.Desc {
	return prometheus.NewDesc(
		"syn_lieutenant_cluster_facts",
		"Lieutenant cluster facts. Keys are normalized to be valid Prometheus labels.",
		lbls,
		nil,
	)
}

// ClusterInfoCollector is a Prometheus collector that collects cluster info metrics.
type ClusterInfoCollector struct {
	Client client.Client

	Namespace string
}

var _ prometheus.Collector = &ClusterInfoCollector{}

// Describe implements prometheus.Collector.
// Sends the descriptors of the metrics to the channel.
func (*ClusterInfoCollector) Describe(chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector.
// Iterates over all clusters and sends cluster information for each cluster.
func (m *ClusterInfoCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	cls := synv1alpha1.ClusterList{}
	if err := m.Client.List(ctx, &cls, client.InNamespace(m.Namespace)); err != nil {
		err := fmt.Errorf("failed to list clusters: %w", err)
		ch <- prometheus.NewInvalidMetric(clusterInfoDesc, err)
	}

	for _, cl := range cls.Items {
		ch <- prometheus.MustNewConstMetric(
			clusterInfoDesc,
			prometheus.GaugeValue,
			1,
			cl.Name, cl.Spec.TenantRef.Name, cl.Spec.DisplayName,
		)

		if err := clusterFacts(cl, ch); err != nil {
			log.Log.Info("failed to collect cluster facts", "error", err)
		}
	}
}

// clusterFacts collects the facts of a cluster and sends them as Prometheus metrics.
// The keys of the facts are normalized to be valid Prometheus labels.
// If the first character of a key is an underscore or an invalid character it is replaced with "fact_".
// If a key is empty it is replaced with "_empty".
// If a key is in the protected list after normalizing it is prefixed with "orig_".
// If a key is a duplicate after normalizing it is suffixed with "_<n>" where n is the number of duplicates.
func clusterFacts(cl synv1alpha1.Cluster, ch chan<- prometheus.Metric) error {
	rks, vs := pairs(cl.Spec.Facts)
	ks := make([]string, len(rks))
	for i, k := range rks {
		ks[i] = normalizeLabelKey(k, []string{"cluster", "tenant"}, "fact_")
	}
	seen := make(map[string]int)
	for i, k := range ks {
		if _, ok := seen[k]; ok {
			ks[i] = fmt.Sprintf("%s_%d", k, seen[k])
		}
		seen[k]++
	}

	m, err := prometheus.NewConstMetric(
		newClusterFactsDesc(append([]string{"cluster", "tenant"}, ks...)...),
		prometheus.GaugeValue,
		1,
		append([]string{cl.Name, cl.Spec.TenantRef.Name}, vs...)...,
	)
	if err != nil {
		return fmt.Errorf("failed to create metric for cluster %q: %w", cl.Name, err)
	}
	ch <- m
	return nil
}

// https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
var validKeyCharacters = regexp.MustCompile(`(?:^[^a-zA-Z_]|[^a-zA-Z0-9_])`)

// normalizeLabelKey normalizes a key to be a valid Prometheus metric name.
// It replaces invalid characters with underscores and prefixes the key with the given prefix if it starts with an underscore character.
// If the key is empty it returns "_empty".
// If the key is in the protected list after normalizing it prefixes the key with "orig_".
func normalizeLabelKey(key string, protected []string, prefixForUnderscore string) string {
	if key == "" {
		return "_empty"
	}

	key = validKeyCharacters.ReplaceAllLiteralString(key, "_")
	if strings.HasPrefix(key, "_") {
		key = prefixForUnderscore + key
	}
	if slices.Contains(protected, key) {
		key = "orig_" + key
	}

	return key
}
