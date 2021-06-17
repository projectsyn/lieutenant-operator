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
		{Name: "create Role", F: createRole},
		{Name: "create RoleBinding", F: createRoleBinding},
		{Name: "update Role", F: tenantUpdateRole},
	}

	return pipeline.RunPipeline(obj, data, steps)
}
