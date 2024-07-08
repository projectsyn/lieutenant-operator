package vault

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	synv1alpha1 "github.com/projectsyn/lieutenant-operator/api/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/testutils"
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

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceClient = nil
			server := testGetHTTPServer(tt.statusCode, []byte(tt.body))

			err = os.Setenv(api.EnvVaultToken, "myroot")
			require.NoError(t, err)
			err = os.Setenv(api.EnvVaultAddress, server.URL)
			require.NoError(t, err)

			defer server.Close()

			_, err := NewClient(synv1alpha1.RetainPolicy, zapr.NewLogger(zapLog))
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
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	tests := []struct {
		name       string
		mountPath  string
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
				log:     zapr.NewLogger(zapLog),
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
			name:      "test different path",
			mountPath: "clusters/kv",
			args: args{
				secrets: []VaultSecret{{Path: "some/test", Value: ""}},
				token:   "test",
				log:     zapr.NewLogger(zapLog),
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
				log:     zapr.NewLogger(zapLog),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceClient = nil
			server := testGetHTTPServer(tt.statusCode, []byte(tt.body))

			err = os.Setenv(api.EnvVaultToken, "myroot")
			require.NoError(t, err)
			err = os.Setenv(api.EnvVaultAddress, server.URL)
			require.NoError(t, err)
			err = os.Setenv("VAULT_SECRET_ENGINE_PATH", tt.mountPath)
			require.NoError(t, err)

			defer server.Close()

			b, err := NewClient(synv1alpha1.ArchivePolicy, tt.args.log)
			require.NoError(t, err)

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
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
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
				log:     zapr.NewLogger(zapLog),
			},
		},
		{
			name:    "archiving",
			wantErr: false,
			args: args{
				secrets: []VaultSecret{{Path: "kv2/test", Value: ""}},
				policy:  synv1alpha1.ArchivePolicy,
				log:     zapr.NewLogger(zapLog),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceClient = nil
			server := getVersionHTTPServer(t)

			err = os.Setenv(api.EnvVaultToken, "myroot")
			require.NoError(t, err)
			err = os.Setenv(api.EnvVaultAddress, server.URL)
			require.NoError(t, err)

			defer server.Close()

			b, err := NewClient(tt.args.policy, tt.args.log)
			require.NoError(t, err)

			if err := b.RemoveSecrets(tt.args.secrets); (err != nil) != tt.wantErr {
				t.Errorf("BankVaultClient.SetToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getVersionHTTPServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/kv/delete/kv2/test/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/kv/delete/kv2/test/bar", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"errors":[]}`)
	})
	mux.HandleFunc("/v1/kv/delete/kv2/test/bar/buzz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1/kv/metadata/kv2/test/bar", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list") == "true" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{
				"data": {
				  "keys": ["buzz"]
				}
			  }`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"errors":[]}`)
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
	mux.HandleFunc("/v1/kv/metadata/kv2/test/bar/buzz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		versionBody := `
		{
			"data": {
			  "created_time": "2018-03-22T02:24:06.945319214Z",
			  "current_version": 1,
			  "max_versions": 0,
			  "oldest_version": 0,
			  "updated_time": "2018-03-22T02:36:43.986212308Z",
			  "versions": {
				"1": {
				  "created_time": "2018-03-22T02:24:06.945319214Z",
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
				  "keys": ["foo", "bar/"]
				}
			  }`)
			return
		}

	})

	mux.HandleFunc("/", testutils.LogNotFoundHandler(t))

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

func getListHTTPServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/kv/metadata/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{
			"data": {
			  "keys": ["foo", "foo/"]
			}
		  }`)
	})

	mux.HandleFunc("/", testutils.LogNotFoundHandler(t))

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
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			instanceClient = nil
			server := getListHTTPServer(t)

			err = os.Setenv(api.EnvVaultToken, "myroot")
			require.NoError(t, err)
			err = os.Setenv(api.EnvVaultAddress, server.URL)
			require.NoError(t, err)

			defer server.Close()

			b, err := newBankVaultClient(tt.args.policy, zapr.NewLogger(zapLog))
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
