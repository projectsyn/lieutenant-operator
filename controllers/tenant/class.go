package tenant

import (
	"github.com/projectsyn/lieutenant-operator/pipeline"
)

const (
	// CommonClassName is the name of the tenant's common class
	CommonClassName         = "common"
	DefaultGlobalGitRepoURL = "DEFAULT_GLOBAL_GIT_REPO_URL"
	ClusterClassContent     = `classes:
- %s.%s
`
)

func addDefaultClassFile(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	commonClassFile := CommonClassName + ".yml"
	if obj.GetGitTemplate().TemplateFiles == nil {
		obj.GetGitTemplate().TemplateFiles = map[string]string{}
	}
	if _, ok := obj.GetGitTemplate().TemplateFiles[commonClassFile]; !ok {
		obj.GetGitTemplate().TemplateFiles[commonClassFile] = ""
	}
	return pipeline.Result{}
}
