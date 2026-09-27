package meter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

var day0 = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

type fakeRepo struct{ meters []Meter }

func (f *fakeRepo) List(context.Context) ([]Meter, error) { return f.meters, nil }
func (f *fakeRepo) Get(_ context.Context, id string) (Meter, error) {
	for _, m := range f.meters {
		if m.MeterID == id {
			return m, nil
		}
	}
	return Meter{}, ErrNotFound
}

type fakeReadings struct {
	byMeter  map[string][]reading.Reading
	lastFrom time.Time
	lastTo   time.Time
}

func (f *fakeReadings) ListByMeter(_ context.Context, id string, from, to time.Time) ([]reading.Reading, error) {
	f.lastFrom, f.lastTo = from, to
	var out []reading.Reading
	for _, r := range f.byMeter[id] {
		if (!from.IsZero() && r.Timestamp.Before(from)) || (!to.IsZero() && r.Timestamp.After(to)) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeReadings) ListAll(context.Context) ([]reading.Reading, error) {
	var out []reading.Reading
	for _, rs := range f.byMeter {
		out = append(out, rs...)
	}
	return out, nil
}

type fakeAnomalies struct{ items []anomaly.Anomaly }

func (f *fakeAnomalies) List(_ context.Context, filter anomaly.Filter) ([]anomaly.Anomaly, error) {
	var out []anomaly.Anomaly
	for _, a := range f.items {
		if filter.MeterID == "" || a.MeterID == filter.MeterID {
			out = append(out, a)
		}
	}
	return out, nil
}

type fakeEvents struct{ items []event.Event }

func (f *fakeEvents) ListByMeter(_ context.Context, id string) ([]event.Event, error) {
	var out []event.Event
	for _, e := range f.items {
		if e.MeterID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

func days(meterID string, perHour ...float64) []reading.Reading {
	var out []reading.Reading
	for d, v := range perHour {
		for h := 0; h < 24; h++ {
			out = append(out, reading.Reading{MeterID: meterID, Timestamp: day0.Add(time.Duration(d*24+h) * time.Hour), ConsumptionKWh: v, VoltageV: 220, CurrentA: v * 5, PowerFactor: 0.93, Status: "OK"})
		}
	}
	return out
}

func newTestService() (*Service, *fakeReadings) {
	readings := &fakeReadings{byMeter: map[string][]reading.Reading{
		"M-101": days("M-101", 34, 34, 34),
		"M-109": days("M-109", 43.75, 43.75, 91.75),
		"M-200": days("M-200", 10, 10, 10),
	}}
	svc := NewService(
		&fakeRepo{meters: []Meter{
			{ID: 1, MeterID: "M-101", Name: "Oficinas", Location: "Sede Central", Status: StatusOK},
			{ID: 2, MeterID: "M-109", Name: "Subestación Auxiliar", Location: "Planta Norte", Status: StatusCritical},
			{ID: 3, MeterID: "M-200", Name: "Bodega", Location: "Centro Logístico", Status: StatusPending},
		}},
		readings,
		&fakeAnomalies{items: []anomaly.Anomaly{
			{ID: 7, MeterID: "M-109", Type: anomaly.TypeRealAnomaly, Severity: anomaly.SeverityHigh, Confidence: 0.96, Status: anomaly.StatusOpen},
		}},
		&fakeEvents{items: []event.Event{{ID: 1, MeterID: "M-109", Timestamp: day0.Add(48 * time.Hour), Type: event.TypeUnknown}}},
	)
	return svc, readings
}

func find(items []Overview, id string) *Overview {
	for i := range items {
		if items[i].MeterID == id {
			return &items[i]
		}
	}
	return nil
}

func TestServiceListComputesConsumptionAndStatus(t *testing.T) {
	svc, _ := newTestService()

	items, err := svc.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 meters, got %d", len(items))
	}
	m109 := find(items, "M-109")
	if m109.Status != StatusCritical || m109.Anomaly == nil || m109.Anomaly.ID != 7 {
		t.Fatalf("unexpected M-109 overview %+v", m109)
	}
	if m109.CurrentDailyKWh != 2202 || m109.BaselineDailyKWh != 1050 || m109.VariationPct < 109 || m109.VariationPct > 110 {
		t.Fatalf("unexpected M-109 figures %+v", m109)
	}
	if m109.ReadingsCount != 72 || m109.LastReadingAt == nil {
		t.Fatalf("unexpected reading stats %+v", m109)
	}
	if find(items, "M-200").Status != StatusPending {
		t.Fatal("meters never analyzed must stay pending")
	}
	if find(items, "M-101").Anomaly != nil {
		t.Fatal("M-101 must not reference an anomaly")
	}
}

func TestServiceListStatusIsRecomputedFromActiveAnomalies(t *testing.T) {
	svc, _ := newTestService()
	svc.anomalies.(*fakeAnomalies).items[0].Status = anomaly.StatusResolved

	items, _ := svc.List(context.Background(), ListQuery{})
	if got := find(items, "M-109").Status; got != StatusOK {
		t.Fatalf("resolved anomaly should leave meter OK, got %s", got)
	}
}

func TestServiceListFiltersAndSearch(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	cases := []struct {
		name  string
		query ListQuery
		want  []string
	}{
		{"critical", ListQuery{Status: "critical"}, []string{"M-109"}},
		{"normal", ListQuery{Status: "normal"}, []string{"M-101"}},
		{"search by id", ListQuery{Search: "109"}, []string{"M-109"}},
		{"search by name", ListQuery{Search: "oficinas"}, []string{"M-101"}},
		{"sort by variation", ListQuery{Sort: "variation", Order: "desc"}, []string{"M-109", "M-101", "M-200"}},
		{"sort by consumption asc", ListQuery{Sort: "consumption", Order: "asc"}, []string{"M-200", "M-101", "M-109"}},
		{"sort by severity", ListQuery{Sort: "severity"}, []string{"M-109", "M-101", "M-200"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			items, err := svc.List(ctx, c.query)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != len(c.want) {
				t.Fatalf("got %d items, want %d", len(items), len(c.want))
			}
			for i, id := range c.want {
				if items[i].MeterID != id {
					t.Fatalf("position %d = %s, want %s", i, items[i].MeterID, id)
				}
			}
		})
	}
}

func TestServiceListRejectsInvalidQuery(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.List(context.Background(), ListQuery{Status: "weird"}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected ErrInvalidQuery for status, got %v", err)
	}
	if _, err := svc.List(context.Background(), ListQuery{Sort: "weird"}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected ErrInvalidQuery for sort, got %v", err)
	}
}

func TestServiceGet(t *testing.T) {
	svc, _ := newTestService()

	d, err := svc.Get(context.Background(), "M-109")
	if err != nil {
		t.Fatal(err)
	}
	if d.MeterID != "M-109" || len(d.Events) != 1 || len(d.Anomalies) != 1 {
		t.Fatalf("unexpected detail %+v", d)
	}
	if len(d.Daily) != 3 || d.Daily[2].KWh != 2202 || d.Daily[2].BaselineKWh != 1050 {
		t.Fatalf("unexpected daily series %+v", d.Daily)
	}
	if d.Latest == nil || d.Latest.ConsumptionKWh != 91.75 {
		t.Fatalf("unexpected latest reading %+v", d.Latest)
	}
	if d.Electrical.VoltageV != 220 {
		t.Fatalf("unexpected electrical averages %+v", d.Electrical)
	}

	if _, err := svc.Get(context.Background(), "M-999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceReadingsHourlyIncludesExpectedBaseline(t *testing.T) {
	svc, _ := newTestService()

	res, err := svc.Readings(context.Background(), "M-109", ReadingsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Granularity != "hour" || len(res.Points) != 72 {
		t.Fatalf("unexpected result %s %d", res.Granularity, len(res.Points))
	}
	if res.Points[0].ExpectedKWh != 43.75 {
		t.Fatalf("expected hourly baseline 43.75, got %v", res.Points[0].ExpectedKWh)
	}
}

func TestServiceReadingsDaily(t *testing.T) {
	svc, _ := newTestService()

	res, err := svc.Readings(context.Background(), "M-109", ReadingsQuery{Granularity: "day"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Points) != 3 || res.Points[2].ConsumptionKWh != 2202 || res.Points[2].ExpectedKWh != 1050 {
		t.Fatalf("unexpected daily points %+v", res.Points)
	}
	if res.Points[0].VoltageV != 220 || res.Points[0].PowerFactor != 0.93 {
		t.Fatalf("daily points must average electrical values, got %+v", res.Points[0])
	}
}

func TestServiceReadingsRange(t *testing.T) {
	svc, repo := newTestService()
	from, to := day0.Add(24*time.Hour), day0.Add(47*time.Hour)

	res, err := svc.Readings(context.Background(), "M-109", ReadingsQuery{From: from, To: to})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Points) != 24 || !repo.lastFrom.Equal(from) || !repo.lastTo.Equal(to) {
		t.Fatalf("expected ranged query, got %d points", len(res.Points))
	}
}

func TestServiceReadingsErrors(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.Readings(context.Background(), "M-999", ReadingsQuery{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if _, err := svc.Readings(context.Background(), "M-109", ReadingsQuery{Granularity: "week"}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected ErrInvalidQuery, got %v", err)
	}
}
