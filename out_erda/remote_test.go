package outerda

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_collectorService_sendWithPath(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := io.ReadAll(r.Body)
		assert.Nil(t, err)
		t.Logf("path: %s", r.URL.Path)
		t.Logf("headers: %+v", r.Header)
		t.Logf("body: %s", string(buf))
	}))
	defer ts.Close()

	type fields struct {
		cfg RemoteConfig
	}
	type args struct {
		data interface{}
		url  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "",
			fields: fields{cfg: RemoteConfig{
				Headers: map[string]string{
					"authorization":    "Bearer xxx",
					"Content-Type":     "application/json; charset=UTF-8",
					"Content-Encoding": "gzip",
				},
				URL:                  ts.URL,
				RequestTimeout:       10 * time.Second,
				KeepAliveIdleTimeout: 3 * time.Minute,
			}},
			args: args{
				data: "hello world",
				url:  hostJoinPath(ts.URL, "/collector"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCollectorService(tt.fields.cfg)
			if err := c.SendLogWithURL(tt.args.data, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("sendWithPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_hostJoinPath(t *testing.T) {
	type args struct {
		host string
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{
				host: "http://localhost:7096",
				path: "/collector/logs/container",
			},
			want: "http://localhost:7096/collector/logs/container",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostJoinPath(tt.args.host, tt.args.path); got != tt.want {
				t.Errorf("hostJoinPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
