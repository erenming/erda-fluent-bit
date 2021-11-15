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
	ass := assert.New(t)
	type fields struct {
		cfg Config
	}
	tests := []struct {
		name     string
		fields   fields
		wantSize int
	}{
		{
			name: "normal",
			fields: fields{
				cfg: Config{
					RootPath:           "testdata/containers",
					EnvIncludeList:     []string{"KUBE_DNS_SERVICE_HOST"},
					MaxExpiredDuration: time.Hour,
				},
			},
			wantSize: 2,
		},
		{
			name: "expired entry",
			fields: fields{
				cfg: Config{
					RootPath:           "testdata/containers",
					EnvIncludeList:     []string{"KUBE_DNS_SERVICE_HOST"},
					MaxExpiredDuration: time.Second,
					SyncInterval: 2*time.Second,
				},
			},
			wantSize: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ci := NewContainerInfoCenter(tt.fields.cfg)
			src, dst := filepath.Join("testdata", "container-b"), filepath.Join("testdata/containers", "container-b")
			err := copy2.Copy(src, dst, copy2.Options{
				Sync: true,
			})
			ass.Nil(err)
			ass.Nil(ci.scan())
			ass.Equal(2, len(ci.Data))

			ass.Nil(os.RemoveAll(dst))
			time.Sleep(ci.syncInterval)
			ass.Nil(ci.scan())
			// old data kept
			ass.Equal(tt.wantSize, len(ci.Data))
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
	ass.Equal(1, len(ci.Data))

	go ci.watchFileChange()

	src, dst := filepath.Join("testdata", "container-b"), filepath.Join("testdata/containers", "container-b")
	err := copy2.Copy(src, dst, copy2.Options{
		Sync: true,
	})
	ass.Nil(err)
	defer os.RemoveAll(dst)

	time.Sleep(2 * time.Second)
	ass.Equal(2, len(ci.Data))
}
