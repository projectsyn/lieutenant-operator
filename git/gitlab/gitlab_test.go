package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"
	"go.uber.org/zap"

	"github.com/projectsyn/lieutenant-operator/git/manager"
	"github.com/projectsyn/lieutenant-operator/testutils"
)

func testGetHTTPServer(statusCode int, body []byte) *httptest.Server {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(statusCode)
		_, _ = res.Write(body)
	}))

	return testServer
}

//goland:noinspection HttpUrlsUsage
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

			serverURL, _ := url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				ops: manager.RepoOptions{
					URL: serverURL,
				},
				credentials: tt.fields.credentials,
			}

			err := g.Connect()
			require.NoError(t, err)

			if err := g.Read(); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//goland:noinspection HttpUrlsUsage
func testGetCreateServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/projects/3/deploy_keys", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`[{"id":1,"title":"Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T10:12:29Z","can_push":false},{"id":3,"title":"Another Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T11:12:29Z","can_push":false}]`))
	})

	mux.HandleFunc("/api/v4/projects", func(res http.ResponseWriter, req *http.Request) {
		createProjectOptions := gitlab.CreateProjectOptions{}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(req.Body)
		err := json.Unmarshal(buf.Bytes(), &createProjectOptions)
		response := http.StatusOK
		if err != nil {
			response = http.StatusInternalServerError
		}
		res.WriteHeader(response)
		_, _ = res.Write([]byte(`{"id":3,"description":"` + *createProjectOptions.Description + `","default_branch":"master","visibility":"private","ssh_url_to_repo":"git@example.com:diaspora/diaspora-project-site.git","http_url_to_repo":"http://example.com/diaspora/diaspora-project-site.git","web_url":"http://example.com/diaspora/diaspora-project-site","readme_url":"http://example.com/diaspora/diaspora-project-site/blob/master/README.md","tag_list":["example","disapora project"],"owner":{"id":3,"name":"Diaspora","created_at":"2013-09-30T13:46:02Z"},"name":"` + *createProjectOptions.Name + `","name_with_namespace":"group1 / Diaspora Project Site","path":"diaspora-project-site","path_with_namespace":"group1/diaspora-project-site","issues_enabled":true,"open_issues_count":1,"merge_requests_enabled":true,"jobs_enabled":true,"wiki_enabled":true,"snippets_enabled":false,"resolve_outdated_diff_discussions":false,"container_registry_enabled":false,"container_expiration_policy":{"cadence":"7d","enabled":false,"keep_n":null,"older_than":null,"name_regex":null,"next_run_at":"2020-01-07T21:42:58.658Z"},"created_at":"2013-09-30T13:46:02Z","last_activity_at":"2013-09-30T13:46:02Z","creator_id":2,"namespace":{"id":2,"name":"group1","path":"group1","kind":"group","full_path":"group1","parent_id":null,"members_count_with_descendants":2},"import_status":"none","import_error":null,"permissions":{"project_access":{"access_level":10,"notification_level":3},"group_access":{"access_level":50,"notification_level":3}},"archived":false,"avatar_url":"http://example.com/uploads/project/avatar/3/uploads/avatar.png","license_url":"http://example.com/diaspora/diaspora-client/blob/master/LICENSE","license":{"key":"lgpl-3.0","name":"GNU Lesser General Public License v3.0","nickname":"GNU LGPLv3","html_url":"http://choosealicense.com/licenses/lgpl-3.0/","source_url":"http://www.gnu.org/licenses/lgpl-3.0.txt"},"shared_runners_enabled":true,"forks_count":0,"star_count":0,"runners_token":"b8bc4a7a29eb76ea83cf79e4908c2b","ci_default_git_depth":50,"public_jobs":true,"shared_with_groups":[{"group_id":4,"group_name":"Twitter","group_full_path":"twitter","group_access_level":30},{"group_id":3,"group_name":"Gitlab Org","group_full_path":"gitlab-org","group_access_level":10}],"repository_storage":"default","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false,"remove_source_branch_after_merge":false,"printing_merge_requests_link_enabled":true,"request_access_enabled":false,"merge_method":"merge","auto_devops_enabled":true,"auto_devops_deploy_strategy":"continuous","approvals_before_merge":0,"mirror":false,"mirror_user_id":45,"mirror_trigger_builds":false,"only_mirror_protected_branches":false,"mirror_overwrites_diverged_branches":false,"external_authorization_classification_label":null,"packages_enabled":true,"service_desk_enabled":false,"service_desk_address":null,"autoclose_referenced_issues":true,"suggestion_commit_message":null,"statistics":{"commit_count":37,"storage_size":1038090,"repository_size":1038090,"wiki_size":0,"lfs_objects_size":0,"job_artifacts_size":0,"packages_size":0},"_links":{"self":"http://example.com/api/v4/projects","issues":"http://example.com/api/v4/projects/1/issues","merge_requests":"http://example.com/api/v4/projects/1/merge_requests","repo_branches":"http://example.com/api/v4/projects/1/repository_branches","labels":"http://example.com/api/v4/projects/1/labels","events":"http://example.com/api/v4/projects/1/events","members":"http://example.com/api/v4/projects/1/members"}}`))
	})

	mux.HandleFunc("/api/v4/namespaces/group1", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`{"id":2,"name":"group1","path":"group1","kind":"group","full_path":"group1","parent_id":null,"members_count_with_descendants":2}`))
	})
	mux.HandleFunc("/api/v4/namespaces/path%2Fto%2Fgroup2", func(res http.ResponseWriter, req *http.Request) {
		if req.URL.RawPath != "/api/v4/namespaces/path%2Fto%2Fgroup2" {
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte(`{"message":"500 Request path not URL-escaped as expected"}`))
		} else {
			res.WriteHeader(http.StatusOK)
			_, _ = res.Write([]byte(`{"id":6,"name":"group2","path":"group2","kind":"group","full_path":"path/to/group2","parent_id":5,"members_count_with_descendants":2}`))
		}
	})

	mux.HandleFunc("/", testutils.LogNotFoundHandler(t))

	return httptest.NewServer(mux)
}

