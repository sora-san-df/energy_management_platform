package anomaly

import (
	"context"
	"errors"
	"testing"
)

type memoryRepo struct{ items []Anomaly }

func (m *memoryRepo) List(_ context.Context, f Filter) ([]Anomaly, error) {
	var out []Anomaly
	for _, a := range m.items {
		if f.MeterID != "" && a.MeterID != f.MeterID {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func (m *memoryRepo) Get(_ context.Context, id int64) (Anomaly, error) {
	for _, a := range m.items {
		if a.ID == id {
			return a, nil
		}
	}
	return Anomaly{}, ErrNotFound
}

func (m *memoryRepo) UpdateStatus(_ context.Context, id int64, s Status) error {
	for i := range m.items {
		if m.items[i].ID == id {
			m.items[i].Status = s
			return nil
		}
	}
	return ErrNotFound
}

func newMemoryService() *Service {
	return NewService(&memoryRepo{items: []Anomaly{
		{ID: 1, MeterID: "M-106", Type: TypeFalsePositive, Severity: SeverityLow, Confidence: 0.88, Status: StatusDismissed},
		{ID: 2, MeterID: "M-104", Type: TypeExplainable, Severity: SeverityMedium, Confidence: 0.92, Status: StatusOpen},
		{ID: 3, MeterID: "M-109", Type: TypeRealAnomaly, Severity: SeverityHigh, Confidence: 0.97, Status: StatusOpen},
		{ID: 4, MeterID: "M-112", Type: TypeDataQuality, Severity: SeverityHigh, Confidence: 0.97, Status: StatusOpen},
	}})
}

func TestServiceListIsRankedGlobally(t *testing.T) {
	svc := newMemoryService()

	items, err := svc.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"M-109", "M-112", "M-104", "M-106"}
	for i, id := range want {
		if items[i].MeterID != id || items[i].Priority != i+1 {
			t.Fatalf("position %d = %s (priority %d), want %s", i, items[i].MeterID, items[i].Priority, id)
		}
	}
}

func TestServiceListFiltersKeepGlobalPriority(t *testing.T) {
	svc := newMemoryService()
	ctx := context.Background()

	items, err := svc.List(ctx, ListQuery{Severity: SeverityHigh})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].MeterID != "M-112" || items[1].Priority != 2 {
		t.Fatalf("unexpected high severity list %+v", items)
	}

	items, _ = svc.List(ctx, ListQuery{Type: TypeFalsePositive})
	if len(items) != 1 || items[0].Priority != 4 {
		t.Fatalf("filtered item must keep its global priority, got %+v", items)
	}

	items, _ = svc.List(ctx, ListQuery{MeterID: "M-104"})
	if len(items) != 1 || items[0].ID != 2 {
		t.Fatalf("unexpected meter filter result %+v", items)
	}

	items, _ = svc.List(ctx, ListQuery{Status: StatusDismissed})
	if len(items) != 1 || items[0].MeterID != "M-106" {
		t.Fatalf("unexpected status filter result %+v", items)
	}
}

func TestServiceListRejectsInvalidFilters(t *testing.T) {
	svc := newMemoryService()
	for _, q := range []ListQuery{{Severity: "EXTREME"}, {Type: "WHATEVER"}, {Status: "LOST"}} {
		if _, err := svc.List(context.Background(), q); !errors.Is(err, ErrInvalidFilter) {
			t.Fatalf("expected ErrInvalidFilter for %+v, got %v", q, err)
		}
	}
}

func TestServiceGetIncludesPriority(t *testing.T) {
	svc := newMemoryService()
	a, err := svc.Get(context.Background(), 4)
	if err != nil {
		t.Fatal(err)
	}
	if a.MeterID != "M-112" || a.Priority != 2 {
		t.Fatalf("unexpected anomaly %+v", a)
	}
	if _, err := svc.Get(context.Background(), 99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceUpdateStatus(t *testing.T) {
	svc := newMemoryService()
	ctx := context.Background()

	a, err := svc.UpdateStatus(ctx, 3, StatusInvestigating)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != StatusInvestigating || a.Priority != 1 {
		t.Fatalf("unexpected anomaly %+v", a)
	}
	if _, err := svc.UpdateStatus(ctx, 3, Status("LOST")); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
	if _, err := svc.UpdateStatus(ctx, 99, StatusResolved); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
