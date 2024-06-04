package cluster

import (
	"github.com/projectsyn/lieutenant-operator/pipeline"
	"github.com/projectsyn/lieutenant-operator/vault"
)

func SpecificSteps(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	steps := []pipeline.Step{
		{Name: "create cluster RBAC", F: createClusterRBAC},
		{Name: "deletion check", F: pipeline.CheckIfDeleted},
		{Name: "set bootstrap token", F: setBootstrapToken},
		{Name: "create or update vault", F: vault.CreateOrUpdateVault},
		{Name: "delete vault entries", F: vault.HandleVaultDeletion},
		{Name: "set tenant owner", F: setTenantOwner},
		{Name: "apply cluster template from tenant", F: applyClusterTemplateFromTenant},
	}

	return pipeline.RunPipeline(obj, data, steps)
}
