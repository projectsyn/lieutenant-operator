package tenant

import (
	"context"
	"fmt"
	"os"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/git/manager"
	"github.com/projectsyn/lieutenant-operator/pipeline"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func updateTenantGitRepo(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenantCR, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	var oldFiles map[string]string
	if tenantCR.Spec.GitRepoTemplate != nil {
		oldFiles = tenantCR.Spec.GitRepoTemplate.TemplateFiles
	} else {
		tenantCR.Spec.GitRepoTemplate = &synv1alpha1.GitRepoTemplate{}
	}

	tenantCR.Spec.GitRepoTemplate.TemplateFiles = map[string]string{}

	clusterList := &synv1alpha1.ClusterList{}

	selector := labels.Set(map[string]string{synv1alpha1.LabelNameTenant: tenantCR.Name}).AsSelector()

	listOptions := &client.ListOptions{
		LabelSelector: selector,
		Namespace:     obj.GetNamespace(),
	}

	err := data.Client.List(context.TODO(), clusterList, listOptions)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	for _, cluster := range clusterList.Items {
		fileName := cluster.GetName() + ".yml"
		fileContent := fmt.Sprintf(ClusterClassContent, tenantCR.Name, CommonClassName)
		tenantCR.Spec.GitRepoTemplate.TemplateFiles[fileName] = fileContent
		delete(oldFiles, fileName)
	}

	for fileName := range oldFiles {
		if fileName == CommonClassName+".yml" {
			tenantCR.Spec.GitRepoTemplate.TemplateFiles[CommonClassName+".yml"] = ""
		} else {
			tenantCR.Spec.GitRepoTemplate.TemplateFiles[fileName] = manager.DeletionMagicString

		}
	}

	return pipeline.Result{}
}

func setGlobalGitRepoURL(obj pipeline.Object, _ *pipeline.Context) pipeline.Result {
	instance, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	defaultGlobalGitRepoURL := os.Getenv(DefaultGlobalGitRepoURL)
	if len(instance.Spec.GlobalGitRepoURL) == 0 && len(defaultGlobalGitRepoURL) > 0 {
		instance.Spec.GlobalGitRepoURL = defaultGlobalGitRepoURL
	}
	return pipeline.Result{}
}
