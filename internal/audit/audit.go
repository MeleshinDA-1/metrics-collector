// Пакет audit передаёт события успешного обновления метрик файловым и HTTP-наблюдателям.
package audit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

const observerQueueSize = 64

// Event описывает успешный запрос обновления, включая пакетный.
type Event struct {
	// Timestamp содержит время события в секундах Unix.
	Timestamp int64 `json:"ts"`
	// Metrics содержит имена обновлённых метрик в порядке запроса, включая повторы.
	Metrics []string `json:"metrics"`
	// IPAddress содержит RemoteAddr клиента без порта, если адрес разобран.
	IPAddress string `json:"ip_address"`
}

// Observer получает события аудита. Notify должен учитывать отмену ctx.
// Изменять Metrics события нельзя.
type Observer interface {
	Notify(context.Context, Event) error
}

// Publisher доставляет события через отдельную очередь и работника для каждого наблюдателя.
type Publisher struct {
	mu            sync.RWMutex
	subscriptions []subscription
	closed        bool
	active        int
	done          chan struct{}
	cancel        context.CancelFunc
}

type subscription struct {
	observer Observer
	events   chan Event
}

// NewPublisher запускает по одной фоновой горутине на наблюдателя.
// Для завершения горутин вызовите Close.
func NewPublisher(observers ...Observer) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Publisher{
		active: len(observers),
		done:   make(chan struct{}),
		cancel: cancel,
	}
	for _, observer := range observers {
		s := subscription{observer: observer, events: make(chan Event, observerQueueSize)}
		p.subscriptions = append(p.subscriptions, s)
		go p.run(ctx, s)
	}
	if len(observers) == 0 {
		close(p.done)
	}
	return p
}

// Publish ставит событие в очереди без ожидания доставки.
// Для заполненной очереди событие отбрасывается и возвращается ошибка.
func (p *Publisher) Publish(event Event) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return errors.New("audit publisher is closed")
	}
	if len(p.subscriptions) == 0 {
		return nil
	}
	event.Metrics = append([]string(nil), event.Metrics...)
	var errs []error
	for _, s := range p.subscriptions {
		select {
		case s.events <- event:
		default:
			errs = append(errs, fmt.Errorf("audit queue for %T is full", s.observer))
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) run(ctx context.Context, s subscription) {
	defer func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.active--
		if p.active == 0 {
			close(p.done)
		}
	}()
	for event := range s.events {
		if ctx.Err() != nil {
			return
		}
		if err := s.observer.Notify(ctx, event); err != nil {
			slog.Error("unable to deliver metrics audit", "observer", fmt.Sprintf("%T", s.observer), "metrics", event.Metrics, "error", err)
		}
	}
}

// Close прекращает приём событий, ждёт доставки очередей и отменяет её при завершении ctx.
func (p *Publisher) Close(ctx context.Context) error {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		for _, s := range p.subscriptions {
			close(s.events)
		}
	}
	p.mu.Unlock()

	select {
	case <-p.done:
		p.cancel()
		return nil
	case <-ctx.Done():
		p.cancel()
		<-p.done
		return ctx.Err()
	}
}
