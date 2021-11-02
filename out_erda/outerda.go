package outerda

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erda-project/erda-for-fluent-bit/out_erda/sources/containerfile"
	"github.com/fluent/fluent-bit-go/output"
)

var (
	ErrKeyMustExist = errors.New("entry key must exist")
	ErrTypeInvalid  = errors.New("invalid data type")
)

type Event struct {
	Record    map[interface{}]interface{}
	Timestamp time.Time
}

type Output struct {
	cfg            Config
	cache          *metadataCache
	batchContainer *BatchSender
	batchJob       *BatchSender
	remoteService  remoteServiceInf

	cancelFunc context.CancelFunc
}

func NewOutput(cfg Config) *Output {
	cfg.Init()
	svc := newCollectorService(cfg.RemoteConfig)

	return &Output{
		remoteService: svc,
		cfg:           cfg,
		cache: newMetadataCache(containerfile.Config{
			RootPath:           cfg.DockerContainerRootPath,
			EnvIncludeList:     cfg.ContainerEnvInclude,
			SyncInterval:       cfg.DockerConfigSyncInterval,
			MaxExpiredDuration: cfg.DockerConfigMaxExpiredDuration,
		}),
		batchContainer: NewBatchSender(batchConfig{
			BatchEventLimit:             cfg.BatchEventLimit,
			BatchEventContentLimitBytes: cfg.BatchEventContentLimitBytes,
			send2remoteServer:           svc.SendContainerLog,
			GzipLevel:                   cfg.CompressLevel,
		}),
		batchJob: NewBatchSender(batchConfig{
			BatchEventLimit:             cfg.BatchEventLimit,
			BatchEventContentLimitBytes: cfg.BatchEventContentLimitBytes,
			send2remoteServer:           svc.SendJobLog,
			GzipLevel:                   cfg.CompressLevel,
		}),
	}
}

func (o *Output) Start() error {
	err := o.cache.dockerConfig.Init()
	if err != nil {
		return fmt.Errorf("cannot init cache: %w", err)
	}

	o.cache.dockerConfig.Start()
	return nil
}

// AddEvent accepts a Record, process and send to the buffer, flushing the buffer if it is full
// the return value is one of: FLB_OK, FLB_RETRY
// 1. process event as LogEvent
// 2. add []byte(encoded LogEvent) to buffer if buffer is not full
// 3. flush when buffer is full, and if flush failed, print error and retry event
func (o *Output) AddEvent(event *Event) int {
	lg, err := o.Process(event.Timestamp, event.Record)
	if err != nil {
		LogError("Record process failed", err)
		return output.FLB_RETRY
	}

	switch lg.Source {
	case "job":
		err = o.batchJob.SendLogEvent(lg)
		if err != nil {
			LogError("batchJob send failed", err)
			return output.FLB_RETRY
		}
	default:
		err = o.batchContainer.SendLogEvent(lg)
		if err != nil {
			LogError("batchContainer send failed", err)
			return output.FLB_RETRY
		}
	}

	return output.FLB_OK
}

func (o *Output) Flush() error {
	err := o.batchContainer.FlushAll()
	if err != nil {
		return fmt.Errorf("batchContainer flush error: %w", err)
	}
	err = o.batchJob.FlushAll()
	if err != nil {
		return fmt.Errorf("batchJob flush error: %w", err)
	}
	return nil
}

func (o *Output) Reset() {
	o.batchContainer.Reset()
	o.batchJob.Reset()
}

func (o *Output) Process(timestamp time.Time, record map[interface{}]interface{}) (*LogEvent, error) {
	// offset, err := getAndConvert("offset", record, uint64(0))
	// if err != nil {
	// 	return nil, err
	// }

	stream, err := getAndConvert("stream", record, []byte("stdout"))
	if err != nil {
		return nil, err
	}
	content, err := getAndConvert("log", record, nil)
	if err != nil {
		return nil, err
	}

	var t time.Time
	if val, err := getTime(record); err != nil {
		LogInfo("cannot get time from record", err)
		t = timestamp
	} else {
		t = val
	}

	logPath := getLogPath(record)

	lg := &LogEvent{
		ID:        o.getDockerContainerIDFromLogPath(logPath),
		Source:    "container",
		Stream:    bs2str(stream.([]byte)),
		Content:   bs2str(stripNewLine(content.([]byte))),
		Timestamp: t.UnixNano(),
		Tags:      make(map[string]string),
	}

	err = o.enrichWithMetadata(lg, record)
	if err != nil {
		LogInfo("enrich metadata error. log_path: "+logPath, err)
	}

	return lg, nil
}

func stripNewLine(data []byte) []byte {
	l := len(data)
	if l > 0 && data[l-1] == '\n' {
		return data[:l-1]
	}
	return data
}

type nestedKubernetes struct {
	PodName        string
	NamespaceName  string
	PodID          string
	DockerID       string
	ContainerImage string
	ContainerName  string
}

func (o *Output) enrichWithMetadata(lg *LogEvent, record map[interface{}]interface{}) error {
	// k8sInfo, ok := record["kubernetes"]
	// if ok {
	// 	o.enrichWithKubernetesMetadata(lg, k8sInfo)
	// }

	err := o.cache.EnrichMetadataWithContainerInfo(lg.ID, lg)
	if err != nil {
		return err
	}

	o.businessLogic(lg)
	return nil
}

func (o *Output) getDockerContainerIDFromLogPath(logPath string) string {
	items := strings.Split(logPath, "/")
	if o.cfg.DockerContainerIDIndex < 0 {
		return items[len(items)+o.cfg.DockerContainerIDIndex]
	} else {
		return items[o.cfg.DockerContainerIDIndex]
	}
}

func (o *Output) enrichWithKubernetesMetadata(lg *LogEvent, k8sInfo interface{}) {
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

func (o *Output) businessLogic(lg *LogEvent) {
	if val, ok := lg.Tags["terminus_define_tag"]; ok {
		lg.ID = val
		lg.Source = "job"
	}

	internalPrefix := "dice_"
	for k, v := range lg.Tags {
		if idx := strings.Index(k, internalPrefix); idx != -1 {
			lg.Tags[k[len(internalPrefix):]] = v
		}
	}
}

func (o *Output) compress() ([]byte, error) {
	return nil, nil
}

func (o *Output) doHTTPRequest(data []byte) error {
	return nil
}

func (o *Output) Close() error {
	if o.cancelFunc != nil {
		o.cancelFunc()
	}
	return o.cache.dockerConfig.Close()
}
