package pipeline

import (
	"os"
	"strings"
)

const (
	clusterClassContent = `classes:
- %s.%s
`
)

func clusterSpecificSteps(obj PipelineObject, data *ExecutionContext) ExecutionResult {
	result := createClusterRBAC(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("create cluster RBAC", result.Err)
		return result
	}

	result = checkIfDeleted(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("deletion check", result.Err)
		return result
	}

	result = setBootstrapToken(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("set bootstrap token", result.Err)
		return result
	}

	if strings.ToLower(os.Getenv("SKIP_VAULT_SETUP")) != "true" {
		result = createOrUpdateVault(obj, data)
		if resultNotOK(result) {
			result.Err = wrapError("create or update vault", result.Err)
			return result
		}

		result = handleVaultDeletion(obj, data)
		if resultNotOK(result) {
			result.Err = wrapError("delete vault entries", result.Err)
			return result
		}

	}

	result = setTenantOwner(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("set tenant owner", result.Err)
		return result
	}

	result = applyTenantTemplate(obj, data)
	if resultNotOK(result) {
		result.Err = wrapError("apply tenant template", result.Err)
		return result
	}

	return ExecutionResult{}
}