func TestGitlab_Create(t *testing.T) {
	type fields struct {
		credentials manager.Credentials
		namespace   string
		projectname string
		description string
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
				description: "desc",
			},
			httpServer: testGetCreateServer(t),
			wantErr:    false,
		},
		{
			name: "subgroup create successful",
			fields: fields{
				credentials: manager.Credentials{},
				namespace:   "path/to/group2",
				projectname: "test",
				description: "desc",
			},
			httpServer: testGetCreateServer(t),
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

			serverURL, _ := url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				credentials: tt.fields.credentials,
				ops: manager.RepoOptions{
					URL:         serverURL,
					Path:        tt.fields.namespace,
					RepoName:    tt.fields.projectname,
					DisplayName: tt.fields.description,
				},
			}

			err := g.Connect()
			require.NoError(t, err)

			if tt.name == "subgroup create successful" {
				fmt.Print("subgroup create successful")
			}

			if err := g.Create(); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.fields.description, g.project.Description, "Description should have been populated")
				assert.Equal(t, tt.fields.projectname, g.project.Name, "Name should have been populated")
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

			serverURL, _ := url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				ops: manager.RepoOptions{
					URL: serverURL,
				},
				credentials: tt.fields.credentials,
			}

			err := g.Connect()
			require.NoError(t, err)

			if err := g.delete(); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//goland:noinspection HttpUrlsUsage
