package outerda

import (
	"strings"
	"time"

	"github.com/erda-project/erda-for-fluent-bit/out_erda/sources/containerfile"
)

type metadataCache struct {
	dockerConfig *containerfile.ContainerInfoCenter
}

func newMetadataCache(globPattern string, envIncludeList []string, syncInterval time.Duration) *metadataCache {
	dc := containerfile.NewContainerInfoCenter(containerfile.Config{
		RootPath:       globPattern,
		EnvIncludeList: envIncludeList,
		SyncInterval:   syncInterval,
	})

	return &metadataCache{
		dockerConfig: dc,
	}
}

func (mc *metadataCache) EnrichMetadataWithContainerEnv(cid string, lg *LogEvent) {
	cinfo, ok := mc.dockerConfig.GetInfoByContainerID(cid)
	if !ok {
		return
	}
	for k, v := range cinfo.EnvMap {
		lg.Tags[strings.ToLower(k)] = v
	}
}
