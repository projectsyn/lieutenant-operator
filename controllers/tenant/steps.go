package tenant

import (
	"github.com/projectsyn/lieutenant-operator/pipeline"
)

func Steps(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	steps := []pipeline.Step{
		{Name: "apply template from TenantTemplate", F: applyTemplateFromTenantTemplate},
		{Name: "add default class file", F: addDefaultClassFile},
		{Name: "uptade tenant git repo", F: updateTenantGitRepo},
		{Name: "set global git repo url", F: setGlobalGitRepoURL},
		{Name: "create ServiceAccount", F: createServiceAccount},
		{Name: "reconcile Role", F: reconcileRole},
		{Name: "create RoleBinding", F: createRoleBinding},
	}

	return pipeline.RunPipeline(obj, data, steps)
}
