package anomaly

import (
	"errors"
	"net/http"
	"strings"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/anomalies", h.list)
	mux.HandleFunc("GET /api/anomalies/{id}", h.get)
	mux.HandleFunc("PATCH /api/anomalies/{id}", h.update)
}

type updateRequest struct {
	Status Status `json:"status"`
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "anomaly_not_found", "Anomalía no encontrada")
	case errors.Is(err, ErrInvalidFilter), errors.Is(err, ErrInvalidStatus):
		httpx.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		httpx.InternalError(w)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.svc.List(r.Context(), ListQuery{
		MeterID:  q.Get("meter_id"),
		Type:     Type(strings.ToUpper(q.Get("type"))),
		Severity: Severity(strings.ToUpper(q.Get("severity"))),
		Status:   Status(strings.ToUpper(q.Get("status"))),
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.NewList(items))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	a, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	var req updateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	a, err := h.svc.UpdateStatus(r.Context(), id, Status(strings.ToUpper(string(req.Status))))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}
