package outerda

import (
	"bytes"
	"compress/gzip"
)

type gzipCompressor struct {
	w *gzip.Writer
}

func newGzipCompressor() *gzipCompressor {
	var buf bytes.Buffer
	return &gzipCompressor{
		w: gzip.NewWriter(&buf),
	}
}

func (gc *gzipCompressor) Compress(data []byte) error {
	return nil
}
