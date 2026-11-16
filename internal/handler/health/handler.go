// Пакет health реализует эндпоинт проверки доступности PostgreSQL.
package health

import (
	"context"
	"net/http"
)

// DBPinger проверяет соединение с БД.
type DBPinger interface {
	// Ping возвращает nil при доступной БД и учитывает отмену ctx.
	Ping(ctx context.Context) error
}

// PingHandler сообщает доступность БД по HTTP.
type PingHandler struct {
	pinger DBPinger
}

// NewPingHandler создаёт обработчик с pinger; nil означает отсутствие соединения.
func NewPingHandler(pinger DBPinger) *PingHandler {
	return &PingHandler{pinger: pinger}
}

// PingDB отвечает на GET /ping: 200 с пустым телом при успешном Ping,
// 500 с пояснением при недоступной или ненастроенной БД.
func (ph *PingHandler) PingDB(w http.ResponseWriter, r *http.Request) {
	if ph.pinger == nil {
		http.Error(w, "database connection is not initialized", http.StatusInternalServerError)
		return
	}
	err := ph.pinger.Ping(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
