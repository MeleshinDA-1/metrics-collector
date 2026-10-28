package audit

import (
	"context"
	"errors"
	"fmt"
)

type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Notify(context.Context, Event) error
}

// Publisher notifies every observer, even if another observer fails.
// Its subscriptions are fixed at startup, so concurrent requests can publish safely.
type Publisher struct {
	observers []Observer
}

func NewPublisher(observers ...Observer) *Publisher {
	return &Publisher{observers: append([]Observer(nil), observers...)}
}

func (p *Publisher) Publish(ctx context.Context, event Event) error {
	var errs []error
	for _, observer := range p.observers {
		if err := observer.Notify(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("%T: %w", observer, err))
		}
	}
	return errors.Join(errs...)
}
