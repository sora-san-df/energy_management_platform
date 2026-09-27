package dashboard

import (
	"net/http"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dashboard/summary", h.summary)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	sum, err := h.svc.Summary(r.Context())
	if err != nil {
		httpx.InternalError(w)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}
