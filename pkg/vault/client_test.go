package vault

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/hashicorp/vault/api"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
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

			_, err := NewClient()
			if err != nil {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

func TestBankVaultClient_SetToken(t *testing.T) {
	type args struct {
		secretPath string
		token      string
		log        logr.Logger
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
				secretPath: "1234/6789",
				token:      "test",
				log:        zap.Logger(),
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
				secretPath: "1234/6789",
				token:      "test",
				log:        zap.Logger(),
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

			b, _ := NewClient()
			if err := b.SetToken(tt.args.secretPath, tt.args.token, tt.args.log); (err != nil) != tt.wantErr {
				t.Errorf("BankVaultClient.SetToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
