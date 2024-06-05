package metrics

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=syn.tools,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=syn.tools,resources=clusters/status,verbs=get

const compileMetaMetricPrefix = "syn_lieutenant_cluster_compile_meta"

var metaLastCompileDesc = prometheus.NewDesc(
	compileMetaMetricPrefix+"_last_compile",
	"The timestamp of the last cluster compilation as unix timestamp in seconds.",
	[]string{"cluster", "tenant"},
	nil,
)

// commodore build info has dynamic labels
func newMetaCommodoreBuildInfoDesc(lbls ...string) *prometheus.Desc {
	return prometheus.NewDesc(
		compileMetaMetricPrefix+"_commodore_build_info",
		"Commodore reported build information.",
		lbls,
		nil,
	)
}

var metaGlobalDesc = prometheus.NewDesc(
	compileMetaMetricPrefix+"_global",
	"Version information of the global defaults Commodore repository.",
	[]string{"cluster", "tenant", "url", "path", "version", "gitSha"},
	nil,
)

var metaTenantDesc = prometheus.NewDesc(
	compileMetaMetricPrefix+"_tenant",
	"Version information of the tenant Commodore repository.",
	[]string{"cluster", "tenant", "url", "path", "version", "gitSha"},
	nil,
)

var metaPackageDesc = prometheus.NewDesc(
	compileMetaMetricPrefix+"_package",
	"Version information of the used Commodore package repositories.",
	[]string{"cluster", "tenant", "name", "url", "path", "version", "gitSha"},
	nil,
)

var metaInstanceDesc = prometheus.NewDesc(
	compileMetaMetricPrefix+"_instance",
	"Version information of the used Commodore component instance repositories.",
	[]string{"cluster", "tenant", "name", "url", "path", "version", "gitSha", "component"},
	nil,
)

// CompileMetaCollector is a Prometheus collector that translates the `status.compileMeta` field of the Cluster CRD into Prometheus metrics.
type CompileMetaCollector struct {
	Client client.Client

	Namespace string
}

var _ prometheus.Collector = &CompileMetaCollector{}

// Describe implements prometheus.Collector.
// This collector does not send any descriptions and thus makes the collector unchecked.
// An unchecked collector is needed because the commodore build info metric is dynamic and has no static description.
func (*CompileMetaCollector) Describe(_ chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector.
// Iterates over all clusters and sends compileMeta metrics for each cluster.
func (m *CompileMetaCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	cls := synv1alpha1.ClusterList{}
	if err := m.Client.List(ctx, &cls, client.InNamespace(m.Namespace)); err != nil {
		err := fmt.Errorf("failed to list clusters: %w", err)
		ch <- prometheus.NewInvalidMetric(metaLastCompileDesc, err)
		ch <- prometheus.NewInvalidMetric(newMetaCommodoreBuildInfoDesc(), err)
		ch <- prometheus.NewInvalidMetric(metaGlobalDesc, err)
		ch <- prometheus.NewInvalidMetric(metaTenantDesc, err)
		ch <- prometheus.NewInvalidMetric(metaPackageDesc, err)
		ch <- prometheus.NewInvalidMetric(metaInstanceDesc, err)
	}

	for _, cl := range cls.Items {
		ch <- prometheus.MustNewConstMetric(
			metaLastCompileDesc,
			prometheus.GaugeValue,
			float64(cl.Status.CompileMeta.LastCompile.Unix()),
			cl.Name, cl.Spec.TenantRef.Name,
		)

		ks, vs := pairs(cl.Status.CompileMeta.CommodoreBuildInfo)
		ks = append(ks, "cluster", "tenant")
		vs = append(vs, cl.Name, cl.Spec.TenantRef.Name)
		m, err := prometheus.NewConstMetric(
			newMetaCommodoreBuildInfoDesc(ks...),
			prometheus.GaugeValue,
			1,
			vs...,
		)
		if err != nil {
			log.Log.Info("failed to create metric", "error", err)
		} else {
			ch <- m
		}

		ch <- prometheus.MustNewConstMetric(
			metaGlobalDesc,
			prometheus.GaugeValue,
			1,
			cl.Name, cl.Spec.TenantRef.Name, cl.Status.CompileMeta.Global.URL, "", cl.Status.CompileMeta.Global.Version, cl.Status.CompileMeta.Global.GitSHA,
		)

		ch <- prometheus.MustNewConstMetric(
			metaTenantDesc,
			prometheus.GaugeValue,
			1,
			cl.Name, cl.Spec.TenantRef.Name, cl.Status.CompileMeta.Tenant.URL, "", cl.Status.CompileMeta.Tenant.Version, cl.Status.CompileMeta.Tenant.GitSHA,
		)

		for name, repo := range cl.Status.CompileMeta.Packages {
			ch <- prometheus.MustNewConstMetric(
				metaPackageDesc,
				prometheus.GaugeValue,
				1,
				cl.Name, cl.Spec.TenantRef.Name, name, repo.URL, repo.Path, repo.Version, repo.GitSHA,
			)
		}

		for name, repo := range cl.Status.CompileMeta.Instances {
			ch <- prometheus.MustNewConstMetric(
				metaInstanceDesc,
				prometheus.GaugeValue,
				1,
				cl.Name, cl.Spec.TenantRef.Name, name, repo.URL, repo.Path, repo.Version, repo.GitSHA, repo.Component,
			)
		}
	}
}

// pairs returns the keys and values of a map as two slices.
// The slices are ordered by keys+values.
func pairs(m map[string]string) (keys []string, values []string) {
	p := make([][2]string, 0, len(m))

	for k, v := range m {
		p = append(p, [2]string{k, v})
	}

	slices.SortFunc(p, func(a, b [2]string) int {
		return 10*strings.Compare(a[0], b[0]) + strings.Compare(a[1], b[1])
	})

	k := make([]string, len(p))
	v := make([]string, len(p))

	for i := range p {
		k[i] = p[i][0]
		v[i] = p[i][1]
	}

	return k, v
}
