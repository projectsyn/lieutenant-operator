package gitlab

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"

	"github.com/xanzy/go-gitlab"
)

func testGetHTTPServer(statusCode int, body []byte) *httptest.Server {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(statusCode)
		_, _ = res.Write(body)
	}))

	return testServer
}

func TestGitlab_Read(t *testing.T) {
	type fields struct {
		credentials manager.Credentials
	}
	tests := []struct {
		name       string
		fields     fields
		httpServer *httptest.Server
		wantErr    bool
	}{
		{
			name: "test read ok",
			fields: fields{
				credentials: manager.Credentials{},
			},
			wantErr:    false,
			httpServer: testGetHTTPServer(http.StatusOK, []byte(`{"id":3,"description":null,"default_branch":"master","visibility":"private","ssh_url_to_repo":"git@example.com:diaspora/diaspora-project-site.git","http_url_to_repo":"http://example.com/diaspora/diaspora-project-site.git","web_url":"http://example.com/diaspora/diaspora-project-site","readme_url":"http://example.com/diaspora/diaspora-project-site/blob/master/README.md","tag_list":["example","disapora project"],"owner":{"id":3,"name":"Diaspora","created_at":"2013-09-30T13:46:02Z"},"name":"Diaspora Project Site","name_with_namespace":"Diaspora / Diaspora Project Site","path":"diaspora-project-site","path_with_namespace":"diaspora/diaspora-project-site","issues_enabled":true,"open_issues_count":1,"merge_requests_enabled":true,"jobs_enabled":true,"wiki_enabled":true,"snippets_enabled":false,"resolve_outdated_diff_discussions":false,"container_registry_enabled":false,"container_expiration_policy":{"cadence":"7d","enabled":false,"keep_n":null,"older_than":null,"name_regex":null,"next_run_at":"2020-01-07T21:42:58.658Z"},"created_at":"2013-09-30T13:46:02Z","last_activity_at":"2013-09-30T13:46:02Z","creator_id":3,"namespace":{"id":3,"name":"Diaspora","path":"diaspora","kind":"group","full_path":"diaspora","avatar_url":"http://localhost:3000/uploads/group/avatar/3/foo.jpg","web_url":"http://localhost:3000/groups/diaspora"},"import_status":"none","import_error":null,"permissions":{"project_access":{"access_level":10,"notification_level":3},"group_access":{"access_level":50,"notification_level":3}},"archived":false,"avatar_url":"http://example.com/uploads/project/avatar/3/uploads/avatar.png","license_url":"http://example.com/diaspora/diaspora-client/blob/master/LICENSE","license":{"key":"lgpl-3.0","name":"GNU Lesser General Public License v3.0","nickname":"GNU LGPLv3","html_url":"http://choosealicense.com/licenses/lgpl-3.0/","source_url":"http://www.gnu.org/licenses/lgpl-3.0.txt"},"shared_runners_enabled":true,"forks_count":0,"star_count":0,"runners_token":"b8bc4a7a29eb76ea83cf79e4908c2b","ci_default_git_depth":50,"public_jobs":true,"shared_with_groups":[{"group_id":4,"group_name":"Twitter","group_full_path":"twitter","group_access_level":30},{"group_id":3,"group_name":"Gitlab Org","group_full_path":"gitlab-org","group_access_level":10}],"repository_storage":"default","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false,"remove_source_branch_after_merge":false,"printing_merge_requests_link_enabled":true,"request_access_enabled":false,"merge_method":"merge","auto_devops_enabled":true,"auto_devops_deploy_strategy":"continuous","approvals_before_merge":0,"mirror":false,"mirror_user_id":45,"mirror_trigger_builds":false,"only_mirror_protected_branches":false,"mirror_overwrites_diverged_branches":false,"external_authorization_classification_label":null,"packages_enabled":true,"service_desk_enabled":false,"service_desk_address":null,"autoclose_referenced_issues":true,"suggestion_commit_message":null,"statistics":{"commit_count":37,"storage_size":1038090,"repository_size":1038090,"wiki_size":0,"lfs_objects_size":0,"job_artifacts_size":0,"packages_size":0},"_links":{"self":"http://example.com/api/v4/projects","issues":"http://example.com/api/v4/projects/1/issues","merge_requests":"http://example.com/api/v4/projects/1/merge_requests","repo_branches":"http://example.com/api/v4/projects/1/repository_branches","labels":"http://example.com/api/v4/projects/1/labels","events":"http://example.com/api/v4/projects/1/events","members":"http://example.com/api/v4/projects/1/members"}}`)),
		},
		{
			name: "test not existing",
			fields: fields{
				credentials: manager.Credentials{},
			},
			wantErr:    true,
			httpServer: testGetHTTPServer(http.StatusNotFound, []byte(`{"error":"not found"}`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			defer tt.httpServer.Close()

			parsedURL, _ := url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				url:         parsedURL,
				credentials: tt.fields.credentials,
			}

			_ = g.Connect()

			if err := g.Read(); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitlab_IsType(t *testing.T) {
	tests := []struct {
		name       string
		want       bool
		httpServer *httptest.Server
	}{
		{
			name:       "is gitlab",
			want:       true,
			httpServer: testGetHTTPServer(http.StatusOK, []byte("")),
		},
		{
			name:       "is not Gitlab",
			want:       false,
			httpServer: testGetHTTPServer(http.StatusNotFound, []byte("")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer tt.httpServer.Close()
			g := &Gitlab{}
			if got, _ := g.IsType(tt.httpServer.URL); got != tt.want {
				t.Errorf("IsType() = %v, want %v", got, tt.want)
			}
		})
	}

}

func testGetCreateServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/projects/3/deploy_keys", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`[{"id":1,"title":"Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T10:12:29Z","can_push":false},{"id":3,"title":"Another Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T11:12:29Z","can_push":false}]`))
	})

	mux.HandleFunc("/api/v4/projects", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`{"id":3,"description":null,"default_branch":"master","visibility":"private","ssh_url_to_repo":"git@example.com:diaspora/diaspora-project-site.git","http_url_to_repo":"http://example.com/diaspora/diaspora-project-site.git","web_url":"http://example.com/diaspora/diaspora-project-site","readme_url":"http://example.com/diaspora/diaspora-project-site/blob/master/README.md","tag_list":["example","disapora project"],"owner":{"id":3,"name":"Diaspora","created_at":"2013-09-30T13:46:02Z"},"name":"Diaspora Project Site","name_with_namespace":"group1 / Diaspora Project Site","path":"diaspora-project-site","path_with_namespace":"group1/diaspora-project-site","issues_enabled":true,"open_issues_count":1,"merge_requests_enabled":true,"jobs_enabled":true,"wiki_enabled":true,"snippets_enabled":false,"resolve_outdated_diff_discussions":false,"container_registry_enabled":false,"container_expiration_policy":{"cadence":"7d","enabled":false,"keep_n":null,"older_than":null,"name_regex":null,"next_run_at":"2020-01-07T21:42:58.658Z"},"created_at":"2013-09-30T13:46:02Z","last_activity_at":"2013-09-30T13:46:02Z","creator_id":2,"namespace":{"id":2,"name":"group1","path":"group1","kind":"group","full_path":"group1","parent_id":null,"members_count_with_descendants":2},"import_status":"none","import_error":null,"permissions":{"project_access":{"access_level":10,"notification_level":3},"group_access":{"access_level":50,"notification_level":3}},"archived":false,"avatar_url":"http://example.com/uploads/project/avatar/3/uploads/avatar.png","license_url":"http://example.com/diaspora/diaspora-client/blob/master/LICENSE","license":{"key":"lgpl-3.0","name":"GNU Lesser General Public License v3.0","nickname":"GNU LGPLv3","html_url":"http://choosealicense.com/licenses/lgpl-3.0/","source_url":"http://www.gnu.org/licenses/lgpl-3.0.txt"},"shared_runners_enabled":true,"forks_count":0,"star_count":0,"runners_token":"b8bc4a7a29eb76ea83cf79e4908c2b","ci_default_git_depth":50,"public_jobs":true,"shared_with_groups":[{"group_id":4,"group_name":"Twitter","group_full_path":"twitter","group_access_level":30},{"group_id":3,"group_name":"Gitlab Org","group_full_path":"gitlab-org","group_access_level":10}],"repository_storage":"default","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false,"remove_source_branch_after_merge":false,"printing_merge_requests_link_enabled":true,"request_access_enabled":false,"merge_method":"merge","auto_devops_enabled":true,"auto_devops_deploy_strategy":"continuous","approvals_before_merge":0,"mirror":false,"mirror_user_id":45,"mirror_trigger_builds":false,"only_mirror_protected_branches":false,"mirror_overwrites_diverged_branches":false,"external_authorization_classification_label":null,"packages_enabled":true,"service_desk_enabled":false,"service_desk_address":null,"autoclose_referenced_issues":true,"suggestion_commit_message":null,"statistics":{"commit_count":37,"storage_size":1038090,"repository_size":1038090,"wiki_size":0,"lfs_objects_size":0,"job_artifacts_size":0,"packages_size":0},"_links":{"self":"http://example.com/api/v4/projects","issues":"http://example.com/api/v4/projects/1/issues","merge_requests":"http://example.com/api/v4/projects/1/merge_requests","repo_branches":"http://example.com/api/v4/projects/1/repository_branches","labels":"http://example.com/api/v4/projects/1/labels","events":"http://example.com/api/v4/projects/1/events","members":"http://example.com/api/v4/projects/1/members"}}`))
	})

	mux.HandleFunc("/api/v4/namespaces/group1", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`{"id":2,"name":"group1","path":"group1","kind":"group","full_path":"group1","parent_id":null,"members_count_with_descendants":2}`))
	})

	return httptest.NewServer(mux)

}

