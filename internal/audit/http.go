package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type HTTPObserver struct {
	url    string
	client *http.Client
}

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
	}, nil
}

func (o *HTTPObserver) Notify(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode audit event: %w", err)
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
		return fmt.Errorf("audit receiver returned HTTP %d", resp.StatusCode)
	}
	return nil
}
