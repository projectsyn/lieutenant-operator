package pipeline

import (
	"context"
	"fmt"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// CommonClassName is the name of the tenant's common class
	CommonClassName = "common"
)

func tenantSpecificSteps(obj PipelineObject, data *ExecutionContext) ExecutionResult {

	result := addDefaultClassFile(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("add default class file", result.Err)
		return result
	}

	result = updateTenantGitRepo(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("update tenant git repo", result.Err)
		return result
	}

	return ExecutionResult{}

}

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

	oldFiles := tenantCR.Spec.GitRepoTemplate.TemplateFiles

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
		fileContent := fmt.Sprintf(clusterClassContent, cluster.GetName(), CommonClassName)
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
