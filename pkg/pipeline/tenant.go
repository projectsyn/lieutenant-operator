package pipeline

const (
	// CommonClassName is the name of the tenant's common class
	CommonClassName         = "common"
	DefaultGlobalGitRepoURL = "DEFAULT_GLOBAL_GIT_REPO_URL"
)

func tenantSpecificSteps(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	steps := []Step{
		{Name: "add default class file", F: addDefaultClassFile},
		{Name: "uptade tenant git repo", F: updateTenantGitRepo},
		{Name: "set global git repo url", F: setGlobalGitRepoURL},
	}

	err := RunPipeline(obj, data, steps)

	return ExecutionResult{Err: err}
}
