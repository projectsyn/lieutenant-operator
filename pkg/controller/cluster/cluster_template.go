package cluster

import (
	"fmt"

	"github.com/imdario/mergo"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"
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
	str, err := helpers.RenderTemplate(in, r.cluster)
	if err != nil {
		r.err = err
		return in
	}
	return str
}

func applyClusterTemplate(cluster *synv1alpha1.Cluster, tenant *synv1alpha1.Tenant) error {
	if cluster.Spec.GitRepoTemplate == nil {
		cluster.Spec.GitRepoTemplate = &synv1alpha1.GitRepoTemplate{}
	}

	if tenant.Spec.ClusterTemplate == nil {
		return nil
	}

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
