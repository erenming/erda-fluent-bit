package outerda

import (
	"testing"
)

func Test_prettyRecord(t *testing.T) {
	type args struct {
		record map[interface{}]interface{}
		depth  int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "",
			args: args{
				record: mockRecord(),
				depth:  1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrettyRecord(tt.args.record, tt.args.depth)
		})
	}
}

func TestMarshalRecord(t *testing.T) {
	d, _ := MarshalRecord(mockRecord())
	t.Log(string(d))
}