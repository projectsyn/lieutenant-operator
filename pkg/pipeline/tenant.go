package pipeline

const (
	// CommonClassName is the name of the tenant's common class
	CommonClassName         = "common"
	DefaultGlobalGitRepoURL = "DEFAULT_GLOBAL_GIT_REPO_URL"
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

	result = setGlobalGitRepoURL(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("set global gir repo URL", result.Err)
		return result
	}

	return ExecutionResult{}

}