func testGetUpdateServer(t *testing.T, fail bool) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/projects/3/deploy_keys", func(res http.ResponseWriter, req *http.Request) {

		respH := http.StatusOK
		if fail {
			respH = http.StatusInternalServerError
		}
		res.WriteHeader(respH)
		_, _ = res.Write([]byte(`[{"id":1,"title":"Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T10:12:29Z","can_push":false},{"id":3,"title":"Another Public key","key":"ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=","created_at":"2013-10-02T11:12:29Z","can_push":false}]`))
	})

	mux.HandleFunc("/api/v4/projects/updated%2Frepo", func(res http.ResponseWriter, req *http.Request) {
		respH := http.StatusOK
		res.WriteHeader(respH)
		_, _ = res.Write([]byte(`{"id":3,"description":"oldDesc","default_branch":"master","visibility":"private","ssh_url_to_repo":"git@example.com:luzifern/luzifern-project-site.git","http_url_to_repo":"http://example.com/diaspora/diaspora-project-site.git","web_url":"http://example.com/diaspora/diaspora-project-site","readme_url":"http://example.com/diaspora/diaspora-project-site/blob/master/README.md","tag_list":["example","disapora project"],"owner":{"id":3,"name":"Diaspora","created_at":"2013-09-30T13:46:02Z"},"name":"repo","name_with_namespace":"group1 / Diaspora Project Site","path":"diaspora-project-site","path_with_namespace":"group1/diaspora-project-site","issues_enabled":true,"open_issues_count":1,"merge_requests_enabled":true,"jobs_enabled":true,"wiki_enabled":true,"snippets_enabled":false,"resolve_outdated_diff_discussions":false,"container_registry_enabled":false,"container_expiration_policy":{"cadence":"7d","enabled":false,"keep_n":null,"older_than":null,"name_regex":null,"next_run_at":"2020-01-07T21:42:58.658Z"},"created_at":"2013-09-30T13:46:02Z","last_activity_at":"2013-09-30T13:46:02Z","creator_id":2,"namespace":{"id":2,"name":"group1","path":"group1","kind":"group","full_path":"group1","parent_id":null,"members_count_with_descendants":2},"import_status":"none","import_error":null,"permissions":{"project_access":{"access_level":10,"notification_level":3},"group_access":{"access_level":50,"notification_level":3}},"archived":false,"avatar_url":"http://example.com/uploads/project/avatar/3/uploads/avatar.png","license_url":"http://example.com/diaspora/diaspora-client/blob/master/LICENSE","license":{"key":"lgpl-3.0","name":"GNU Lesser General Public License v3.0","nickname":"GNU LGPLv3","html_url":"http://choosealicense.com/licenses/lgpl-3.0/","source_url":"http://www.gnu.org/licenses/lgpl-3.0.txt"},"shared_runners_enabled":true,"forks_count":0,"star_count":0,"runners_token":"b8bc4a7a29eb76ea83cf79e4908c2b","ci_default_git_depth":50,"public_jobs":true,"shared_with_groups":[{"group_id":4,"group_name":"Twitter","group_full_path":"twitter","group_access_level":30},{"group_id":3,"group_name":"Gitlab Org","group_full_path":"gitlab-org","group_access_level":10}],"repository_storage":"default","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false,"remove_source_branch_after_merge":false,"printing_merge_requests_link_enabled":true,"request_access_enabled":false,"merge_method":"merge","auto_devops_enabled":true,"auto_devops_deploy_strategy":"continuous","approvals_before_merge":0,"mirror":false,"mirror_user_id":45,"mirror_trigger_builds":false,"only_mirror_protected_branches":false,"mirror_overwrites_diverged_branches":false,"external_authorization_classification_label":null,"packages_enabled":true,"service_desk_enabled":false,"service_desk_address":null,"autoclose_referenced_issues":true,"suggestion_commit_message":null,"statistics":{"commit_count":37,"storage_size":1038090,"repository_size":1038090,"wiki_size":0,"lfs_objects_size":0,"job_artifacts_size":0,"packages_size":0},"_links":{"self":"http://example.com/api/v4/projects","issues":"http://example.com/api/v4/projects/1/issues","merge_requests":"http://example.com/api/v4/projects/1/merge_requests","repo_branches":"http://example.com/api/v4/projects/1/repository_branches","labels":"http://example.com/api/v4/projects/1/labels","events":"http://example.com/api/v4/projects/1/events","members":"http://example.com/api/v4/projects/1/members"}}`))
	})

	mux.HandleFunc("/api/v4/projects/3", func(res http.ResponseWriter, req *http.Request) {
		editProjectOptions := gitlab.EditProjectOptions{}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(req.Body)
		err := json.Unmarshal(buf.Bytes(), &editProjectOptions)
		response := http.StatusOK
		if err != nil {
			response = http.StatusInternalServerError
		}
		res.WriteHeader(response)
		_, _ = res.Write([]byte(`{"id":3,"description":"` + *editProjectOptions.Description + `","default_branch":"master","visibility":"private","ssh_url_to_repo":"git@example.com:luzifern/luzifern-project-site.git","http_url_to_repo":"http://example.com/diaspora/diaspora-project-site.git","web_url":"http://example.com/diaspora/diaspora-project-site","readme_url":"http://example.com/diaspora/diaspora-project-site/blob/master/README.md","tag_list":["example","disapora project"],"owner":{"id":3,"name":"Diaspora","created_at":"2013-09-30T13:46:02Z"},"name":"repo","name_with_namespace":"group1 / Diaspora Project Site","path":"diaspora-project-site","path_with_namespace":"group1/diaspora-project-site","issues_enabled":true,"open_issues_count":1,"merge_requests_enabled":true,"jobs_enabled":true,"wiki_enabled":true,"snippets_enabled":false,"resolve_outdated_diff_discussions":false,"container_registry_enabled":false,"container_expiration_policy":{"cadence":"7d","enabled":false,"keep_n":null,"older_than":null,"name_regex":null,"next_run_at":"2020-01-07T21:42:58.658Z"},"created_at":"2013-09-30T13:46:02Z","last_activity_at":"2013-09-30T13:46:02Z","creator_id":2,"namespace":{"id":2,"name":"group1","path":"group1","kind":"group","full_path":"group1","parent_id":null,"members_count_with_descendants":2},"import_status":"none","import_error":null,"permissions":{"project_access":{"access_level":10,"notification_level":3},"group_access":{"access_level":50,"notification_level":3}},"archived":false,"avatar_url":"http://example.com/uploads/project/avatar/3/uploads/avatar.png","license_url":"http://example.com/diaspora/diaspora-client/blob/master/LICENSE","license":{"key":"lgpl-3.0","name":"GNU Lesser General Public License v3.0","nickname":"GNU LGPLv3","html_url":"http://choosealicense.com/licenses/lgpl-3.0/","source_url":"http://www.gnu.org/licenses/lgpl-3.0.txt"},"shared_runners_enabled":true,"forks_count":0,"star_count":0,"runners_token":"b8bc4a7a29eb76ea83cf79e4908c2b","ci_default_git_depth":50,"public_jobs":true,"shared_with_groups":[{"group_id":4,"group_name":"Twitter","group_full_path":"twitter","group_access_level":30},{"group_id":3,"group_name":"Gitlab Org","group_full_path":"gitlab-org","group_access_level":10}],"repository_storage":"default","only_allow_merge_if_pipeline_succeeds":false,"only_allow_merge_if_all_discussions_are_resolved":false,"remove_source_branch_after_merge":false,"printing_merge_requests_link_enabled":true,"request_access_enabled":false,"merge_method":"merge","auto_devops_enabled":true,"auto_devops_deploy_strategy":"continuous","approvals_before_merge":0,"mirror":false,"mirror_user_id":45,"mirror_trigger_builds":false,"only_mirror_protected_branches":false,"mirror_overwrites_diverged_branches":false,"external_authorization_classification_label":null,"packages_enabled":true,"service_desk_enabled":false,"service_desk_address":null,"autoclose_referenced_issues":true,"suggestion_commit_message":null,"statistics":{"commit_count":37,"storage_size":1038090,"repository_size":1038090,"wiki_size":0,"lfs_objects_size":0,"job_artifacts_size":0,"packages_size":0},"_links":{"self":"http://example.com/api/v4/projects","issues":"http://example.com/api/v4/projects/1/issues","merge_requests":"http://example.com/api/v4/projects/1/merge_requests","repo_branches":"http://example.com/api/v4/projects/1/repository_branches","labels":"http://example.com/api/v4/projects/1/labels","events":"http://example.com/api/v4/projects/1/events","members":"http://example.com/api/v4/projects/1/members"}}`))
	})

	deleteOk := func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`{"message":"202 Accepted"}`))
	}

	mux.HandleFunc("/api/v4/projects/3/deploy_keys/1", deleteOk)

	mux.HandleFunc("/api/v4/projects/3/deploy_keys/3", deleteOk)

	mux.HandleFunc("/", testutils.LogNotFoundHandler(t))

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
					ID:          3,
					Path:        "updated",
					Name:        "repo",
					Description: "newDesc",
				},
			},
			wantErr:    false,
			httpServer: testGetUpdateServer(t, false),
		},
		{
			name: "update failed",
			fields: fields{
				project: &gitlab.Project{
					ID:          1,
					Path:        "updated",
					Name:        "repo",
					Description: "newDesc",
				},
			},
			wantErr:    true,
			httpServer: testGetUpdateServer(t, true),
		},
	}

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			defer tt.httpServer.Close()

			serverURL, _ := url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				ops: manager.RepoOptions{
					URL:         serverURL,
					Path:        tt.fields.project.Path,
					RepoName:    tt.fields.project.Name,
					DisplayName: tt.fields.project.Description,
				},
				project: tt.fields.project,
				log:     zapr.NewLogger(zapLog),
			}

			err := g.Connect()
			require.NoError(t, err)

			if _, err := g.Update(); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				assert.Equal(t, tt.fields.project.Description, g.project.Description, "Description should have been updated")
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

