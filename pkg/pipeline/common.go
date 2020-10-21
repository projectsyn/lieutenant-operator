// pipeline contains pipelines that define all the steps that a CRD has to go
// through in order to be considered reconciled.

package pipeline

const (
	protectionSettingEnvVar    = "LIEUTENANT_DELETE_PROTECTION"
	DeleteProtectionAnnotation = "syn.tools/protected-delete"
)

func common(obj PipelineObject, data *ExecutionContext) ExecutionResult {

	result := handleDeletion(obj.GetObjectMeta(), data)
	if result.Abort || result.Err != nil {
		result.Err = wrapError("deletion", result.Err)
		return result
	}

	result = addTenantLabel(obj, data)
	if result.Abort || result.Err != nil {
		result.Err = wrapError("add tenant label", result.Err)
		return result
	}

	result = addDeletionProtection(obj, data)
	if result.Abort || result.Err != nil {
		result.Err = wrapError("add deletion protection", result.Err)
		return result
	}

	result = handleFinalizer(obj, data)
	if result.Abort || result.Err != nil {
		result.Err = wrapError("add deletion protection", result.Err)
		return result
	}

	result = updateObject(obj, data)
	if result.Abort || result.Err != nil {
		result.Err = wrapError("update object", result.Err)
		return result
	}

	return ExecutionResult{}
}
