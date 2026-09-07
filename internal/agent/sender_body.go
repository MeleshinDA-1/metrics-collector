package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
)

type bodyEncoder func(io.Writer) error

func encodeJSON(value any) bodyEncoder {
	return func(dst io.Writer) error {
		return json.NewEncoder(dst).Encode(value)
	}
}

func withGzipCompression(next bodyEncoder) bodyEncoder {
	return func(dst io.Writer) error {
		gz := gzip.NewWriter(dst)

		if err := next(gz); err != nil {
			_ = gz.Close()
			return err
		}

		return gz.Close()
	}
}

func buildRequestBody(encode bodyEncoder) ([]byte, error) {
	var body bytes.Buffer
	if err := encode(&body); err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}
