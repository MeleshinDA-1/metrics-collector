package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/retry"
)

// HTTPObserver отправляет события JSON-запросами POST.
// Поддерживает параллельные уведомления.
type HTTPObserver struct {
	url         string
	client      *http.Client
	retryPolicy retry.Policy
}

// NewHTTPObserver требует абсолютный HTTP(S) URL.
// Тайм-аут каждой попытки составляет две секунды; перенаправления отключены.
func NewHTTPObserver(address string) (*HTTPObserver, error) {
	u, err := url.Parse(address)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("audit URL must be an absolute HTTP or HTTPS URL")
	}
	return &HTTPObserver{
		url: address,
		client: &http.Client{
			Timeout: 2 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		retryPolicy: retry.DefaultPolicy(),
	}, nil
}

func (o *HTTPObserver) Notify(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode audit event: %w", err)
	}
	return o.retryPolicy.Do(ctx, func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create audit request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := o.client.Do(req)
		if err != nil {
			return fmt.Errorf("send audit event: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return httpStatusError(resp.StatusCode)
		}
		return nil
	}, func(err error) bool {
		return ctx.Err() == nil && isRetriableHTTPError(err)
	})
}

type httpStatusError int

func (e httpStatusError) Error() string {
	return fmt.Sprintf("audit receiver returned HTTP %d", e)
}

func isRetriableHTTPError(err error) bool {
	var status httpStatusError
	if errors.As(err, &status) {
		return status >= 500 && status < 600
	}
	var requestErr *url.Error
	if errors.As(err, &requestErr) {
		err = requestErr.Err
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsTimeout || dnsErr.IsTemporary
	}
	var networkErr net.Error
	return errors.As(err, &networkErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}
