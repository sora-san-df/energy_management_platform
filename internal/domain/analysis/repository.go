package analysis

import (
	"context"
	"errors"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

var (
	ErrUnknownMeter       = errors.New("unknown meter")
	ErrAnalysisInProgress = errors.New("analysis already in progress")

	ErrExplainerDisabled        = errors.New("explainer disabled")
	ErrExplainerRateLimited     = errors.New("explainer rate limited")
	ErrExplainerTimeout         = errors.New("explainer timeout")
	ErrExplainerInvalidResponse = errors.New("explainer invalid response")
	ErrExplainerUnavailable     = errors.New("explainer unavailable")
)

type RunRepository interface {
	Create(ctx context.Context, r *Run) error
	Update(ctx context.Context, r Run) error
	Get(ctx context.Context, id int64) (Run, error)
	Latest(ctx context.Context) (Run, error)
}

type MeterSource interface {
	List(ctx context.Context) ([]meter.Meter, error)
	UpdateStatus(ctx context.Context, meterID string, status meter.Status) error
}

type ReadingSource interface {
	ListAll(ctx context.Context) ([]reading.Reading, error)
}

type EventSource interface {
	List(ctx context.Context) ([]event.Event, error)
}

type AnomalyStore interface {
	ReplaceForMeters(ctx context.Context, meterIDs []string, items []anomaly.Anomaly) ([]anomaly.Anomaly, error)
}

type ExplainRequest struct {
	Finding   Finding `json:"finding"`
	MeterName string  `json:"meter_name"`
	Location  string  `json:"location"`
}

type Explanation struct {
	Reason            string   `json:"reason"`
	RecommendedAction string   `json:"recommended_action"`
	Confidence        float64  `json:"confidence"`
	Insights          []string `json:"insights"`
}

type Explainer interface {
	Name() string
	Explain(ctx context.Context, req ExplainRequest) (Explanation, error)
}
