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
		{Name: "apply tenant template", F: applyTenantTemplate},
	}

	err := RunPipeline(obj, data, steps)

	return ExecutionResult{Err: err}
}
