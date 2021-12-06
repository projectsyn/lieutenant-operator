package cluster

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/imdario/mergo"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/ryankurte/go-structparse"
)

type templateParser struct {
	data templateData
	err  error
}

type templateData struct {
	*synv1alpha1.Cluster
	Tenant *synv1alpha1.Tenant
}

func (r *templateParser) ParseString(in string) interface{} {
	if r.err != nil || len(in) == 0 {
		return in
	}
	str, err := RenderTemplate(in, r.data)
	if err != nil {
		r.err = err
		return in
	}
	return str
}

func applyClusterTemplate(cluster *synv1alpha1.Cluster, tenant *synv1alpha1.Tenant) error {
	if tenant.Spec.ClusterTemplate == nil {
		return nil
	}

	// To avoid rendering the template in the actual tenant
	tenant = tenant.DeepCopy()

	parser := &templateParser{
		data: templateData{
			Cluster: cluster,
			Tenant:  tenant,
		},
		err: nil,
	}

	structparse.Strings(parser, tenant.Spec.ClusterTemplate)
	if parser.err != nil {
		return fmt.Errorf("An error occured during template manifestation: %w", parser.err)
	}

	if err := mergo.Merge(&cluster.Spec, tenant.Spec.ClusterTemplate); err != nil {
		return fmt.Errorf("an error occured during cluster template merging: %w", err)
	}

	return nil
}

// RenderTemplate renders a given template with the given data
func RenderTemplate(tmpl string, data interface{}) (string, error) {
	tmp, err := template.New("template").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("could not parse template: %w", err)
	}

	buf := new(bytes.Buffer)
	err = tmp.Execute(buf, data)
	if err != nil {
		return "", fmt.Errorf("could not render template: %w", err)
	}
	return buf.String(), nil
}
