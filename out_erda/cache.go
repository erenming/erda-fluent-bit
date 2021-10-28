package outerda

import (
	"fmt"
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

func (mc *metadataCache) EnrichMetadataWithContainerInfo(cid string, lg *LogEvent) error {
	cinfo, ok := mc.dockerConfig.GetInfoByContainerID(cid)
	if !ok {
		return fmt.Errorf("can't find docker with cid<%s>", cid)
	}
	for k, v := range cinfo.EnvMap {
		lg.Tags[strings.ToLower(k)] = v
	}

	lg.Tags["pod_name"] = cinfo.Labels["io.kubernetes.pod.name"]
	lg.Tags["pod_namespace"] = cinfo.Labels["io.kubernetes.pod.namespace"]
	lg.Tags["pod_id"] = cinfo.Labels["io.kubernetes.pod.uid"]
	lg.Tags["container_id"] = string(cinfo.ID)
	lg.Tags["container_name"] = cinfo.Labels["io.kubernetes.container.name"]
	return nil
}