func TestGitlab_Create(t *testing.T) {
	type fields struct {
		credentials manager.Credentials
		namespace   string
		projectname string
	}
	tests := []struct {
		name       string
		fields     fields
		httpServer *httptest.Server
		wantErr    bool
	}{
		{
			name: "create successful",
			fields: fields{
				credentials: manager.Credentials{},
				namespace:   "group1",
				projectname: "test",
			},
			httpServer: testGetCreateServer(),
			wantErr:    false,
		},
		{
			name: "create not successful",
			fields: fields{
				credentials: manager.Credentials{},
			},
			wantErr:    true,
			httpServer: testGetHTTPServer(http.StatusInternalServerError, []byte("")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			defer tt.httpServer.Close()

			parsedURL, _ := url.Parse(tt.httpServer.URL + "/" + tt.fields.namespace + "/" + tt.fields.projectname)

			g := &Gitlab{
				credentials: tt.fields.credentials,
				url:         parsedURL,
			}

			_ = g.Connect()

			if err := g.Create(); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitlab_Delete(t *testing.T) {
	type fields struct {
		credentials manager.Credentials
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		httpServer *httptest.Server
	}{
		{
			name:       "delete successful",
			fields:     fields{credentials: manager.Credentials{}},
			wantErr:    false,
			httpServer: testGetHTTPServer(http.StatusOK, []byte(`{"message":"202 Accepted"}`)),
		},
		{
			name:       "delete unsuccessful",
			fields:     fields{credentials: manager.Credentials{}},
			wantErr:    true,
			httpServer: testGetHTTPServer(http.StatusInternalServerError, []byte(``)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			defer tt.httpServer.Close()

			parsedURL, _ := url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				url:         parsedURL,
				credentials: tt.fields.credentials,
			}

			_ = g.Connect()

			if err := g.Delete(); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testGetUpdateServer(fail bool) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/projects/3/deploy_keys", func(res http.ResponseWriter, req *http.Request) {

		respH := http.StatusOK
		if fail {
			respH = http.StatusInternalServerError
		}
		res.WriteHeader(respH)
		_, _ = res.Write([]byte(`[{"id":1,"title":"Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T10:12:29Z","can_push":false},{"id":3,"title":"Another Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T11:12:29Z","can_push":false}]`))
	})

	deleteOk := func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`{"message":"202 Accepted"}`))
	}

	mux.HandleFunc("/api/v4/projects/3/deploy_keys/1", deleteOk)

	mux.HandleFunc("/api/v4/projects/3/deploy_keys/3", deleteOk)

	return httptest.NewServer(mux)

}

func TestGitlab_Update(t *testing.T) {
	type fields struct {
		project *gitlab.Project
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		httpServer *httptest.Server
	}{
		{
			name: "update successful",
			fields: fields{
				project: &gitlab.Project{
					ID: 3,
				},
			},
			wantErr:    false,
			httpServer: testGetUpdateServer(false),
		},
		{
			name: "update failed",
			fields: fields{
				project: &gitlab.Project{
					ID: 1,
				},
			},
			wantErr:    true,
			httpServer: testGetUpdateServer(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			defer tt.httpServer.Close()

			parsedURL, _ := url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				url:     parsedURL,
				project: tt.fields.project,
				log:     zap.Logger(),
			}

			_ = g.Connect()

			if _, err := g.Update(); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitlab_Type(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "type gitlab",
			want: "gitlab",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Gitlab{}
			if got := g.Type(); got != tt.want {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}
