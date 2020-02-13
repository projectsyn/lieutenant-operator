package manager

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type testImplementation struct {
	found bool
}

func (t *testImplementation) IsType(URL *url.URL) (bool, error) {
	if strings.Contains(URL.String(), "notfound") {
		return false, nil
	}
	return t.found, nil
}

func (t *testImplementation) New(options RepoOptions) (Repo, error) {
	if strings.Contains(options.URL.String(), "fail") {
		return nil, fmt.Errorf("expected error")
	}
	return nil, nil
}

func TestNewRepo(t *testing.T) {

	Register(&testImplementation{found: false})
	Register(&testImplementation{found: true})
	Register(&testImplementation{found: false})

	type args struct {
		options RepoOptions
	}
	tests := []struct {
		name    string
		args    args
		want    Repo
		wantErr bool
	}{
		{
			name: "test successfull",
			args: args{
				options: RepoOptions{
					URL: &url.URL{Scheme: "http://", Host: "test"},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "test fail",
			args: args{
				options: RepoOptions{
					URL: &url.URL{Scheme: "http://", Host: "fail"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test not found",
			args: args{
				options: RepoOptions{
					URL: &url.URL{Scheme: "http://", Host: "notfound"},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRepo(tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}
