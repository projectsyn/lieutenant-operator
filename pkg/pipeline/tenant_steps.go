package pipeline

import (
	"context"
	"fmt"
	"os"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addDefaultClassFile(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	commonClassFile := CommonClassName + ".yml"
	if obj.GetGitTemplate().TemplateFiles == nil {
		obj.GetGitTemplate().TemplateFiles = map[string]string{}
	}
	if _, ok := obj.GetGitTemplate().TemplateFiles[commonClassFile]; !ok {
		obj.GetGitTemplate().TemplateFiles[commonClassFile] = ""
	}
	return ExecutionResult{}
}

func updateTenantGitRepo(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	tenantCR, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return ExecutionResult{Err: fmt.Errorf("object is not a tenant")}
	}

	var oldFiles map[string]string
	if tenantCR.Spec.GitRepoTemplate != nil {
		oldFiles = tenantCR.Spec.GitRepoTemplate.TemplateFiles
	} else {
		tenantCR.Spec.GitRepoTemplate = &synv1alpha1.GitRepoTemplate{}
	}

	tenantCR.Spec.GitRepoTemplate.TemplateFiles = map[string]string{}

	clusterList := &synv1alpha1.ClusterList{}

	selector := labels.Set(map[string]string{apis.LabelNameTenant: tenantCR.Name}).AsSelector()

	listOptions := &client.ListOptions{
		LabelSelector: selector,
		Namespace:     obj.GetObjectMeta().GetNamespace(),
	}

	err := data.Client.List(context.TODO(), clusterList, listOptions)
	if err != nil {
		return ExecutionResult{Err: err}
	}

	for _, cluster := range clusterList.Items {
		fileName := cluster.GetName() + ".yml"
		fileContent := fmt.Sprintf(clusterClassContent, tenantCR.Name, CommonClassName)
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

	return ExecutionResult{}
}

func setGlobalGitRepoURL(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	instance, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return ExecutionResult{Err: fmt.Errorf("object is not a tenant")}
	}

	defaultGlobalGitRepoURL := os.Getenv(DefaultGlobalGitRepoURL)
	if len(instance.Spec.GlobalGitRepoURL) == 0 && len(defaultGlobalGitRepoURL) > 0 {
		instance.Spec.GlobalGitRepoURL = defaultGlobalGitRepoURL
	}
	return ExecutionResult{}
}
