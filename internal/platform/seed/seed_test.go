package seed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/dataset"
)

type fakeMeters struct {
	count    int
	inserted []meter.Meter
}

func (f *fakeMeters) Count(context.Context) (int, error) { return f.count, nil }
func (f *fakeMeters) InsertMany(_ context.Context, ms []meter.Meter) error {
	f.inserted = append(f.inserted, ms...)
	return nil
}

type fakeReadings struct {
	inserted []reading.Reading
	err      error
}

func (f *fakeReadings) InsertMany(_ context.Context, rs []reading.Reading) error {
	if f.err != nil {
		return f.err
	}
	f.inserted = append(f.inserted, rs...)
	return nil
}

type fakeEvents struct{ inserted []event.Event }

func (f *fakeEvents) InsertMany(_ context.Context, es []event.Event) error {
	f.inserted = append(f.inserted, es...)
	return nil
}

func sampleData() dataset.Data {
	ts := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	return dataset.Data{
		Meters:   []meter.Meter{{MeterID: "M-101"}},
		Readings: []reading.Reading{{MeterID: "M-101", Timestamp: ts}},
		Events:   []event.Event{{MeterID: "M-101", Timestamp: ts}},
	}
}

func TestSeedLoadsDataWhenEmpty(t *testing.T) {
	m, r, e := &fakeMeters{}, &fakeReadings{}, &fakeEvents{}

	seeded, err := New(m, r, e).Run(context.Background(), sampleData())

	if err != nil || !seeded {
		t.Fatalf("expected seeding, got seeded=%v err=%v", seeded, err)
	}
	if len(m.inserted) != 1 || len(r.inserted) != 1 || len(e.inserted) != 1 {
		t.Fatalf("unexpected inserts: %d meters, %d readings, %d events", len(m.inserted), len(r.inserted), len(e.inserted))
	}
}

func TestSeedSkipsWhenDataExists(t *testing.T) {
	m, r, e := &fakeMeters{count: 12}, &fakeReadings{}, &fakeEvents{}

	seeded, err := New(m, r, e).Run(context.Background(), sampleData())

	if err != nil || seeded {
		t.Fatalf("expected skip, got seeded=%v err=%v", seeded, err)
	}
	if len(m.inserted) != 0 || len(r.inserted) != 0 {
		t.Fatal("must not insert when data exists")
	}
}

func TestSeedPropagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	_, err := New(&fakeMeters{}, &fakeReadings{err: boom}, &fakeEvents{}).Run(context.Background(), sampleData())
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
