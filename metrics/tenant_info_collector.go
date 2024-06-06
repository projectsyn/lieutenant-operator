package metrics

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=syn.tools,resources=tenants,verbs=get;list;watch
//+kubebuilder:rbac:groups=syn.tools,resources=tenants/status,verbs=get

var tenantInfoDesc = prometheus.NewDesc(
	"syn_lieutenant_tenant_info",
	"Tenant information metric.",
	[]string{"tenant", "display_name"},
	nil,
)

// TenantInfoCollector is a Prometheus collector that collects tenant info metrics.
type TenantInfoCollector struct {
	Client client.Client

	Namespace string
}

var _ prometheus.Collector = &TenantInfoCollector{}

// Describe implements prometheus.Collector.
// Sends the descriptors of the metrics to the channel.
func (*TenantInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- tenantInfoDesc
}

// Collect implements prometheus.Collector.
// Iterates over all tenants and sends tenant information for each tenant.
func (m *TenantInfoCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	cls := synv1alpha1.TenantList{}
	if err := m.Client.List(ctx, &cls, client.InNamespace(m.Namespace)); err != nil {
		err := fmt.Errorf("failed to list tenants: %w", err)
		ch <- prometheus.NewInvalidMetric(tenantInfoDesc, err)
	}

	for _, cl := range cls.Items {
		ch <- prometheus.MustNewConstMetric(
			tenantInfoDesc,
			prometheus.GaugeValue,
			1,
			cl.Name, cl.Spec.DisplayName,
		)
	}
}
