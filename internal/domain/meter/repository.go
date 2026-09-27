package meter

import (
	"context"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

type Repository interface {
	List(ctx context.Context) ([]Meter, error)
	Get(ctx context.Context, meterID string) (Meter, error)
}

type ReadingSource interface {
	ListByMeter(ctx context.Context, meterID string, from, to time.Time) ([]reading.Reading, error)
	ListAll(ctx context.Context) ([]reading.Reading, error)
}

type AnomalySource interface {
	List(ctx context.Context, filter anomaly.Filter) ([]anomaly.Anomaly, error)
}

type EventSource interface {
	ListByMeter(ctx context.Context, meterID string) ([]event.Event, error)
}
