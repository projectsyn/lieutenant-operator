package gitrepo

import (
	"context"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func fetchGitRepoTemplate(obj *synv1alpha1.GitRepo, data *pipeline.Context) error {
	tenant := &synv1alpha1.Tenant{}

	tenantName := types.NamespacedName{Name: obj.GetObjectMeta().GetName(), Namespace: obj.GetObjectMeta().GetNamespace()}

	err := data.Client.Get(context.TODO(), tenantName, tenant)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if tenant.Spec.GitRepoTemplate != nil {
		obj.Spec.GitRepoTemplate = *tenant.Spec.GitRepoTemplate
	}

	cluster := &synv1alpha1.Cluster{}

	clusterName := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}

	err = data.Client.Get(context.TODO(), clusterName, cluster)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if cluster.Spec.GitRepoTemplate != nil {
		obj.Spec.GitRepoTemplate = *cluster.Spec.GitRepoTemplate
	}

	return nil
}
