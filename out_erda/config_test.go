package outerda

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_LoadFromFLBPlugin(t *testing.T) {
	cfg := &Config{}
	finder := func(key string) string {
		switch key {
		case "compress_level": // int
			return "3"
		case "headers": // map
			return "authorization=Bearer xxx,name=abc"
		case "container_env_include": // slice
			return "abc,edf,ghi"
		case "request_timeout": // duration
			return "10s"
		case "docker_container_root_path": // string
			return "/var/lib/docker/containers"
		default:
			return ""
		}
	}
	ass := assert.New(t)
	ass.Nil(LoadFromFLBPlugin(cfg, finder))
	ass.Equal(3, cfg.CompressLevel)
	ass.Equal("/var/lib/docker/containers", cfg.DockerContainerRootPath)
	ass.Equal(10*time.Second, cfg.RemoteConfig.RequestTimeout)
	ass.Equal(map[string]string{
		"authorization": "Bearer xxx",
		"name":          "abc",
	}, cfg.RemoteConfig.Headers)
	ass.Equal([]string{"abc", "edf", "ghi"}, cfg.ContainerEnvInclude)
}

func TestConfig_Init(t *testing.T) {
	type fields struct {
		cfg Config
	}
	tests := []struct {
		name   string
		fields fields
		want   Config
	}{
		{
			name: "normal",
			fields: fields{
				cfg: Config{
					RemoteConfig: RemoteConfig{
						NetLimitBytesPerSecond: 100,
						Headers:                map[string]string{},
					},
					CompressLevel:               3,
					BatchEventContentLimitBytes: 800,
				},
			},
			want: Config{
				RemoteConfig: RemoteConfig{
					Headers: map[string]string{
						"Content-Type":     "application/json; charset=UTF-8",
						"Content-Encoding": "gzip",
					},
					NetLimitBytesPerSecond: 100,
				},
				CompressLevel:               3,
				BatchEventContentLimitBytes: 200,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.cfg.Init()
			assert.Equal(t, tt.want, tt.fields.cfg)
		})
	}
}
