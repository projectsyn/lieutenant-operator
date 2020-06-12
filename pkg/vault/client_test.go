package vault

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/hashicorp/vault/api"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func testGetHTTPServer(statusCode int, body []byte) *httptest.Server {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(statusCode)
		_, _ = res.Write(body)
	}))

	return testServer
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		want       *VaultClient
		wantErr    bool
		statusCode int
		body       string
	}{
		{
			name:       "test create secret",
			wantErr:    false,
			statusCode: 200,
			body: `{
				"data": {
				  "data": {
					"foo": "bar"
				  },
				  "metadata": {
					"created_time": "2018-03-22T02:24:06.945319214Z",
					"deletion_time": "",
					"destroyed": false,
					"version": 2
				  }
				}
			  }`,
		},
		{
			name:       "test error",
			wantErr:    true,
			statusCode: 404,
			body:       "{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceClient = nil
			server := testGetHTTPServer(tt.statusCode, []byte(tt.body))

			os.Setenv(api.EnvVaultToken, "myroot")
			os.Setenv(api.EnvVaultAddress, server.URL)

			defer server.Close()

			_, err := NewClient(synv1alpha1.RetainPolicy, zap.Logger())
			if err != nil {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

func TestBankVaultClient_AddSecrets(t *testing.T) {
	type args struct {
		secrets []VaultSecret
		token   string
		log     logr.Logger
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		statusCode int
		body       string
	}{
		{
			name: "test SetToken",
			args: args{
				secrets: []VaultSecret{{Path: "1234/6789", Value: ""}},
				token:   "test",
				log:     zap.Logger(),
			},
			body: `{
				"data": {
				  "data": {
					"foo": "bar"
				  },
				  "metadata": {
					"created_time": "2018-03-22T02:24:06.945319214Z",
					"deletion_time": "",
					"destroyed": false,
					"version": 2
				  }
				}
			  }`,
			statusCode: 200,
		},
		{
			name:       "test error",
			wantErr:    true,
			statusCode: 404,
			body:       "{}",
			args: args{
				secrets: []VaultSecret{{Path: "1234/6789", Value: ""}},
				token:   "test",
				log:     zap.Logger(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceClient = nil
			server := testGetHTTPServer(tt.statusCode, []byte(tt.body))

			os.Setenv(api.EnvVaultToken, "myroot")
			os.Setenv(api.EnvVaultAddress, server.URL)

			defer server.Close()

			b, _ := NewClient(synv1alpha1.ArchivePolicy, tt.args.log)
			if err := b.AddSecrets(tt.args.secrets); (err != nil) != tt.wantErr {
				t.Errorf("BankVaultClient.SetToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBankVaultClient_RemoveSecrets(t *testing.T) {
	type args struct {
		secrets []VaultSecret
		policy  synv1alpha1.DeletionPolicy
		log     logr.Logger
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "deleting",
			wantErr: false,
			args: args{
				secrets: []VaultSecret{{Path: "kv2/test", Value: ""}},
				policy:  synv1alpha1.DeletePolicy,
				log:     zap.Logger(),
			},
		},
		{
			name:    "archiving",
			wantErr: false,
			args: args{
				secrets: []VaultSecret{{Path: "kv2/test", Value: ""}},
				policy:  synv1alpha1.ArchivePolicy,
				log:     zap.Logger(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceClient = nil
			server := getVersionHTTPServer()

			os.Setenv(api.EnvVaultToken, "myroot")
			os.Setenv(api.EnvVaultAddress, server.URL)

			defer server.Close()

			b, _ := NewClient(tt.args.policy, tt.args.log)
			if err := b.RemoveSecrets(tt.args.secrets); (err != nil) != tt.wantErr {
				t.Errorf("BankVaultClient.SetToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getVersionHTTPServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/kv/delete/kv2/test/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1/kv/metadata/kv2/test/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		versionBody := `
		{
			"data": {
			  "created_time": "2018-03-22T02:24:06.945319214Z",
			  "current_version": 3,
			  "max_versions": 0,
			  "oldest_version": 0,
			  "updated_time": "2018-03-22T02:36:43.986212308Z",
			  "versions": {
				"1": {
				  "created_time": "2018-03-22T02:24:06.945319214Z",
				  "deletion_time": "",
				  "destroyed": false
				},
				"2": {
				  "created_time": "2018-03-22T02:36:33.954880664Z",
				  "deletion_time": "",
				  "destroyed": false
				},
				"3": {
				  "created_time": "2018-03-22T02:36:43.986212308Z",
				  "deletion_time": "",
				  "destroyed": false
				}
			  }
			}
		  }`
		_, _ = io.WriteString(w, versionBody)
	})

	mux.HandleFunc("/v1/kv/metadata/kv2/test", func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)

		if r.URL.Query().Get("list") == "true" {
			_, _ = io.WriteString(w, `{
				"data": {
				  "keys": ["foo", "foo/"]
				}
			  }`)
			return
		}

	})

	return httptest.NewServer(mux)
}

func TestBankVaultClient_getVersionList(t *testing.T) {
	type args struct {
		data map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "test parsing",
			args: args{
				data: map[string]interface{}{
					"versions": map[string]interface{}{
						"1": struct{}{},
						"2": struct{}{},
					},
				},
			},
			wantErr: false,
			want: map[string]interface{}{
				"versions": []int{1, 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BankVaultClient{}
			got, err := b.getVersionList(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("BankVaultClient.getVersionList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BankVaultClient.getVersionList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getListHTTPServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/kv/metadata/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{
			"data": {
			  "keys": ["foo", "foo/"]
			}
		  }`)
	})

	return httptest.NewServer(mux)
}

func TestBankVaultClient_listSecrets(t *testing.T) {
	type args struct {
		secretPath string
		policy     synv1alpha1.DeletionPolicy
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "parse test",
			wantErr: false,
			want:    []string{"foo", "foo/"},
			args: args{
				secretPath: "test",
				policy:     synv1alpha1.DeletePolicy,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			instanceClient = nil
			server := getListHTTPServer()

			os.Setenv(api.EnvVaultToken, "myroot")
			os.Setenv(api.EnvVaultAddress, server.URL)

			defer server.Close()

			b, err := newBankVaultClient(tt.args.policy, zap.Logger())
			assert.NoError(t, err)

			got, err := b.listSecrets(tt.args.secretPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("BankVaultClient.listSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BankVaultClient.listSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}
