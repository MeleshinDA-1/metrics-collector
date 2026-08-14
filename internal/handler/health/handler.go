package health

import (
	"context"
	"net/http"
)

type DBPinger interface {
	Ping(ctx context.Context) error
}

type PingHandler struct {
	pinger DBPinger
}

func NewPingHandler(pinger DBPinger) *PingHandler {
	return &PingHandler{pinger: pinger}
}

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
