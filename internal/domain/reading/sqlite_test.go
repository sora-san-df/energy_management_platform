package reading

import (
	"context"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database/dbtest"
)

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLiteRepository(dbtest.New(t))
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	var rs []Reading
	for i := 0; i < 5; i++ {
		rs = append(rs, Reading{MeterID: "M-101", Timestamp: base.Add(time.Duration(4-i) * time.Hour), ConsumptionKWh: float64(i), VoltageV: 220, CurrentA: 10, PowerFactor: 0.9, Status: "OK"})
	}
	rs = append(rs, Reading{MeterID: "M-102", Timestamp: base, ConsumptionKWh: 99, VoltageV: 220, CurrentA: 10, PowerFactor: 0.9, Status: "OK"})

	if err := repo.InsertMany(ctx, rs); err != nil {
		t.Fatalf("insert: %v", err)
	}

	count, err := repo.Count(ctx)
	if err != nil || count != 6 {
		t.Fatalf("expected 6 readings, got %d (%v)", count, err)
	}

	got, err := repo.ListByMeter(ctx, "M-101", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 readings for M-101, got %d", len(got))
	}
	if !got[0].Timestamp.Equal(base) || got[0].ID == 0 {
		t.Fatalf("readings must be ordered by timestamp asc, got %+v", got[0])
	}

	ranged, err := repo.ListByMeter(ctx, "M-101", base.Add(time.Hour), base.Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(ranged) != 3 {
		t.Fatalf("expected inclusive range of 3 readings, got %d", len(ranged))
	}

	all, err := repo.ListAll(ctx)
	if err != nil || len(all) != 6 {
		t.Fatalf("expected 6 readings, got %d (%v)", len(all), err)
	}
}
