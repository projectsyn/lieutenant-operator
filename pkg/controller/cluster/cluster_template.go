package cluster

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/imdario/mergo"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/ryankurte/go-structparse"
)

type templateParser struct {
	cluster *synv1alpha1.Cluster
	err     error
}

func (r *templateParser) ParseString(in string) interface{} {
	if r.err != nil || len(in) == 0 {
		return in
	}
	str, err := RenderTemplate(in, r.cluster)
	if err != nil {
		r.err = err
		return in
	}
	return str
}

func ApplyClusterTemplate(cluster *synv1alpha1.Cluster, tenant *synv1alpha1.Tenant) error {
	if tenant.Spec.ClusterTemplate == nil {
		return nil
	}

	// To avoid rendering the template in the actual tenant
	tenant = tenant.DeepCopy()

	parser := &templateParser{
		cluster: cluster,
		err:     nil,
	}

	structparse.Strings(parser, tenant.Spec.ClusterTemplate)
	if parser.err != nil {
		return fmt.Errorf("An error occured during template manifestation: %w", parser.err)
	}

	if err := mergo.Merge(&cluster.Spec, tenant.Spec.ClusterTemplate); err != nil {
		return fmt.Errorf("An error occured during cluster template merging: %w", err)
	}

	return nil
}

// RenderTemplate renders a given template with the given data
func RenderTemplate(tmpl string, data interface{}) (string, error) {
	tmp, err := template.New("template").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("Could not parse template: %w", err)
	}
	buf := new(bytes.Buffer)
	if err := tmp.Execute(buf, data); err != nil {
		return "", fmt.Errorf("Could not render template: %w", err)
	}
	return buf.String(), nil
}
