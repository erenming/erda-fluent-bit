package outerda

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
	cfg.RemoteConfig.Headers["Content-Type"] = "application/json; charset=UTF-8"
	if cfg.CompressLevel > 0 {
		cfg.RemoteConfig.Headers["Content-Encoding"] = "gzip"
	}
	svc := newCollectorService(cfg.RemoteConfig)

	return &Output{
		remoteService: svc,
		cfg:           cfg,
		cache:         newMetadataCache(cfg.DockerContainerRootPath, cfg.ContainerEnvInclude, cfg.DockerConfigSyncInterval),
		batchContainer: NewBatchSender(batchConfig{
			BatchEventLimit:             cfg.BatchEventLimit,
			BatchTriggerTimeout:         cfg.BatchTriggerTimeout,
			BatchNetWriteBytesPerSecond: cfg.BatchNetWriteBytesPerSecond,
			BatchEventContentLimitBytes: cfg.BatchEventContentLimitBytes,
			send2remoteServer:           svc.SendContainerLog,
			GzipLevel:                   cfg.CompressLevel,
		}),
		batchJob: NewBatchSender(batchConfig{
			BatchEventLimit:             cfg.BatchEventLimit / 4,
			BatchTriggerTimeout:         2 * time.Second,
			BatchNetWriteBytesPerSecond: cfg.BatchNetWriteBytesPerSecond,
			BatchEventContentLimitBytes: cfg.BatchEventContentLimitBytes / 4,
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

func (o *Output) Process(timestamp time.Time, record map[interface{}]interface{}) (*LogEvent, error) {
	offset, err := getAndConvert("offset", record)
	if err != nil {
		return nil, err
	}

	stream, err := getAndConvert("stream", record)
	if err != nil {
		return nil, err
	}
	content, err := getAndConvert("log", record)
	if err != nil {
		return nil, err
	}

	var t time.Time
	if val, err := getTime(record); err != nil {
		LogInfo("cannot get time from Record", err)
		t = timestamp
	} else {
		t = val
	}

	lg := &LogEvent{
		Stream:    bs2str(stream.([]byte)),
		Content:   bs2str(content.([]byte)),
		Offset:    offset.(uint64),
		Timestamp: t.UnixNano(),
	}

	err = o.enrichWithMetadata(lg, record)
	if err != nil {
		LogInfo("some error happened when enrich metadata", err)
	}

	return lg, nil
}

// TODO auto_retry_requests
func (o *Output) Send(lg *LogEvent) error {
	data, err := o.compress()
	if err != nil {
		return fmt.Errorf("compress data: %w", err)
	}
	return o.doHTTPRequest(data)
}

type nestedKubernetes struct {
	PodName        string `mapstructure:"pod_name"`
	NamespaceName  string `mapstructure:"namespace"`
	PodID          string `mapstructure:"pod_id"`
	DockerID       string `mapstructure:"docker_id"`
	ContainerImage string `mapstructure:"container_image"`
	ContainerName  string `mapstructure:"container_name"`
}

func unmarshalNestedKubernetes(data interface{}) (*nestedKubernetes, error) {
	mm, ok := data.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("must be map[string]string")
	}
	return &nestedKubernetes{
		PodName:        bs2str(mm["pod_name"].([]byte)),
		NamespaceName:  bs2str(mm["namespace_name"].([]byte)),
		PodID:          bs2str(mm["pod_id"].([]byte)),
		DockerID:       bs2str(mm["docker_id"].([]byte)),
		ContainerImage: bs2str(mm["container_image"].([]byte)),
		ContainerName:  bs2str(mm["container_name"].([]byte)),
	}, nil
}

func (o *Output) enrichWithMetadata(lg *LogEvent, record map[interface{}]interface{}) error {
	k8sInfo, ok := record["kubernetes"]
	if !ok {
		return fmt.Errorf("key kubernetes: %w", ErrKeyMustExist)
	}

	// todo no copy way
	nk, err := unmarshalNestedKubernetes(k8sInfo)
	if err != nil {
		return fmt.Errorf("decode nested kubernetes: %w", err)
	}

	lg.ID = nk.DockerID
	lg.Source = "container"
	lg.Tags = make(map[string]string)
	lg.Tags["pod_ip"] = nk.PodID
	lg.Tags["pod_name"] = nk.PodName
	lg.Tags["pod_namespace"] = nk.NamespaceName
	lg.Tags["pod_id"] = nk.PodID
	lg.Tags["container_id"] = nk.DockerID
	// lg.Tags["container_image"] = nk.ContainerImage
	// lg.Tags["container_name"] = nk.ContainerName

	o.cache.EnrichMetadataWithContainerEnv(nk.DockerID, lg)

	o.businessLogic(lg)
	return nil
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
	return nil
}
