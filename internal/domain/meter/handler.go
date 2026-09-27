package meter

import (
	"errors"
	"net/http"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/httpx"
)

var zeroTime time.Time

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/meters", h.list)
	mux.HandleFunc("GET /api/meters/{meterId}", h.get)
	mux.HandleFunc("GET /api/meters/{meterId}/readings", h.readings)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "meter_not_found", "Medidor no encontrado")
	case errors.Is(err, ErrInvalidQuery):
		httpx.Error(w, http.StatusBadRequest, "invalid_query", err.Error())
	default:
		httpx.InternalError(w)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.svc.List(r.Context(), ListQuery{
		Status: q.Get("status"),
		Search: q.Get("search"),
		Sort:   q.Get("sort"),
		Order:  q.Get("order"),
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.NewList(items))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	detail, err := h.svc.Get(r.Context(), r.PathValue("meterId"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

func (h *Handler) readings(w http.ResponseWriter, r *http.Request) {
	from, err := httpx.QueryTime(r, "from")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	to, err := httpx.QueryTime(r, "to")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	res, err := h.svc.Readings(r.Context(), r.PathValue("meterId"), ReadingsQuery{From: from, To: to, Granularity: r.URL.Query().Get("granularity")})
	if err != nil {
		h.writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
