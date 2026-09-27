package analysis

import (
	"errors"
	"fmt"
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
	mux.HandleFunc("POST /api/ai/analyze", h.analyze)
	mux.HandleFunc("GET /api/ai/analysis/latest", h.latest)
	mux.HandleFunc("GET /api/ai/analysis/{id}", h.get)
}

type analyzeRequest struct {
	MeterIDs []string `json:"meter_ids"`
}

type conflictResponse struct {
	Error    httpx.ErrorBody `json:"error"`
	Analysis Run             `json:"analysis"`
}

func (h *Handler) analyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	run, err := h.svc.Start(r.Context(), req.MeterIDs)
	switch {
	case errors.Is(err, ErrAnalysisInProgress):
		httpx.JSON(w, http.StatusConflict, conflictResponse{
			Error:    httpx.ErrorBody{Code: "analysis_in_progress", Message: "Ya hay un análisis en curso"},
			Analysis: run,
		})
		return
	case errors.Is(err, ErrUnknownMeter):
		httpx.Error(w, http.StatusBadRequest, "unknown_meter", err.Error())
		return
	case err != nil:
		httpx.InternalError(w)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/ai/analysis/%d", run.ID))
	httpx.JSON(w, http.StatusAccepted, run)
}

func (h *Handler) writeRun(w http.ResponseWriter, run Run, err error) {
	switch {
	case errors.Is(err, ErrRunNotFound):
		httpx.Error(w, http.StatusNotFound, "analysis_not_found", "Análisis no encontrado")
	case err != nil:
		httpx.InternalError(w)
	default:
		httpx.JSON(w, http.StatusOK, run)
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	run, err := h.svc.Get(r.Context(), id)
	h.writeRun(w, run, err)
}

func (h *Handler) latest(w http.ResponseWriter, r *http.Request) {
	run, err := h.svc.Latest(r.Context())
	h.writeRun(w, run, err)
}
