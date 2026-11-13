// Пакет metrics реализует текстовые, JSON- и HTML-эндпоинты метрик.
package metrics

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/MeleshinDA-1/metrics-collector/internal/audit"
	"github.com/MeleshinDA-1/metrics-collector/internal/model"
	"github.com/gorilla/mux"
)

type MetricsSnapshot = model.MetricsSnapshot

// MetricsStorage задаёт операции MetricsHandler с поддержкой параллельных запросов.
type MetricsStorage interface {
	SetGauge(ctx context.Context, name string, value float64) error
	AddCounter(ctx context.Context, name string, delta int64) (int64, error)
	// UpdateBatch применяет пакет с проверенными типами и Value/Delta не nil.
	UpdateBatch(ctx context.Context, metrics []model.Metrics) error
	// GetGauge возвращает значение и признак наличия метрики.
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	// GetCounter возвращает итог и признак наличия метрики.
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	Snapshot(ctx context.Context) (MetricsSnapshot, error)
}

// MetricsHandler читает и обновляет метрики, синхронно отправляет успешные обновления в аудит.
// Ошибки аудита журналируются и не меняют HTTP-ответ.
type MetricsHandler struct {
	storage MetricsStorage
	audit   *audit.Publisher
}

// NewMetricsHandler требует хранилище не nil; наблюдатели аудита необязательны.
func NewMetricsHandler(storage MetricsStorage, observers ...audit.Observer) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
		audit:   audit.NewPublisher(observers...),
	}
}

func (handler *MetricsHandler) auditMetrics(req *http.Request, names ...string) {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		ip = req.RemoteAddr
	}
	event := audit.Event{Timestamp: time.Now().Unix(), Metrics: names, IPAddress: ip}
	// Сохранённые метрики требуют аудита даже после отключения клиента.
	if err := handler.audit.Publish(context.WithoutCancel(req.Context()), event); err != nil {
		slog.Error("unable to deliver metrics audit", "metrics", names, "error", err)
	}
}

func (handler *MetricsHandler) auditBatch(req *http.Request, metrics []model.Metrics) {
	names := make([]string, len(metrics))
	for i, metric := range metrics {
		names[i] = metric.ID
	}
	handler.auditMetrics(req, names...)
}

func pathValue(req *http.Request, name string) string {
	return mux.Vars(req)[name]
}

func writeError(res http.ResponseWriter, status int, err error) {
	if status >= http.StatusInternalServerError {
		http.Error(res, http.StatusText(status), status)
		return
	}

	http.Error(res, err.Error(), status)
}
