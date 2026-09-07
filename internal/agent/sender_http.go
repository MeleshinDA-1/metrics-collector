package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
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

func (sender *metricsSender) post(ctx context.Context, url string, body []byte) error {
	return sender.retryPolicy.Do(ctx, func(ctx context.Context) error {
		return sender.doPost(ctx, url, body)
	}, isRetriableSendError)
}

func (sender *metricsSender) doPost(ctx context.Context, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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
