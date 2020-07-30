package pipeline

// Function defines the general form of a pipeline function.
type Function func(PipelineObject, *ExecutionContext) ExecutionResult

func ReconcileTenant(obj PipelineObject, data *ExecutionContext) error {

	result := tenantSpecificSteps(obj, data)
	if resultNotOK(result) {
		return wrapError("tenant specific steps", result.Err)
	}

	result = createOrUpdateGitRepo(obj, data)
	if resultNotOK(result) {
		return wrapError("create or uptdate git repo", result.Err)
	}

	result = setGitRepoURLAndHostKeys(obj, data)
	if resultNotOK(result) {
		return wrapError("set gitrepo url and hostkeys", result.Err)
	}

	result = common(obj, data)
	if resultNotOK(result) {
		return wrapError("common", result.Err)
	}
	return nil
}

func ReconcileCluster(obj PipelineObject, data *ExecutionContext) error {

	result := clusterSpecificSteps(obj, data)
	if resultNotOK(result) {
		return wrapError("cluster specific steps failes", result.Err)
	}

	result = createOrUpdateGitRepo(obj, data)
	if resultNotOK(result) {
		return wrapError("create or uptdate git repo", result.Err)
	}

	result = setGitRepoURLAndHostKeys(obj, data)
	if resultNotOK(result) {
		return wrapError("set gitrepo url and hostkeys", result.Err)
	}

	result = common(obj, data)
	if resultNotOK(result) {
		return wrapError("common", result.Err)
	}

	return nil
}

func ReconcileGitRep(obj PipelineObject, data *ExecutionContext) error {

	result := checkIfDeleted(obj, data)
	if resultNotOK(result) {
		return wrapError("deletion check", result.Err)
	}

	result = gitRepoSpecificSteps(obj, data)
	if resultNotOK(result) {
		return wrapError("cluster specific steps failes", result.Err)
	}

	result = common(obj, data)
	if resultNotOK(result) {
		return wrapError("common", result.Err)
	}

	return nil
}
