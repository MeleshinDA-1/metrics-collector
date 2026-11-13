package agent

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"runtime"
	"testing"
)

func TestCompressedBodiesRemainIndependent(t *testing.T) {
	var bodies [][]byte
	payloads := [][]byte{nil, []byte("first request"), bytes.Repeat([]byte("large-metric-value"), 8192), []byte("after large body")}
	for cycle := range 3 {
		for _, payload := range payloads {
			body, err := buildRequestBody(withGzipCompression(writeRaw(payload)))
			if err != nil {
				t.Fatal(err)
			}
			bodies = append(bodies, body)
		}
		if cycle == 1 {
			runtime.GC()
		}
	}
	// Decode after all encodes: returning a compressor must not invalidate an
	// earlier body which the sender may still need for a retry.
	for i, body := range bodies {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || !bytes.Equal(got, payloads[i%len(payloads)]) {
			t.Fatalf("body %d: decoded %d bytes, error %v, want %d bytes", i, len(got), err, len(payloads[i%len(payloads)]))
		}
	}
}

type failingBodyWriter struct{}

func (failingBodyWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCompressionRecoversAfterErrors(t *testing.T) {
	encoderError := errors.New("encode failed")
	if _, err := buildRequestBody(withGzipCompression(func(w io.Writer) error {
		_, _ = w.Write([]byte("partial body"))
		return encoderError
	})); !errors.Is(err, encoderError) {
		t.Fatalf("encode error = %v, want %v", err, encoderError)
	}
	if err := withGzipCompression(writeRaw([]byte("write failure")))(failingBodyWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write error = %v, want closed pipe", err)
	}
	for range 2 {
		body, err := buildRequestBody(withGzipCompression(writeRaw([]byte("recovered"))))
		if err != nil {
			t.Fatal(err)
		}
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || string(decoded) != "recovered" {
			t.Fatalf("after failure: body %q, error %v", decoded, err)
		}
	}
}
