package outerda

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type LogEvent struct {
	Source  string `json:"source"`
	ID      string `json:"id"`
	Stream  string `json:"stream"`
	Content string `json:"content"`
	// deprecated
	Offset    uint64            `json:"offset"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
	// deprecated. compatibility for log exporter
	Labels map[string]string `json:"labels"`
}

func (l *LogEvent) Size() int {
	size := len(l.Content) + len(l.ID) + len(l.Source) + len(l.Stream)
	for k, v := range l.Tags {
		size += len(k) + len(v)
	}
	return size
}

func NewBatchSender(service *remoteService) *BatchSender {
	bs := &BatchSender{
		batchLogEvent: make([]*LogEvent, 0, 10),
		remoteServer:  service,
	}

	return bs
}

type BatchSender struct {
	remoteServer  *remoteService
	batchLogEvent []*LogEvent
}

func (bs *BatchSender) SendLogEvent(lg *LogEvent) error {
	bs.batchLogEvent = append(bs.batchLogEvent, lg)
	return nil
}

func (bs *BatchSender) FlushAll() error {
	err := bs.flush(bs.batchLogEvent)
	if err != nil {
		return err
	}
	bs.Reset()
	return nil
}

func (bs *BatchSender) Reset() {
	bs.batchLogEvent = make([]*LogEvent, 0, 10)
}

func (bs *BatchSender) flush(data []*LogEvent) error {
	if len(data) == 0 {
		return nil
	}

	// In fluent-bit, logs from a same target(container or pod) always exist in same chunk.
	// So we cant get url from first log entry
	u := bs.remoteServer.cfg.URL
	if rsc := bs.remoteServer.cfg; rsc.URL == "" && rsc.URLFromLogLabel != "" {
		u = data[0].Labels[rsc.URLFromLogLabel]
	}
	logrus.Infof("cfg: %+v", bs.remoteServer.cfg)
	logrus.Infof("url: %s", u)
	err := bs.remoteServer.SendLogWithURL(bs.batchLogEvent, u)
	if err != nil {
		return fmt.Errorf("send remote: %w", err)
	}
	return nil
}
