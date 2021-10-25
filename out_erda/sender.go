package outerda

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type LogEvent struct {
	Source    string            `json:"source"`
	ID        string            `json:"id"`
	Stream    string            `json:"stream"`
	Content   string            `json:"content"`
	// Offset    uint64            `json:"offset"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

type batchConfig struct {
	send2remoteServer           func(data []byte) error
	BatchEventLimit             int
	BatchTriggerTimeout         time.Duration
	BatchNetWriteBytesPerSecond int
	BatchEventContentLimitBytes int
	GzipLevel                   int
}

type gzipper struct {
	buf    *bytes.Buffer
	writer *gzip.Writer
}

func NewBatchSender(cfg batchConfig) *BatchSender {
	bs := &BatchSender{
		timeTrigger:   time.NewTimer(cfg.BatchTriggerTimeout),
		batchLogEvent: make([]*LogEvent, cfg.BatchEventLimit),
		cfg:           cfg,
	}
	if cfg.GzipLevel > 0 {
		buf := bytes.NewBuffer(nil)
		gc, _ := gzip.NewWriterLevel(buf, cfg.GzipLevel)
		bs.compressor = &gzipper{
			buf:    buf,
			writer: gc,
		}
	}

	go bs.timerCheck()
	return bs
}

type BatchSender struct {
	// todo WAL
	compressor    *gzipper
	batchLogEvent []*LogEvent
	cfg           batchConfig

	timeTrigger        *time.Timer
	currentIndex       int
	currentContentSize int
}

func (bs *BatchSender) SendLogEvent(lg *LogEvent) error {
	exceedEventLimit := bs.currentIndex >= bs.cfg.BatchEventLimit
	exceedContent := (bs.currentContentSize + len(lg.Content)) > bs.cfg.BatchEventContentLimitBytes

	if exceedEventLimit || exceedContent {
		err := bs.flush(bs.batchLogEvent[:bs.currentIndex])
		if err != nil {
			return err
		}
		bs.reset()
	}

	bs.batchLogEvent[bs.currentIndex] = lg
	bs.currentContentSize += len(lg.Content)
	bs.currentIndex++
	return nil
}

func (bs *BatchSender) timerCheck() {
	for {
		select {
		case <-bs.timeTrigger.C:
			logrus.Debugf("timeTrigger trigger started")
			for {
				err := bs.flush(bs.batchLogEvent[:bs.currentIndex])
				if err != nil {
					LogError("timeTrigger triggered flush failed, retry after 5 seconds", err)
					time.Sleep(time.Second * 5)
				} else {
					break
				}
			}
			bs.reset()
			// todo stop
		}
	}

}

func (bs *BatchSender) reset() {
	bs.currentIndex = 0
	bs.currentContentSize = 0
	bs.timeTrigger.Reset(bs.cfg.BatchTriggerTimeout)
}

func (bs *BatchSender) flush(data []*LogEvent) error {
	if len(data) == 0 {
		return nil
	}

	buf, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	if bs.cfg.GzipLevel > 0 && bs.compressor != nil {
		cbuf, err := bs.compress(buf)
		if err != nil {
			return fmt.Errorf("compress failed: %w", err)
		}
		buf = cbuf
	}

	err = bs.cfg.send2remoteServer(buf)
	if err != nil {
		return fmt.Errorf("send remote: %w", err)
	}
	return nil
}

func (bs *BatchSender) compress(data []byte) ([]byte, error) {
	defer func() {
		bs.compressor.buf.Reset()
		bs.compressor.writer.Reset(bs.compressor.buf)
	}()
	if _, err := bs.compressor.writer.Write(data); err != nil {
		return nil, fmt.Errorf("gizp write data: %w",err)
	}
	if err := bs.compressor.writer.Flush(); err != nil {
		return nil, fmt.Errorf("gzip flush data: %w",err)
	}
	if err := bs.compressor.writer.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	buf := bytes.NewBuffer(nil) // todo init size?
	if _, err := io.Copy(buf, bs.compressor.buf); err != nil {
		return nil, fmt.Errorf("gzip copy: %w", err)
	}
	return buf.Bytes(), nil
}
