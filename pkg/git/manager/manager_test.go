package manager

import (
	"fmt"
	"reflect"
	"testing"
)

type testImplementation struct {
	found bool
}

func (t *testImplementation) IsType(URL string) (bool, error) {
	if URL == "notfound" {
		return false, nil
	}
	return t.found, nil
}

func (t *testImplementation) New(URL string, options RepoOptions) (Repo, error) {
	if URL == "fail" {
		return nil, fmt.Errorf("expected error")
	}
	return nil, nil
}

func TestNewRepo(t *testing.T) {

	Register(&testImplementation{found: false})
	Register(&testImplementation{found: true})
	Register(&testImplementation{found: false})

	type args struct {
		URL   string
		creds RepoOptions
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
				URL:   "test",
				creds: RepoOptions{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "test fail",
			args: args{
				URL:   "fail",
				creds: RepoOptions{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test not found",
			args: args{
				URL:   "notfound",
				creds: RepoOptions{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRepo(tt.args.URL, tt.args.creds)
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
