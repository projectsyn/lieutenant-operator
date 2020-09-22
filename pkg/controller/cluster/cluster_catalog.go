package cluster

import (
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/helpers"
)

type tmplData struct {
	ClusterID string
	TenantID  string
	Facts     *synv1alpha1.Facts
}

func setClusterCatalog(cluster *synv1alpha1.Cluster, tenant *synv1alpha1.Tenant) error {
	tmplInput := tmplData{
		ClusterID: cluster.Name,
		TenantID:  tenant.Name,
		Facts:     cluster.Spec.Facts,
	}

	if cluster.Spec.GitRepoTemplate == nil {
		cluster.Spec.GitRepoTemplate = &synv1alpha1.GitRepoTemplate{}
	}

	if err := helpers.SetTemplateIfEmpty(
		&cluster.Spec.GitRepoTemplate.RepoName,
		tenant.Spec.ClusterCatalog.GitRepoTemplate.RepoName,
		tmplInput); err != nil {
		return err
	}

	if err := helpers.SetTemplateIfEmpty(
		&cluster.Spec.GitRepoTemplate.Path,
		tenant.Spec.ClusterCatalog.GitRepoTemplate.Path,
		tmplInput); err != nil {
		return err
	}

	if err := helpers.SetTemplateIfEmpty(
		&cluster.Spec.GitRepoTemplate.APISecretRef.Name,
		tenant.Spec.ClusterCatalog.GitRepoTemplate.APISecretRef.Name,
		tmplInput); err != nil {
		return err
	}

	if err := helpers.SetTemplateIfEmpty(
		&cluster.Spec.GitRepoTemplate.APISecretRef.Namespace,
		tenant.Spec.ClusterCatalog.GitRepoTemplate.APISecretRef.Namespace,
		tmplInput); err != nil {
		return err
	}
	return nil
}
