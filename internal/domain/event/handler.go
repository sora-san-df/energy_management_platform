package event

import (
	"context"
	"net/http"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/httpx"
)

type Lister interface {
	List(ctx context.Context) ([]Event, error)
	ListByMeter(ctx context.Context, meterID string) ([]Event, error)
}

type Handler struct {
	events Lister
}

func NewHandler(events Lister) *Handler {
	return &Handler{events: events}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events", h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	var items []Event
	var err error
	if meterID := r.URL.Query().Get("meter_id"); meterID != "" {
		items, err = h.events.ListByMeter(r.Context(), meterID)
	} else {
		items, err = h.events.List(r.Context())
	}
	if err != nil {
		httpx.InternalError(w)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.NewList(items))
}
