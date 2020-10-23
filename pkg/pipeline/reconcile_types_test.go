package pipeline

import synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"

type genericCases map[string]struct {
	args    args
	wantErr bool
}

type args struct {
	cluster       *synv1alpha1.Cluster
	tenant        *synv1alpha1.Tenant
	gitRepo       synv1alpha1.GitRepo
	data          *ExecutionContext
	finalizerName string
}
