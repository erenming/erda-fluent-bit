package outerda

import (
	"bytes"
	"compress/gzip"
	"io"
	"math"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBatchSender_SendLogEvent(t *testing.T) {
	logrus.SetLevel(logrus.InfoLevel)

	type fields struct {
		dataNum                     int
		batchEventSize              int
		waitDuration                time.Duration
		BatchEventLimit             int
		BatchEventContentLimitBytes int
	}
	type args struct {
		lg *LogEvent
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "event limit trigger mode",
			fields: fields{
				dataNum:                     1000,
				batchEventSize:              10,
				waitDuration:                time.Second * 2,
				BatchEventLimit:             10,
				BatchEventContentLimitBytes: math.MaxInt64,
			},
			args: args{
				lg: mockLogEvent,
			},
			want: 100,
		},
		{
			name: "content limit trigger mode",
			fields: fields{
				dataNum:                     1000,
				batchEventSize:              10,
				waitDuration:                time.Second * 2,
				BatchEventLimit:             1001,
				BatchEventContentLimitBytes: len(mockLogEvent.Content) * 10,
			},
			args: args{
				lg: mockLogEvent,
			},
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mm := &mockRemote{}
			bs := NewBatchSender(batchConfig{
				remoteServer:                mm,
				BatchEventLimit:             tt.fields.BatchEventLimit,
				BatchEventContentLimitBytes: tt.fields.BatchEventContentLimitBytes,
				GzipLevel:                   3,
			})
			for i := 0; i < tt.fields.dataNum; i++ {
				err := bs.SendLogEvent(tt.args.lg)
				if err != nil {
					t.Errorf("get error: %s", err)
				}
			}
			assert.Nil(t, bs.FlushAll())
			assert.Equal(t, tt.want, mm.sendCount)
			assert.Equal(t, 0, bs.currentIndex)
			assert.Equal(t, 0, bs.currentContentSizeBytes)
		})
	}
}

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
