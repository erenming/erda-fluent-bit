package containerfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	copy2 "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
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
				done: make(chan struct{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.envKeyInclude, NewContainerInfoCenter(tt.args.cfg).envKeyInclude)
			assert.Equal(t, tt.want.Data, NewContainerInfoCenter(tt.args.cfg).Data)
			assert.Equal(t, tt.want.globPattern, NewContainerInfoCenter(tt.args.cfg).globPattern)
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

func TestContainerInfoCenter_watchFileChange(t *testing.T) {
	ci := NewContainerInfoCenter(Config{
		RootPath:       "testdata/containers",
		EnvIncludeList: []string{"KUBE_DNS_SERVICE_HOST"},
	})
	ass := assert.New(t)
	ass.Nil(ci.initWatcher())
	ass.Nil(ci.scan())
	ass.Equal(filepath.Join(ci.rootPath, "123", configJson), ci.Data["1c3d08e59bb39ee1ab2fca95c6b6ed72d7662eac58074af1962bf3b3a113040b"].configFilePath)

	go ci.watchFileChange()

	err := copy2.Copy(filepath.Join(ci.rootPath, "123"), filepath.Join(ci.rootPath, "456"), copy2.Options{
		Sync: true,
	})
	ass.Nil(err)
	defer os.RemoveAll(filepath.Join(ci.rootPath, "456"))

	time.Sleep(2 * time.Second)
	ass.Equal(1, len(ci.Data))
	ass.Equal(filepath.Join(ci.rootPath, "456", configJson), ci.Data["1c3d08e59bb39ee1ab2fca95c6b6ed72d7662eac58074af1962bf3b3a113040b"].configFilePath)
}
