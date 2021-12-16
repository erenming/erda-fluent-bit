package outerda

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata_enrichWithErdaMetadata(t *testing.T) {
	type args struct {
		lg     *LogEvent
		record map[interface{}]interface{}
	}
	tests := []struct {
		name string
		args args
		want *LogEvent
	}{
		{
			name: "",
			args: args{
				lg: &LogEvent{
					Tags: map[string]string{},
				},
				record: map[interface{}]interface{}{
					"__meta_erda_level":      []byte( "INFO"),
					"__meta_erda_request_id": []byte( "abc"),
				},
			},
			want: &LogEvent{
				Tags: map[string]string{
					"level":      "INFO",
					"request_id": "abc",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &metadata{}
			md.enrichWithErdaMetadata(tt.args.lg, tt.args.record)
			assert.Equal(t, tt.want, tt.args.lg)
		})
	}
}
