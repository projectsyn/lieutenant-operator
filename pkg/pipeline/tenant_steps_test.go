package pipeline

import (
	"os"
	"testing"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var addDefaultClassFileCases = genericCases{
	"add default class": {
		args: args{
			tenant: &synv1alpha1.Tenant{},
			data:   &ExecutionContext{},
		},
	},
}

func Test_addDefaultClassFile(t *testing.T) {
	for name, tt := range addDefaultClassFileCases {
		t.Run(name, func(t *testing.T) {

			got := addDefaultClassFile(tt.args.tenant, tt.args.data)
			assert.NoError(t, got.Err)

			assert.Contains(t, tt.args.tenant.Spec.GitRepoTemplate.TemplateFiles, "common.yml")
			assert.NotEmpty(t, tt.args.tenant.Spec.GitRepoTemplate.TemplateFiles)

		})
	}
}

type setGlobalGitRepoURLArgs struct {
	tenant      *synv1alpha1.Tenant
	defaultRepo string
	data        *ExecutionContext
}

var setGlobalGitRepoURLCases = map[string]struct {
	want string
	args setGlobalGitRepoURLArgs
}{
	"set global git repo url via env var": {
		want: "test",
		args: setGlobalGitRepoURLArgs{
			tenant:      &synv1alpha1.Tenant{},
			defaultRepo: "test",
		},
	},
	"don't override": {
		want: "foo",
		args: setGlobalGitRepoURLArgs{
			tenant: &synv1alpha1.Tenant{
				Spec: synv1alpha1.TenantSpec{
					GlobalGitRepoURL: "foo",
				},
			},
			defaultRepo: "test",
		},
	},
}

func Test_setGlobalGitRepoURL(t *testing.T) {
	for name, tt := range setGlobalGitRepoURLCases {
		t.Run(name, func(t *testing.T) {

			os.Setenv(DefaultGlobalGitRepoURL, tt.args.defaultRepo)

			got := setGlobalGitRepoURL(tt.args.tenant, tt.args.data)
			assert.NoError(t, got.Err)

			assert.Equal(t, tt.want, tt.args.tenant.Spec.GlobalGitRepoURL)

		})
	}
}

var updateTenantGitRepoCases = map[string]struct {
	want *synv1alpha1.GitRepoTemplate
	args args
}{
	"fetch git repos": {
		want: &synv1alpha1.GitRepoTemplate{
			TemplateFiles: map[string]string{
				"testCluster.yml": "classes:\n- testCluster.common\n",
			},
		},
		args: args{
			data: &ExecutionContext{},
			tenant: &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testTenant",
				},
			},
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testCluster",
					Labels: map[string]string{
						apis.LabelNameTenant: "testTenant",
					},
				},
				Spec: synv1alpha1.ClusterSpec{
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						TemplateFiles: map[string]string{
							"testCluster.yml": "classes:\n- testCluster.common\n",
						},
					},
				},
			},
		},
	},
	"remove files": {
		want: &synv1alpha1.GitRepoTemplate{
			TemplateFiles: map[string]string{
				"testCluster.yml": "classes:\n- testCluster.common\n",
				"oldFile.yml":     "{delete}",
			},
		},
		args: args{
			data: &ExecutionContext{},
			tenant: &synv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testTenant",
				},
				Spec: synv1alpha1.TenantSpec{
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						TemplateFiles: map[string]string{
							"oldFile.yml": "not important",
						},
					},
				},
			},
			cluster: &synv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testCluster",
					Labels: map[string]string{
						apis.LabelNameTenant: "testTenant",
					},
				},
				Spec: synv1alpha1.ClusterSpec{
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						TemplateFiles: map[string]string{
							"testCluster.yml": "classes:\n- testCluster.common\n",
						},
					},
				},
			},
		},
	},
}

func Test_updateTenantGitRepo(t *testing.T) {
	for name, tt := range updateTenantGitRepoCases {
		t.Run(name, func(t *testing.T) {

			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.cluster,
				tt.args.tenant,
				&synv1alpha1.ClusterList{},
			})

			got := updateTenantGitRepo(tt.args.tenant, tt.args.data)
			assert.NoError(t, got.Err)

			assert.Equal(t, tt.want, tt.args.tenant.GetGitTemplate())

		})
	}
}
