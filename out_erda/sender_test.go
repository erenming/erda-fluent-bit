package outerda

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

var mockLogEvent = &LogEvent{
	Source:    "container",
	ID:        "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
	Stream:    "stderr",
	Content:   "time=\"2021-10-12 16:00:14.130242184\" level=info msg=\"finish to run the task: executor K8S/MARATHONFORTERMINUSDEV (id: 1120384ca1, action: 5)\"\n",
	Timestamp: 1634025614130323755,
	Tags: map[string]string{
		"pod_name":              "scheduler-3feb156fc4-cf6b45b89-cwh5s",
		"pod_namespace":         "project-387-dev",
		"pod_id":                "ad05d65a-b8b0-4b7c-84f3-88a2abc11bde",
		"pod_ip":                "10.0.46.1",
		"container_id":          "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
		"dice_cluster_name":     "terminus-dev",
		"dice_application_name": "scheduler",
		"msp_env_id":            "abc111",
		"cluster_name":          "terminus-dev",
		"application_name":      "scheduler",
	},
}

type mockRemote struct {
	expected  []*LogEvent
	sendCount int
	url       string
}

func (m *mockRemote) SendLog(data []byte) error {
	m.expected = unmarshal(data)
	m.sendCount++
	return nil
}

func (m *mockRemote) GetURL() string {
	return "http://localhost/collector"
}

func (m *mockRemote) SetURL(u string) {
	m.url = u
}

func (m *mockRemote) Type() collectorType {
	return centralCollector
}

func TestBatchSender_flush(t *testing.T) {
	mr := &mockRemote{}
	cfg := batchConfig{
		GzipLevel:    3,
		remoteServer: mr,
	}
	bs := &BatchSender{
		buffer: make([]*LogEvent, 0, 10),
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

	ass := assert.New(t)
	source := []*LogEvent{
		mockLogEvent,
	}
	err := bs.flush(source)
	ass.Nil(err)
	ass.Equal(mr.expected, source)
}

func unmarshal(data []byte) []*LogEvent {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	out, err := io.ReadAll(gr)
	if err != nil {
		panic(err)
	}
	var res []*LogEvent
	err = json.Unmarshal(out, &res)
	if err != nil {
		panic(err)
	}
	return res
}
