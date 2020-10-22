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

func Test_addDefaultClassFile(t *testing.T) {
	type args struct {
		obj  *synv1alpha1.Tenant
		data *ExecutionContext
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add default class",
			args: args{
				obj:  &synv1alpha1.Tenant{},
				data: &ExecutionContext{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := addDefaultClassFile(tt.args.obj, tt.args.data)
			assert.NoError(t, got.Err)

			assert.Contains(t, tt.args.obj.Spec.GitRepoTemplate.TemplateFiles, "common.yml")
			assert.NotEmpty(t, tt.args.obj.Spec.GitRepoTemplate.TemplateFiles)

		})
	}
}

func Test_setGlobalGitRepoURL(t *testing.T) {
	type args struct {
		obj         *synv1alpha1.Tenant
		data        *ExecutionContext
		defaultRepo string
	}
	tests := []struct {
		name string
		want string
		args args
	}{
		{
			name: "set global git repo url via env var",
			want: "test",
			args: args{
				obj:         &synv1alpha1.Tenant{},
				defaultRepo: "test",
			},
		},
		{
			name: "don't override",
			want: "foo",
			args: args{
				obj: &synv1alpha1.Tenant{
					Spec: synv1alpha1.TenantSpec{
						GlobalGitRepoURL: "foo",
					},
				},
				defaultRepo: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			os.Setenv(DefaultGlobalGitRepoURL, tt.args.defaultRepo)

			got := setGlobalGitRepoURL(tt.args.obj, tt.args.data)
			assert.NoError(t, got.Err)

			assert.Equal(t, tt.want, tt.args.obj.Spec.GlobalGitRepoURL)

		})
	}
}

func Test_updateTenantGitRepo(t *testing.T) {
	type args struct {
		obj     *synv1alpha1.Tenant
		cluster *synv1alpha1.Cluster
		data    *ExecutionContext
	}
	tests := []struct {
		name string
		args args
		want *synv1alpha1.GitRepoTemplate
	}{
		{
			name: "fetch git repos",
			want: &synv1alpha1.GitRepoTemplate{
				TemplateFiles: map[string]string{
					"testCluster.yml": "classes:\n- testCluster.common\n",
				},
			},
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.Tenant{
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
		{
			name: "remove files",
			want: &synv1alpha1.GitRepoTemplate{
				TemplateFiles: map[string]string{
					"testCluster.yml": "classes:\n- testCluster.common\n",
					"oldFile.yml":     "{delete}",
				},
			},
			args: args{
				data: &ExecutionContext{},
				obj: &synv1alpha1.Tenant{
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.args.data.Client, _ = testSetupClient([]runtime.Object{
				tt.args.cluster,
				tt.args.obj,
				&synv1alpha1.ClusterList{},
			})

			got := updateTenantGitRepo(tt.args.obj, tt.args.data)
			assert.NoError(t, got.Err)

			assert.Equal(t, tt.want, tt.args.obj.GetGitTemplate())

		})
	}
}
