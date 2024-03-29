package cluster

import (
	"testing"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var clusterTemplateTestCases = map[string]struct {
	cluster *synv1alpha1.Cluster
	tenant  *synv1alpha1.Tenant
	out     *synv1alpha1.Cluster
}{
	"noop": {
		cluster: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName: "Foo",
				GitRepoURL:  "git.foo.example.com",
			},
		},
		tenant: &synv1alpha1.Tenant{
			Spec: synv1alpha1.TenantSpec{
				ClusterTemplate: &synv1alpha1.ClusterSpec{},
			},
		},
		out: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName: "Foo",
				GitRepoURL:  "git.foo.example.com",
			},
		},
	},
	"simple": {
		cluster: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName: "Foo",
				GitRepoURL:  "git.foo.example.com",
			},
		},
		tenant: &synv1alpha1.Tenant{
			Spec: synv1alpha1.TenantSpec{
				ClusterTemplate: &synv1alpha1.ClusterSpec{
					DisplayName:    "BLUB",
					DeletionPolicy: "Delete",
				},
			},
		},
		out: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName:    "Foo",
				GitRepoURL:     "git.foo.example.com",
				DeletionPolicy: "Delete",
			},
		},
	},
	"template": {
		cluster: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName: "Foo",
				GitRepoURL:  "git.foo.example.com",
			},
		},
		tenant: &synv1alpha1.Tenant{
			Spec: synv1alpha1.TenantSpec{
				ClusterTemplate: &synv1alpha1.ClusterSpec{
					DisplayName:    "BLUB",
					DeletionPolicy: "Delete",
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoName:    "{{ .Name }}",
						DisplayName: "{{ .Spec.DisplayName }}",
					},
				},
			},
		},
		out: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName:    "Foo",
				GitRepoURL:     "git.foo.example.com",
				DeletionPolicy: "Delete",
				GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
					RepoName:    "foo",
					DisplayName: "Foo",
				},
			},
		},
	},
	"template accessing tenant": {
		cluster: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName: "Foo",
				GitRepoURL:  "git.foo.example.com",
			},
		},
		tenant: &synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: "t-Foo",
			},
			Spec: synv1alpha1.TenantSpec{
				ClusterTemplate: &synv1alpha1.ClusterSpec{
					DisplayName:    "BLUB",
					DeletionPolicy: "Delete",
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoName:    "{{ .Name }}",
						DisplayName: "{{ .Spec.DisplayName }} of {{ .Tenant.Name }}",
					},
				},
			},
		},
		out: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName:    "Foo",
				GitRepoURL:     "git.foo.example.com",
				DeletionPolicy: "Delete",
				GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
					RepoName:    "foo",
					DisplayName: "Foo of t-Foo",
				},
			},
		},
	},
	"template accessing tenant annotation": {
		cluster: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				GitRepoURL: "git.foo.example.com",
			},
		},
		tenant: &synv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: "t-Foo",
				Annotations: map[string]string{
					"confidential": "true",
				},
			},
			Spec: synv1alpha1.TenantSpec{
				ClusterTemplate: &synv1alpha1.ClusterSpec{
					DisplayName:    "{{if eq .Tenant.Annotations.confidential  \"true\" }}CONFIDENTIAL{{ else }}Foo{{ end }}",
					DeletionPolicy: "Delete",
					GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
						RepoName: "{{ .Name }}",
					},
				},
			},
		},
		out: &synv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: synv1alpha1.ClusterSpec{
				DisplayName:    "CONFIDENTIAL",
				GitRepoURL:     "git.foo.example.com",
				DeletionPolicy: "Delete",
				GitRepoTemplate: &synv1alpha1.GitRepoTemplate{
					RepoName: "foo",
				},
			},
		},
	},
}

func TestApplyClusterTemplate(t *testing.T) {

	for key, tc := range clusterTemplateTestCases {
		t.Run(key, func(t *testing.T) {
			err := applyClusterTemplate(tc.cluster, tc.tenant)
			require.NoError(t, err)
			assert.Equal(t, tc.cluster, tc.out)
		})
	}
}
