package metrics_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/metrics"
)

func Test_TenantInfoCollector(t *testing.T) {
	namespace := "testns"
	expectedMetricNames := []string{
		"syn_lieutenant_tenant_info",
	}

	c := prepareClient(t,
		&synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "t-empty",
			},
		},
		&synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "t2",
			},
			Spec: synv1alpha1.TenantSpec{
				DisplayName: "Tenant 2",
			},
		},
	)

	subject := &metrics.TenantInfoCollector{
		Client: c,

		Namespace: namespace,
	}

	metrics := `# HELP syn_lieutenant_tenant_info Tenant information metric.
# TYPE syn_lieutenant_tenant_info gauge
syn_lieutenant_tenant_info{display_name="",tenant="t-empty"} 1
syn_lieutenant_tenant_info{display_name="Tenant 2",tenant="t2"} 1
`
	require.NoError(t,
		testutil.CollectAndCompare(subject, strings.NewReader(metrics), expectedMetricNames...),
	)
}

func Test_TenantInfoCollector_ListFail(t *testing.T) {
	namespace := "testns"

	listErr := errors.New("whoopsie daisy")

	c := prepareFailingClient(t, listErr)

	subject := &metrics.TenantInfoCollector{
		Client: c,

		Namespace: namespace,
	}

	require.ErrorContains(t, testutil.CollectAndCompare(subject, strings.NewReader(``)), listErr.Error())
}
