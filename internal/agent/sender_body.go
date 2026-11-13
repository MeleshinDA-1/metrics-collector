package agent

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/MeleshinDA-1/metrics-collector/internal/compression"
)

type bodyEncoder func(io.Writer) error

func encodeJSON(value any) bodyEncoder {
	return func(dst io.Writer) error {
		return json.NewEncoder(dst).Encode(value)
	}
}

func writeRaw(data []byte) bodyEncoder {
	return func(dst io.Writer) error {
		_, err := dst.Write(data)

		return err
	}
}

func withGzipCompression(next bodyEncoder) bodyEncoder {
	return func(dst io.Writer) error {
		gz := compression.NewWriter(dst)

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
