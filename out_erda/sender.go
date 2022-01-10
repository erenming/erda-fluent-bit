package outerda

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

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
	// deprecated
	Labels         map[string]string `json:"labels"`
	logAnalysisURL string
}

func (l *LogEvent) Size() int {
	size := len(l.Content) + len(l.ID) + len(l.Source) + len(l.Stream)
	for k, v := range l.Tags {
		size += len(k) + len(v)
	}
	return size
}

type batchConfig struct {
	remoteServer remoteServiceInf
	GzipLevel    int
}

type gzipper struct {
	buf    *bytes.Buffer
	writer *gzip.Writer
}

func NewBatchSender(cfg batchConfig) *BatchSender {
	bs := &BatchSender{
		buffer: make([]*LogEvent, 0, 100),
		cfg:    cfg,
	}
	if cfg.GzipLevel > 0 {
		buf := bytes.NewBuffer(nil)
		gc, _ := gzip.NewWriterLevel(buf, cfg.GzipLevel)
		bs.compressor = &gzipper{
			buf:    buf,
			writer: gc,
		}
	}

	return bs
}

type BatchSender struct {
	compressor *gzipper
	buffer     []*LogEvent
	cfg        batchConfig
}

func (bs *BatchSender) AddLogEvent(lg *LogEvent) error {
	bs.buffer = append(bs.buffer, lg)
	return nil
}

func (bs *BatchSender) FlushAll() error {
	err := bs.flush(bs.buffer)
	if err != nil {
		return err
	}
	bs.Reset()
	return nil
}

func (bs *BatchSender) Reset() {
	// reuse buffer
	bs.buffer = bs.buffer[:0]
}

func (bs *BatchSender) flush(data []*LogEvent) error {
	if len(data) == 0 {
		return nil
	}

	if rs := bs.cfg.remoteServer; rs.Type() == logAnalysis && rs.GetURL() == "" {
		rs.SetURL(data[0].logAnalysisURL)
	}

	buf, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("[out_erda] flushed json data: %s", string(buf))
	}

	if bs.cfg.GzipLevel > 0 && bs.compressor != nil {
		cbuf, err := bs.compress(buf)
		if err != nil {
			return fmt.Errorf("compress failed: %w", err)
		}
		buf = cbuf
	}

	err = bs.cfg.remoteServer.SendLog(buf)
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
		return nil, fmt.Errorf("gizp write data: %w", err)
	}
	if err := bs.compressor.writer.Flush(); err != nil {
		return nil, fmt.Errorf("gzip flush data: %w", err)
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
