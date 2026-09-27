package analysis

import (
	"math"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

var t0 = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func at(day, hour int) time.Time {
	return t0.Add(time.Duration(day*24+hour) * time.Hour)
}

type series []reading.Reading

func baseSeries(meterID string, days int, dailyKWh float64) series {
	out := make(series, 0, days*24)
	for i := 0; i < days*24; i++ {
		ts := t0.Add(time.Duration(i) * time.Hour)
		shape := 1 + 0.25*math.Sin(float64(ts.Hour())/24*2*math.Pi)
		noise := 1 + 0.03*math.Sin(float64(i)*1.7)
		kwh := dailyKWh / 24 * shape * noise
		pf := 0.93
		v := 220 + math.Sin(float64(i)*0.9)
		out = append(out, reading.Reading{
			MeterID:        meterID,
			Timestamp:      ts,
			ConsumptionKWh: kwh,
			VoltageV:       v,
			CurrentA:       kwh * 1000 / (v * pf),
			PowerFactor:    pf,
			Status:         "OK",
		})
	}
	return out
}

func (s series) scale(from, to time.Time, factor, pf float64) series {
	for i := range s {
		if !s[i].Timestamp.Before(from) && s[i].Timestamp.Before(to) {
			s[i].ConsumptionKWh *= factor
			if pf > 0 {
				s[i].PowerFactor = pf
			}
			s[i].CurrentA = s[i].ConsumptionKWh * 1000 / (s[i].VoltageV * s[i].PowerFactor)
		}
	}
	return s
}

func (s series) corruptElectrical(from time.Time) series {
	voltages := []float64{241, 203, 239, 202}
	pfs := []float64{0.98, 0.58, 0.72, 0.58}
	k := 0
	for i := range s {
		if !s[i].Timestamp.Before(from) && s[i].Timestamp.Hour()%3 == 0 {
			s[i].VoltageV = voltages[k%len(voltages)]
			s[i].PowerFactor = pfs[k%len(pfs)]
			s[i].CurrentA = 160 - float64(k%3)*50
			k++
		}
	}
	return s
}

func end(s series) time.Time {
	return s[len(s)-1].Timestamp.Add(time.Hour)
}

func detect(t *testing.T, s series, events ...event.Event) Finding {
	t.Helper()
	return NewDetector(DefaultConfig()).Detect(s[0].MeterID, s, events)
}

func TestDetectNormalMeterHasNoAnomaly(t *testing.T) {
	f := detect(t, baseSeries("M-101", 14, 730))
	if f.Anomaly {
		t.Fatalf("expected no anomaly, got %s/%s: %s", f.Type, f.Severity, f.Reason)
	}
	if f.Shift != nil {
		t.Fatalf("expected no shift, got %+v", f.Shift)
	}
	if f.Quality.AffectedReadings != 0 {
		t.Fatalf("expected no quality issues, got %d", f.Quality.AffectedReadings)
	}
}

func TestDetectUnexplainedLargeIncreaseIsHighRealAnomaly(t *testing.T) {
	s := baseSeries("M-109", 14, 1050)
	s = s.scale(at(11, 14), end(s), 2.1, 0.74)
	unknown := event.Event{ID: 3, MeterID: "M-109", Timestamp: at(11, 14), Type: event.TypeUnknown, Description: "No operational event reported"}

	f := detect(t, s, unknown)

	if !f.Anomaly || f.Type != anomaly.TypeRealAnomaly || f.Severity != anomaly.SeverityHigh {
		t.Fatalf("expected REAL_ANOMALY/HIGH, got %v %s/%s", f.Anomaly, f.Type, f.Severity)
	}
	if f.Shift == nil || f.Shift.Direction != DirectionUp || !f.Shift.Persistent {
		t.Fatalf("expected persistent upward shift, got %+v", f.Shift)
	}
	if !f.Shift.Start.Equal(at(11, 14)) {
		t.Fatalf("shift should start at the change point, got %s", f.Shift.Start)
	}
	if f.VariationPct < 100 {
		t.Fatalf("expected variation > 100%%, got %.1f", f.VariationPct)
	}
	if f.Electrical == nil || !f.Electrical.Significant || f.Electrical.PowerFactorDelta > -0.1 {
		t.Fatalf("expected significant electrical change with power factor drop, got %+v", f.Electrical)
	}
	if f.Confidence < 0.9 {
		t.Fatalf("expected high confidence, got %.2f", f.Confidence)
	}
	if f.RelatedEvent == nil || f.RelatedEvent.Type != event.TypeUnknown {
		t.Fatalf("unknown event should be related but not explanatory, got %+v", f.RelatedEvent)
	}
	if f.Explained {
		t.Fatal("unknown event must not explain the change")
	}
	assertContainsVariable(t, f, "consumption_kwh", "current_a", "power_factor")
	assertHasText(t, f)
}

func TestDetectIncreaseWithOperationalChangeIsExplainableMedium(t *testing.T) {
	s := baseSeries("M-104", 14, 1170)
	s = s.scale(at(10, 0), end(s), 1.47, 0)
	ev := event.Event{ID: 1, MeterID: "M-104", Timestamp: at(10, 0), Type: event.TypeOperationalChange, Description: "New production line activated"}

	f := detect(t, s, ev)

	if !f.Anomaly || f.Type != anomaly.TypeExplainable || f.Severity != anomaly.SeverityMedium {
		t.Fatalf("expected EXPLAINABLE_ANOMALY/MEDIUM, got %s/%s", f.Type, f.Severity)
	}
	if !f.Explained || f.RelatedEvent == nil || f.RelatedEvent.ID != 1 {
		t.Fatalf("expected event 1 to explain the change, got %+v", f.RelatedEvent)
	}
	if f.VariationPct < 40 || f.VariationPct > 55 {
		t.Fatalf("expected variation near +47%%, got %.1f", f.VariationPct)
	}
	assertHasText(t, f)
}

func TestDetectDropDuringScheduledOutageIsLowFalsePositive(t *testing.T) {
	s := baseSeries("M-106", 14, 1350)
	s = s.scale(at(7, 0), at(7, 12), 0.18, 0)
	ev := event.Event{ID: 2, MeterID: "M-106", Timestamp: at(7, 0), Type: event.TypeScheduledOutage, Description: "Scheduled maintenance outage for 12 hours"}

	f := detect(t, s, ev)

	if !f.Anomaly || f.Type != anomaly.TypeFalsePositive || f.Severity != anomaly.SeverityLow {
		t.Fatalf("expected FALSE_POSITIVE/LOW, got %s/%s", f.Type, f.Severity)
	}
	if f.Shift == nil || f.Shift.Direction != DirectionDown || f.Shift.Persistent {
		t.Fatalf("expected transient downward shift, got %+v", f.Shift)
	}
	if f.Shift.Hours < 10 || f.Shift.Hours > 13 {
		t.Fatalf("expected ~12h window, got %d", f.Shift.Hours)
	}
	if f.Type.IsEscalable() {
		t.Fatal("false positive must not be escalated")
	}
	assertHasText(t, f)
}

func TestDetectDropWithoutEventIsNotFalsePositive(t *testing.T) {
	s := baseSeries("M-106", 14, 1350)
	s = s.scale(at(7, 0), at(7, 12), 0.18, 0)

	f := detect(t, s)

	if f.Type == anomaly.TypeFalsePositive {
		t.Fatal("an unexplained drop must not be classified as false positive")
	}
	if f.Type != anomaly.TypeRealAnomaly {
		t.Fatalf("expected REAL_ANOMALY, got %s", f.Type)
	}
}

func TestDetectEventFarFromChangeDoesNotExplainIt(t *testing.T) {
	s := baseSeries("M-104", 14, 1170)
	s = s.scale(at(10, 0), end(s), 1.47, 0)
	ev := event.Event{MeterID: "M-104", Timestamp: at(3, 0), Type: event.TypeOperationalChange}

	f := detect(t, s, ev)

	if f.Explained || f.Type != anomaly.TypeRealAnomaly {
		t.Fatalf("expected unexplained REAL_ANOMALY, got explained=%v %s", f.Explained, f.Type)
	}
}

func TestDetectEventsFromOtherMetersAreIgnored(t *testing.T) {
	s := baseSeries("M-104", 14, 1170)
	s = s.scale(at(10, 0), end(s), 1.47, 0)
	ev := event.Event{MeterID: "M-999", Timestamp: at(10, 0), Type: event.TypeOperationalChange}

	f := detect(t, s, ev)

	if f.Explained {
		t.Fatal("events from other meters must not explain the change")
	}
}

func TestDetectInconsistentElectricalReadingsIsHighDataQuality(t *testing.T) {
	s := baseSeries("M-112", 14, 660)
	s = s.corruptElectrical(at(12, 0))
	ev := event.Event{ID: 4, MeterID: "M-112", Timestamp: at(12, 0), Type: event.TypeDataQuality, Description: "Intermittent readings"}

	f := detect(t, s, ev)

	if !f.Anomaly || f.Type != anomaly.TypeDataQuality || f.Severity != anomaly.SeverityHigh {
		t.Fatalf("expected DATA_QUALITY/HIGH, got %s/%s", f.Type, f.Severity)
	}
	if f.Shift != nil {
		t.Fatalf("consumption is stable, expected no shift, got %+v", f.Shift)
	}
	if f.Quality.VoltageOutOfRange == 0 || f.Quality.VoltageJumps == 0 || f.Quality.PowerFactorJumps == 0 {
		t.Fatalf("expected voltage and power factor issues, got %+v", f.Quality)
	}
	if f.RelatedEvent == nil || f.RelatedEvent.ID != 4 {
		t.Fatalf("expected data quality event to corroborate, got %+v", f.RelatedEvent)
	}
	if math.Abs(f.VariationPct) > 5 {
		t.Fatalf("expected stable consumption, got %.1f%%", f.VariationPct)
	}
	assertContainsVariable(t, f, "voltage_v", "power_factor")
	assertHasText(t, f)
}

func TestDetectDataQualityWithoutEventStillDetected(t *testing.T) {
	s := baseSeries("M-112", 14, 660).corruptElectrical(at(12, 0))
	f := detect(t, s)
	if f.Type != anomaly.TypeDataQuality {
		t.Fatalf("expected DATA_QUALITY, got %s", f.Type)
	}
}

func TestDetectFewQualityGlitchesAreTolerated(t *testing.T) {
	s := baseSeries("M-101", 14, 730)
	s[100].VoltageV = 205
	s[200].PowerFactor = 0.7
	f := detect(t, s)
	if f.Anomaly {
		t.Fatalf("isolated glitches must not raise an anomaly, got %s", f.Type)
	}
}

func TestDetectWithNoReadings(t *testing.T) {
	f := NewDetector(DefaultConfig()).Detect("M-000", nil, nil)
	if f.Anomaly || f.MeterID != "M-000" {
		t.Fatalf("unexpected finding %+v", f)
	}
}

func TestDetectUnsortedReadings(t *testing.T) {
	s := baseSeries("M-109", 14, 1050)
	s = s.scale(at(11, 14), end(s), 2.1, 0.74)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	f := detect(t, s)
	if f.Type != anomaly.TypeRealAnomaly {
		t.Fatalf("expected REAL_ANOMALY regardless of input order, got %s", f.Type)
	}
}

func TestFindingToAnomaly(t *testing.T) {
	s := baseSeries("M-109", 14, 1050)
	s = s.scale(at(11, 14), end(s), 2.1, 0.74)
	f := detect(t, s)

	a := f.ToAnomaly(42)

	if a.AnalysisID != 42 || a.MeterID != "M-109" || a.Type != anomaly.TypeRealAnomaly || a.Severity != anomaly.SeverityHigh {
		t.Fatalf("unexpected anomaly %+v", a)
	}
	if a.Source != anomaly.SourceRules || a.Status != anomaly.StatusOpen {
		t.Fatalf("unexpected source/status %s/%s", a.Source, a.Status)
	}
	if a.WindowStart == nil || !a.WindowStart.Equal(at(11, 14)) {
		t.Fatalf("unexpected window start %v", a.WindowStart)
	}
	if !a.DetectedAt.Equal(at(11, 14)) {
		t.Fatalf("detected_at should be the change point, got %s", a.DetectedAt)
	}
	if len(a.Evidence) == 0 || len(a.AffectedVariables) == 0 {
		t.Fatal("anomaly must carry evidence and affected variables")
	}
}

func assertContainsVariable(t *testing.T, f Finding, vars ...string) {
	t.Helper()
	set := map[string]bool{}
	for _, v := range f.AffectedVariables {
		set[v] = true
	}
	for _, v := range vars {
		if !set[v] {
			t.Fatalf("expected affected variable %s in %v", v, f.AffectedVariables)
		}
	}
}

func assertHasText(t *testing.T, f Finding) {
	t.Helper()
	if f.Reason == "" || f.RecommendedAction == "" {
		t.Fatalf("finding must include reason and recommended action: %+v", f)
	}
	if len(f.Evidence) < 3 {
		t.Fatalf("finding must include evidence, got %d items", len(f.Evidence))
	}
	if f.Confidence <= 0 || f.Confidence > 1 {
		t.Fatalf("confidence out of range: %v", f.Confidence)
	}
}
