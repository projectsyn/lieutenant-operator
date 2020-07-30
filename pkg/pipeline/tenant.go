package pipeline

const (
	// CommonClassName is the name of the tenant's common class
	CommonClassName = "common"
)

func tenantSpecificSteps(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	commonClassFile := CommonClassName + ".yml"
	if obj.GetGitTemplate().TemplateFiles == nil {
		obj.GetGitTemplate().TemplateFiles = map[string]string{}
	}
	if _, ok := obj.GetGitTemplate().TemplateFiles[commonClassFile]; !ok {
		obj.GetGitTemplate().TemplateFiles[commonClassFile] = ""
	}
	return ExecutionResult{}
}