//goland:noinspection HttpUrlsUsage
func testGetCommitServer(t *testing.T, files []string) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/projects/3/repository/tree", func(res http.ResponseWriter, req *http.Request) {
		if len(files) == 0 {
			_, _ = res.Write([]byte(`[]`))
			return
		}

		items := []string{}
		for _, f := range files {
			items = append(items, fmt.Sprintf(`{"id":"a1e8f8d745cc87e3a9248358d9352bb7f9a0aeba","name":"dir1","type":"tree","path":"%s","mode":"040000"}`, f))
		}
		page, err := strconv.Atoi(req.URL.Query().Get("page"))
		if err != nil && req.URL.Query().Get("page") != "" {
			res.WriteHeader(http.StatusBadRequest)
			_, _ = res.Write([]byte(`{"error":"page NaN"}`))
			return
		}
		if req.URL.Query().Get("page") == "" {
			page = 1
		}

		res.Header().Add("x-page", strconv.Itoa(page))
		if page >= len(items) {
			res.Header().Add("x-next-page", "")
		} else {
			res.Header().Add("x-next-page", strconv.Itoa(page+1))
		}
		res.Header().Add("x-per-page", "1")
		res.Header().Add("x-total", strconv.Itoa(len(items)))
		res.Header().Add("x-total-pages", strconv.Itoa(len(items)))

		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte("[" + items[page-1] + "]"))
	})

	mux.HandleFunc("/api/v4/projects/3/repository/commits", func(res http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			_, _ = res.Write([]byte(`{"error":"body empty"}`))
			return
		}
		commit := struct {
			Actions []struct {
				Action   string
				Content  string
				FilePath string `json:"file_path"`
			}
		}{}
		err = json.Unmarshal(body, &commit)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			_, _ = res.Write([]byte(`{"error":"body not json"}`))
			return
		}
		touchedFiles := map[string]struct{}{}
		for _, a := range commit.Actions {
			if _, ok := touchedFiles[a.FilePath]; ok {
				res.WriteHeader(http.StatusBadRequest)
				_, _ = res.Write([]byte(`{"error":"file created twice"}`))
				return
			}
			touchedFiles[a.FilePath] = struct{}{}
			if a.Content == manager.DeletionMagicString && gitlab.FileActionValue(a.Action) != gitlab.FileDelete {
				res.WriteHeader(http.StatusBadRequest)
				_, _ = res.Write([]byte(`{"error":"creating a file containing { deleted } instead of deleting"}`))
				return
			}
		}
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte(`{"id":"ed899a2f4b50b4370feeea94676502b42383c746","short_id":"ed899a2f4b5","title":"some commit message","author_name":"Example User","author_email":"user@example.com","committer_name":"Example User","committer_email":"user@example.com","created_at":"2016-09-20T09:26:24.000-07:00","message":"some commit message","parent_ids":["ae1d9fb46aa2b07ee9836d49862ec4e2c46fbbba"],"committed_date":"2016-09-20T09:26:24.000-07:00","authored_date":"2016-09-20T09:26:24.000-07:00","stats":{"additions":2,"deletions":2,"total":4},"status":null,"web_url":"https://localhost:8080/thedude/gitlab-foss/-/commit/ed899a2f4b50b4370feeea94676502b42383c746"}`))
	})

	mux.HandleFunc("/", testutils.LogNotFoundHandler(t))

	return httptest.NewServer(mux)
}

