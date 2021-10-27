package outerda

import (
	"testing"
)

func TestMarshalRecord(t *testing.T) {
	d, _ := MarshalRecord(mockRecord())
	t.Log(string(d))
}
