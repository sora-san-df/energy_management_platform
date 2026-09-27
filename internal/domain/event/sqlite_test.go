package event

import (
	"context"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database/dbtest"
)

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLiteRepository(dbtest.New(t))
	ts := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)

	err := repo.InsertMany(ctx, []Event{
		{MeterID: "M-104", Timestamp: ts, Type: TypeOperationalChange, Description: "New production line activated"},
		{MeterID: "M-106", Timestamp: ts.Add(-72 * time.Hour), Type: TypeScheduledOutage, Description: "Outage"},
	})
	if err != nil {
		t.Fatal(err)
	}

	all, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].MeterID != "M-106" || all[0].ID == 0 {
		t.Fatalf("expected events ordered by timestamp, got %+v", all)
	}

	byMeter, err := repo.ListByMeter(ctx, "M-104")
	if err != nil {
		t.Fatal(err)
	}
	if len(byMeter) != 1 || byMeter[0].Type != TypeOperationalChange || !byMeter[0].Timestamp.Equal(ts) {
		t.Fatalf("unexpected events %+v", byMeter)
	}
}