func TestGitlab_CommitTemplateFiles(t *testing.T) {
	type fields struct {
		project *gitlab.Project
		ops     manager.RepoOptions
	}
	tests := map[string]struct {
		fields     fields
		wantErr    bool
		httpServer *httptest.Server
	}{
		"set template files": {
			wantErr:    false,
			httpServer: testGetCommitServer(t, []string{"file1"}),
			fields: fields{
				project: &gitlab.Project{
					ID: 3,
				},
				ops: manager.RepoOptions{
					TemplateFiles: map[string]string{
						"test": "testContent",
					},
				},
			},
		},
		"set existing file": {
			wantErr:    false,
			httpServer: testGetCommitServer(t, []string{"file1"}),
			fields: fields{
				project: &gitlab.Project{
					ID: 3,
				},
				ops: manager.RepoOptions{
					TemplateFiles: map[string]string{
						"file1": "testContent",
					},
				},
			},
		},
		"set multiple template files": {
			wantErr:    false,
			httpServer: testGetCommitServer(t, []string{"file1"}),
			fields: fields{
				project: &gitlab.Project{
					ID: 3,
				},
				ops: manager.RepoOptions{
					TemplateFiles: map[string]string{
						"test1": "testContent",
						"test2": "testContent",
						"test3": "testContent",
					},
				},
			},
		},
		"delete  file": {
			wantErr:    false,
			httpServer: testGetCommitServer(t, []string{"file1"}),
			fields: fields{
				project: &gitlab.Project{
					ID: 3,
				},
				ops: manager.RepoOptions{
					TemplateFiles: map[string]string{
						"file1": manager.DeletionMagicString,
					},
				},
			},
		},
		"set and delete file while empty": {
			wantErr:    false,
			httpServer: testGetCommitServer(t, []string{}),
			fields: fields{
				project: &gitlab.Project{
					ID: 3,
				},
				ops: manager.RepoOptions{
					TemplateFiles: map[string]string{
						"test1": "testContent",
						"test2": manager.DeletionMagicString,
						"test3": "testContent",
					},
				},
			},
		},
	}
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}

	ListItemsPerPage = 1 // simulate pagination
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			tt.fields.ops.URL, _ = url.Parse(tt.httpServer.URL)

			g := &Gitlab{
				project: tt.fields.project,
				log:     zapr.NewLogger(zapLog),
				ops:     tt.fields.ops,
			}

			err := g.Connect()
			require.NoError(t, err)

			err = g.CommitTemplateFiles()
			if tt.wantErr {
				assert.Errorf(t, err, "Gitlab.CommitTemplateFiles() error = %v", err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestGitlab_FullURL(t *testing.T) {
	serverURL, err := url.Parse("git.example.com/foo/bar")
	require.NoError(t, err)
	expectedFullURL := "ssh://git@git.example.com/foo/bar.git"
	g := &Gitlab{
		ops: manager.RepoOptions{
			URL: serverURL,
		},
	}
	assert.Equal(t, expectedFullURL, g.FullURL().String())
	assert.Equal(t, expectedFullURL, g.FullURL().String())
	assert.Equal(t, expectedFullURL, g.FullURL().String())
}
