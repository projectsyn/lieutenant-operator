// pipeline contains pipelines that define all the steps that a CRD has to go
// through in order to be considered reconciled.

package pipeline

const (
	protectionSettingEnvVar    = "LIEUTENANT_DELETE_PROTECTION"
	DeleteProtectionAnnotation = "syn.tools/protected-delete"
)

func common(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	steps := []Step{
		{Name: "deletion", F: handleDeletion},
		{Name: "add deletion protection", F: addDeletionProtection},
		{Name: "handle finalizer", F: handleFinalizer},
		{Name: "update object", F: updateObject},
	}

	return RunPipeline(obj, data, steps)
}
