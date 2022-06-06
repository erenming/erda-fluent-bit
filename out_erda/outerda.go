package outerda

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/sirupsen/logrus"
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
	batchLogExport *BatchSender
	remoteService  *remoteService

	cancelFunc context.CancelFunc
}

func NewOutput(cfg Config) *Output {
	cfg.Init()
	logrus.Infof("cfg: %+v", cfg)
	return &Output{
		cfg:            cfg,
		batchLogExport: NewBatchSender(newCollectorService(cfg.RemoteConfig)),
	}
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

	err = o.batchLogExport.SendLogEvent(lg)
	if err != nil {
		LogError("batchLogExport send failed", err)
		return output.FLB_RETRY
	}

	return output.FLB_OK
}

func (o *Output) Flush() error {
	err := o.batchLogExport.FlushAll()
	if err != nil {
		return fmt.Errorf("batchLogExport flush error: %w", err)
	}
	return nil
}

func (o *Output) Reset() {
	o.batchLogExport.Reset()
}

func (o *Output) Process(timestamp time.Time, record map[interface{}]interface{}) (*LogEvent, error) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("record: %s", jsonRecord(record))
	}

	id, err := getAndConvert("id", record, "")
	if err != nil {
		return nil, fmt.Errorf("can't get id from record: %w", err)
	}
	stream, err := getAndConvert("stream", record, "stdout")
	if err != nil {
		return nil, err
	}
	content, err := getAndConvert("content", record, "")
	if err != nil {
		return nil, err
	}

	var t time.Time
	if val, err := getTime(record); err != nil {
		t = timestamp
	} else {
		t = val
	}

	tags, err := getAndConvert("tags", record, map[string]string{})
	if err != nil {
		LogInfo("can't get tags from record", err)
	}

	labels, err := getAndConvert("labels", record, map[string]string{})
	if err != nil {
		LogInfo("can't get labels from record", err)
	}

	lg := &LogEvent{
		ID:        id.(string),
		Source:    "container",
		Stream:    stream.(string),
		Content:   content.(string),
		Timestamp: t.UnixNano(),
		Tags:      tags.(map[string]string),
		Labels:    labels.(map[string]string),
	}
	return lg, nil
}


func (o *Output) Close() error {
	if o.cancelFunc != nil {
		o.cancelFunc()
	}
	return nil
}
