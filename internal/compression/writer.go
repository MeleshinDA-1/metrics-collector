package compression

import (
	"compress/gzip"
	"io"
	"sync"
)

var writers = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// Writer owns a reusable gzip compressor until Close. Like gzip.Writer, a single
// Writer must not be used concurrently; separate Writers can be used in parallel.
type Writer struct {
	gzip *gzip.Writer
}

func NewWriter(dst io.Writer) *Writer {
	writer := writers.Get().(*gzip.Writer)
	writer.Reset(dst)
	return &Writer{gzip: writer}
}

func (w *Writer) Write(data []byte) (int, error) {
	if w.gzip == nil {
		return 0, io.ErrClosedPipe
	}
	return w.gzip.Write(data)
}

func (w *Writer) Close() error {
	if w.gzip == nil {
		return nil
	}
	writer := w.gzip
	w.gzip = nil
	err := writer.Close()
	// Drop the previous destination even after an error, so idle compressors do
	// not retain request/response buffers. Reset also clears gzip headers/errors.
	writer.Reset(io.Discard)
	writers.Put(writer)
	return err
}
