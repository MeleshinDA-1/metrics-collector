package agent

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

const (
	requestTimeout = 3 * time.Second
	jsonMediaType  = "application/json"
	gzipEncoding   = "gzip"
)

type transportError struct {
	Err error
}

func (e *transportError) Error() string {
	return e.Err.Error()
}

func (e *transportError) Unwrap() error {
	return e.Err
}

func isRetriableSendError(err error) bool {
	var transportErr *transportError
	return errors.As(err, &transportErr)
}

func (sender *metricsSender) post(url string, body []byte) error {
	return retry.Do(func() error {
		return sender.doPost(url, body)
	}, isRetriableSendError)
}

func (sender *metricsSender) doPost(url string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", jsonMediaType)
	req.Header.Set("Content-Encoding", gzipEncoding)

	resp, err := sender.httpClient.Do(req)
	if err != nil {
		return &transportError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func normalizeServerAddress(serverAddress string) string {
	serverAddress = strings.TrimRight(serverAddress, "/")
	if strings.HasPrefix(serverAddress, "http://") || strings.HasPrefix(serverAddress, "https://") {
		return serverAddress
	}

	return "http://" + serverAddress
}
