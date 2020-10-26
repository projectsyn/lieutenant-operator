package pipeline

// Function defines the general form of a pipeline function.
type Function func(PipelineObject, *ExecutionContext) ExecutionResult

type Step struct {
	Name string
	F    Function
}

func ReconcileTenant(obj PipelineObject, data *ExecutionContext) error {
	steps := []Step{
		{Name: "tenant specific steps", F: tenantSpecificSteps},
		{Name: "create git repo", F: createGitRepo},
		{Name: "set gitrepo url and hostkeys", F: setGitRepoURLAndHostKeys},
		{Name: "common", F: common},
	}

	return RunPipeline(obj, data, steps)
}

func ReconcileCluster(obj PipelineObject, data *ExecutionContext) error {
	steps := []Step{
		{Name: "cluster specific steps", F: clusterSpecificSteps},
		{Name: "create git repo", F: createGitRepo},
		{Name: "set gitrepo url and hostkeys", F: setGitRepoURLAndHostKeys},
		{Name: "add tenant label", F: addTenantLabel},
		{Name: "common", F: common},
	}

	return RunPipeline(obj, data, steps)
}

func ReconcileGitRep(obj PipelineObject, data *ExecutionContext) error {
	steps := []Step{
		{Name: "deletion check", F: checkIfDeleted},
		{Name: "git repo specific steps", F: gitRepoSpecificSteps},
		{Name: "add tenant label", F: addTenantLabel},
		{Name: "common", F: common},
	}

	return RunPipeline(obj, data, steps)
}

func RunPipeline(obj PipelineObject, data *ExecutionContext, steps []Step) error {
	for _, step := range steps {
		if r := step.F(obj, data); resultNotOK(r) {
			return wrapError(step.Name, r.Err)
		}
	}

	return nil
}
