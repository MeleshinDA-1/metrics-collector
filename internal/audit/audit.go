// Пакет audit передаёт события успешного обновления метрик файловым и HTTP-наблюдателям.
package audit

import (
	"context"
	"errors"
	"fmt"
)

// Event описывает успешный запрос обновления, включая пакетный.
type Event struct {
	// Timestamp содержит время события в секундах Unix.
	Timestamp int64 `json:"ts"`
	// Metrics содержит имена обновлённых метрик в порядке запроса, включая повторы.
	Metrics []string `json:"metrics"`
	// IPAddress содержит RemoteAddr клиента без порта, если адрес разобран.
	IPAddress string `json:"ip_address"`
}

// Observer получает события аудита. При параллельных запросах должен поддерживать
// одновременные Notify. Изменять Metrics события нельзя.
type Observer interface {
	Notify(context.Context, Event) error
}

// Publisher уведомляет всех наблюдателей, несмотря на ошибки.
// Подписки неизменны; параллельная работа требует потокобезопасных наблюдателей.
type Publisher struct {
	observers []Observer
}

func NewPublisher(observers ...Observer) *Publisher {
	return &Publisher{observers: append([]Observer(nil), observers...)}
}

// Publish синхронно уведомляет всех наблюдателей в порядке подписки и объединяет ошибки.
func (p *Publisher) Publish(ctx context.Context, event Event) error {
	var errs []error
	for _, observer := range p.observers {
		if err := observer.Notify(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("%T: %w", observer, err))
		}
	}
	return errors.Join(errs...)
}
