package outerda

import (
	"fmt"
	"os"
	"strings"

	"github.com/erda-project/erda-for-fluent-bit/out_erda/sources/containerfile"
)

const diceClusterName = "DICE_CLUSTER_NAME"

type metadata struct {
	dockerConfigMeta *containerfile.ContainerInfoCenter

	cfg metadataConfig
}

type metadataConfig struct {
	dockerMetadataEnable bool
	dcfg                 containerfile.Config
}

type eventExtInfo struct {
	containerID string
	record      map[interface{}]interface{}
}

func newMetadata(cfg metadataConfig) *metadata {
	md := &metadata{
		cfg: cfg,
	}
	if cfg.dockerMetadataEnable {
		md.dockerConfigMeta = containerfile.NewContainerInfoCenter(cfg.dcfg)
	}
	return md
}

func (md *metadata) Close() error {
	if md.cfg.dockerMetadataEnable {
		if err := md.dockerConfigMeta.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (md *metadata) Start() error {
	if md.cfg.dockerMetadataEnable {
		err := md.dockerConfigMeta.Init()
		if err != nil {
			return fmt.Errorf("cannot init meta: %w", err)
		}

		md.dockerConfigMeta.Start()
	}
	return nil
}

func (md *metadata) EnrichMetadata(lg *LogEvent, ext *eventExtInfo) error {
	// default
	if v, ok := os.LookupEnv(diceClusterName); ok {
		lg.Tags["dice_cluster_name"] = v
		lg.Tags["cluster_name"] = v
	}

	if ext.record != nil {
		md.enrichWithErdaMetadata(lg, ext.record)
	}

	if md.cfg.dockerMetadataEnable && ext.containerID != "" {
		err := md.EnrichMetadataWithContainerInfo(ext.containerID, lg)
		if err != nil {
			return fmt.Errorf("EnrichMetadataWithContainerInfo err: %w", err)
		}
	}

	return nil
}

func (md *metadata) enrichWithErdaMetadata(lg *LogEvent, record map[interface{}]interface{}) {
	for k, v := range record {
		ks, ok := k.(string)
		if !ok {
			continue
		}
		if idx := strings.Index(ks, metaErdaPrefix); idx != -1 {
			vs, ok := v.([]byte)
			if ok {
				lg.Tags[ks[len(metaErdaPrefix):]] = bs2str(vs)
			}
		}
	}
}

func (md *metadata) EnrichMetadataWithContainerInfo(cid string, lg *LogEvent) error {
	cinfo, ok := md.dockerConfigMeta.GetInfoByContainerID(cid)
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

// deprecated
func (md *metadata) enrichWithKubernetesMetadata(lg *LogEvent, k8sInfo interface{}) {
	nk := unmarshalNestedKubernetes(k8sInfo)
	if nk == nil {
		return
	}

	lg.ID = nk.DockerID
	lg.Tags["pod_ip"] = nk.PodID
	lg.Tags["pod_name"] = nk.PodName
	lg.Tags["pod_namespace"] = nk.NamespaceName
	lg.Tags["pod_id"] = nk.PodID
	lg.Tags["container_id"] = nk.DockerID
	lg.Tags["container_name"] = nk.ContainerName
}

func unmarshalNestedKubernetes(data interface{}) *nestedKubernetes {
	mm, ok := data.(map[interface{}]interface{})
	if !ok {
		return nil
	}
	nk := &nestedKubernetes{}
	if v, ok := mm["pod_name"]; ok {
		nk.PodName = bs2str(v.([]byte))
	}
	if v, ok := mm["namespace_name"]; ok {
		nk.NamespaceName = bs2str(v.([]byte))
	}
	if v, ok := mm["pod_id"]; ok {
		nk.PodID = bs2str(v.([]byte))
	}
	if v, ok := mm["docker_id"]; ok {
		nk.DockerID = bs2str(v.([]byte))
	}
	if v, ok := mm["container_image"]; ok {
		nk.ContainerImage = bs2str(v.([]byte))
	}
	if v, ok := mm["container_name"]; ok {
		nk.ContainerName = bs2str(v.([]byte))
	}
	return nk
}
