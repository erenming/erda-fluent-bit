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
