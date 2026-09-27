package seed

import (
	"context"
	"fmt"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/dataset"
)

type MeterStore interface {
	Count(ctx context.Context) (int, error)
	InsertMany(ctx context.Context, meters []meter.Meter) error
}

type ReadingStore interface {
	InsertMany(ctx context.Context, readings []reading.Reading) error
}

type EventStore interface {
	InsertMany(ctx context.Context, events []event.Event) error
}

type Seeder struct {
	meters   MeterStore
	readings ReadingStore
	events   EventStore
}

func New(meters MeterStore, readings ReadingStore, events EventStore) *Seeder {
	return &Seeder{meters: meters, readings: readings, events: events}
}

func (s *Seeder) Run(ctx context.Context, data dataset.Data) (bool, error) {
	count, err := s.meters.Count(ctx)
	if err != nil {
		return false, fmt.Errorf("count meters: %w", err)
	}
	if count > 0 {
		return false, nil
	}
	if err := s.meters.InsertMany(ctx, data.Meters); err != nil {
		return false, fmt.Errorf("seed meters: %w", err)
	}
	if err := s.readings.InsertMany(ctx, data.Readings); err != nil {
		return false, fmt.Errorf("seed readings: %w", err)
	}
	if err := s.events.InsertMany(ctx, data.Events); err != nil {
		return false, fmt.Errorf("seed events: %w", err)
	}
	return true, nil
}
