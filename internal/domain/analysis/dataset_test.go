package analysis

import (
	"path/filepath"
	"testing"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/dataset"
)

func TestDetectorAgainstProvidedDataset(t *testing.T) {
	data, err := dataset.LoadDir(filepath.Join("..", "..", "..", "data"))
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	byMeter := map[string][]reading.Reading{}
	for _, r := range data.Readings {
		byMeter[r.MeterID] = append(byMeter[r.MeterID], r)
	}
	eventsByMeter := map[string][]event.Event{}
	for _, e := range data.Events {
		eventsByMeter[e.MeterID] = append(eventsByMeter[e.MeterID], e)
	}

	detector := NewDetector(DefaultConfig())
	expected := map[string]struct {
		typ anomaly.Type
		sev anomaly.Severity
	}{
		"M-104": {anomaly.TypeExplainable, anomaly.SeverityMedium},
		"M-106": {anomaly.TypeFalsePositive, anomaly.SeverityLow},
		"M-109": {anomaly.TypeRealAnomaly, anomaly.SeverityHigh},
		"M-112": {anomaly.TypeDataQuality, anomaly.SeverityHigh},
	}

	var detected []anomaly.Anomaly
	for _, m := range data.Meters {
		f := detector.Detect(m.MeterID, byMeter[m.MeterID], eventsByMeter[m.MeterID])
		want, isCase := expected[m.MeterID]
		if !isCase {
			if f.Anomaly {
				t.Errorf("%s: unexpected anomaly %s/%s: %s", m.MeterID, f.Type, f.Severity, f.Reason)
			}
			continue
		}
		if !f.Anomaly || f.Type != want.typ || f.Severity != want.sev {
			t.Errorf("%s: got %s/%s, want %s/%s (%s)", m.MeterID, f.Type, f.Severity, want.typ, want.sev, f.Reason)
		}
		detected = append(detected, f.ToAnomaly(1))
	}

	if len(detected) != 4 {
		t.Fatalf("expected 4 anomalies, got %d", len(detected))
	}

	ranked := anomaly.Rank(detected)
	if ranked[0].MeterID != "M-109" {
		t.Fatalf("M-109 must be the top priority, got %s", ranked[0].MeterID)
	}
	high := 0
	for _, a := range ranked {
		if a.IsHighPriority() {
			high++
		}
	}
	if high != 2 {
		t.Fatalf("expected 2 high priority anomalies, got %d", high)
	}
	if ranked[len(ranked)-1].MeterID != "M-106" {
		t.Fatalf("false positive M-106 must be last, got %s", ranked[len(ranked)-1].MeterID)
	}
}

func TestDatasetM109Figures(t *testing.T) {
	data, err := dataset.LoadDir(filepath.Join("..", "..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	var rs []reading.Reading
	for _, r := range data.Readings {
		if r.MeterID == "M-109" {
			rs = append(rs, r)
		}
	}
	f := NewDetector(DefaultConfig()).Detect("M-109", rs, data.Events)

	if f.BaselineDailyKWh < 1000 || f.BaselineDailyKWh > 1100 {
		t.Fatalf("baseline should be ~1.050 kWh/day, got %.1f", f.BaselineDailyKWh)
	}
	if f.VariationPct <= 100 {
		t.Fatalf("variation should exceed 100%%, got %.1f", f.VariationPct)
	}
	if f.Shift == nil || f.Shift.Start.Hour() != 14 || f.Shift.Start.Day() != 12 {
		t.Fatalf("shift should start 2026-09-12 14:00, got %+v", f.Shift)
	}
	if f.Confidence < 0.9 {
		t.Fatalf("expected confidence >= 0.9, got %.2f", f.Confidence)
	}
}
