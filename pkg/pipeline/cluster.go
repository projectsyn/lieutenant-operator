package pipeline

const (
	clusterClassContent = `classes:
- %s.%s
`
)

func clusterSpecificSteps(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	steps := []Step{
		{Name: "create cluster RBAC", F: createClusterRBAC},
		{Name: "deletion check", F: checkIfDeleted},
		{Name: "set bootstrap token", F: setBootstrapToken},
		{Name: "create or update vault", F: createOrUpdateVault},
		{Name: "delete vault entries", F: handleVaultDeletion},
		{Name: "set tenant owner", F: setTenantOwner},
		{Name: "apply cluster template from tenant", F: applyClusterTemplateFromTenant},
	}

	return RunPipeline(obj, data, steps)
}
