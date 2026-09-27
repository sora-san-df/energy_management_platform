package dashboard

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/analysis"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

type fakeMeters struct{ items []meter.Overview }

func (f fakeMeters) List(context.Context, meter.ListQuery) ([]meter.Overview, error) {
	return f.items, nil
}

type fakeReadings struct{ items []reading.Reading }

func (f fakeReadings) ListAll(context.Context) ([]reading.Reading, error) { return f.items, nil }

type fakeAnomalies struct{ items []anomaly.Anomaly }

func (f fakeAnomalies) List(context.Context, anomaly.ListQuery) ([]anomaly.Anomaly, error) {
	return f.items, nil
}

type fakeRuns struct {
	run analysis.Run
	err error
}

func (f fakeRuns) Latest(context.Context) (analysis.Run, error) { return f.run, f.err }

var day0 = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func readingsFor(meterID string, perHour ...float64) []reading.Reading {
	var out []reading.Reading
	for d, v := range perHour {
		for h := 0; h < 24; h++ {
			out = append(out, reading.Reading{MeterID: meterID, Timestamp: day0.Add(time.Duration(d*24+h) * time.Hour), ConsumptionKWh: v})
		}
	}
	return out
}

func newService(runs fakeRuns) *Service {
	var rs []reading.Reading
	rs = append(rs, readingsFor("M-101", 10, 10)...)
	rs = append(rs, readingsFor("M-109", 20, 40)...)
	return NewService(
		fakeMeters{items: []meter.Overview{
			{Meter: meter.Meter{MeterID: "M-101", Status: meter.StatusOK}, BaselineDailyKWh: 240, CurrentDailyKWh: 240},
			{Meter: meter.Meter{MeterID: "M-109", Status: meter.StatusCritical}, BaselineDailyKWh: 480, CurrentDailyKWh: 960},
		}},
		fakeReadings{items: rs},
		fakeAnomalies{items: []anomaly.Anomaly{
			{ID: 1, MeterID: "M-109", Type: anomaly.TypeRealAnomaly, Severity: anomaly.SeverityHigh, Confidence: 0.96, Priority: 1},
			{ID: 2, MeterID: "M-106", Type: anomaly.TypeFalsePositive, Severity: anomaly.SeverityLow, Confidence: 0.84, Priority: 2},
		}},
		runs,
	)
}

func TestSummary(t *testing.T) {
	finished := day0.Add(15 * 24 * time.Hour)
	svc := newService(fakeRuns{run: analysis.Run{ID: 9, Status: analysis.RunStatusCompleted, FinishedAt: &finished}})

	s, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.MetersTotal != 2 || s.MetersByStatus[meter.StatusCritical] != 1 || s.MetersByStatus[meter.StatusOK] != 1 {
		t.Fatalf("unexpected meter counters %+v", s)
	}
	if s.ReadingsTotal != 96 || s.TotalConsumptionKWh != 240*2+480+960 {
		t.Fatalf("unexpected consumption %+v", s)
	}
	if s.CurrentDailyKWh != 1200 || s.BaselineDailyKWh != 720 || s.VariationPct < 66.6 || s.VariationPct > 66.7 {
		t.Fatalf("unexpected daily aggregates %+v", s)
	}
	if s.PeriodStart == nil || !s.PeriodStart.Equal(day0) || s.PeriodEnd == nil {
		t.Fatalf("unexpected period %+v %+v", s.PeriodStart, s.PeriodEnd)
	}
	if s.AnomaliesDetected != 2 || s.HighPriority != 1 || s.AvgConfidence != 0.9 || s.ConfidenceLevel != "ALTA" {
		t.Fatalf("unexpected anomaly KPIs %+v", s)
	}
	if s.AnomaliesByType[anomaly.TypeRealAnomaly] != 1 || len(s.TopAnomalies) != 2 || s.TopAnomalies[0].MeterID != "M-109" {
		t.Fatalf("unexpected anomaly breakdown %+v", s)
	}
	if len(s.DailyConsumption) != 2 || s.DailyConsumption[1].KWh != 1200 || s.DailyConsumption[1].BaselineKWh != 720 {
		t.Fatalf("unexpected daily consumption %+v", s.DailyConsumption)
	}
	if s.LastAnalysis == nil || s.LastAnalysis.ID != 9 {
		t.Fatalf("unexpected last analysis %+v", s.LastAnalysis)
	}
}

func TestSummaryWithoutAnalysis(t *testing.T) {
	svc := newService(fakeRuns{err: analysis.ErrRunNotFound})
	s, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.LastAnalysis != nil {
		t.Fatal("expected no last analysis")
	}
}

func TestSummaryPropagatesUnexpectedErrors(t *testing.T) {
	svc := newService(fakeRuns{err: errors.New("db down")})
	if _, err := svc.Summary(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestHandlerSummary(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(newService(fakeRuns{err: analysis.ErrRunNotFound})).Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"meters_total":2`) {
		t.Fatalf("unexpected response %d %s", rec.Code, rec.Body.String())
	}
}
