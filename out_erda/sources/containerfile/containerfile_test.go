package containerfile

import (
	"reflect"
	"testing"
)

func TestNewContainerInfoCenter(t *testing.T) {
	type args struct {
		cfg Config
	}
	tests := []struct {
		name string
		args args
		want *ContainerInfoCenter
	}{
		{
			name: "",
			args: args{cfg: Config{
				RootPath:       "testdata/containers",
				EnvIncludeList: []string{"KUBE_DNS_SERVICE_HOST"},
			}},
			want: &ContainerInfoCenter{
				rootPath:    "testdata/containers",
				globPattern: "testdata/containers/*/config.v2.json",
				envKeyInclude: map[string]struct{}{
					"KUBE_DNS_SERVICE_HOST": {},
				},
				Data: map[DockerContainerID]DockerContainerInfo{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewContainerInfoCenter(tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewContainerInfoCenter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainerInfoCenter_scan(t *testing.T) {
	type fields struct {
		cfg Config
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				cfg: Config{
					RootPath:       "testdata/containers",
					EnvIncludeList: []string{"KUBE_DNS_SERVICE_HOST"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ci := NewContainerInfoCenter(tt.fields.cfg)
			if err := ci.scan(); (err != nil) != tt.wantErr {
				t.Errorf("scan() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
