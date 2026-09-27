package anomaly

import (
	"testing"
)

func TestSeverityWeight(t *testing.T) {
	cases := map[Severity]int{SeverityHigh: 3, SeverityMedium: 2, SeverityLow: 1, Severity("OTHER"): 0}
	for sev, want := range cases {
		if got := sev.Weight(); got != want {
			t.Errorf("%s weight = %d, want %d", sev, got, want)
		}
	}
}

func TestTypeWeightPrioritisesRealAnomaliesOverDataQuality(t *testing.T) {
	if TypeRealAnomaly.Weight() <= TypeDataQuality.Weight() {
		t.Fatal("real anomaly must weigh more than data quality")
	}
	if TypeDataQuality.Weight() <= TypeExplainable.Weight() {
		t.Fatal("data quality must weigh more than explainable anomaly")
	}
	if TypeExplainable.Weight() <= TypeFalsePositive.Weight() {
		t.Fatal("explainable anomaly must weigh more than false positive")
	}
}

func TestTypeIsEscalable(t *testing.T) {
	if TypeFalsePositive.IsEscalable() {
		t.Fatal("false positives must not be escalated")
	}
	for _, typ := range []Type{TypeRealAnomaly, TypeDataQuality, TypeExplainable} {
		if !typ.IsEscalable() {
			t.Fatalf("%s should be escalable", typ)
		}
	}
}

func TestConfidenceLevel(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{0.96, "ALTA"},
		{0.85, "ALTA"},
		{0.84, "MEDIA"},
		{0.65, "MEDIA"},
		{0.40, "BAJA"},
	}
	for _, c := range cases {
		if got := ConfidenceLevel(c.value); got != c.want {
			t.Errorf("ConfidenceLevel(%v) = %s, want %s", c.value, got, c.want)
		}
	}
}

func TestIsHighPriority(t *testing.T) {
	cases := []struct {
		a    Anomaly
		want bool
	}{
		{Anomaly{Type: TypeRealAnomaly, Severity: SeverityHigh}, true},
		{Anomaly{Type: TypeDataQuality, Severity: SeverityHigh}, true},
		{Anomaly{Type: TypeExplainable, Severity: SeverityMedium}, false},
		{Anomaly{Type: TypeFalsePositive, Severity: SeverityHigh}, false},
	}
	for _, c := range cases {
		if got := c.a.IsHighPriority(); got != c.want {
			t.Errorf("%s/%s high priority = %v, want %v", c.a.Type, c.a.Severity, got, c.want)
		}
	}
}

func TestRankOrdersByPriorityAndAssignsPositions(t *testing.T) {
	items := []Anomaly{
		{ID: 1, MeterID: "M-106", Type: TypeFalsePositive, Severity: SeverityLow, Confidence: 0.9},
		{ID: 2, MeterID: "M-112", Type: TypeDataQuality, Severity: SeverityHigh, Confidence: 0.95},
		{ID: 3, MeterID: "M-104", Type: TypeExplainable, Severity: SeverityMedium, Confidence: 0.9},
		{ID: 4, MeterID: "M-109", Type: TypeRealAnomaly, Severity: SeverityHigh, Confidence: 0.9},
	}

	ranked := Rank(items)

	wantOrder := []string{"M-109", "M-112", "M-104", "M-106"}
	for i, meterID := range wantOrder {
		if ranked[i].MeterID != meterID {
			t.Fatalf("position %d = %s, want %s", i+1, ranked[i].MeterID, meterID)
		}
		if ranked[i].Priority != i+1 {
			t.Fatalf("%s priority = %d, want %d", meterID, ranked[i].Priority, i+1)
		}
	}
	if items[0].Priority != 0 {
		t.Fatal("Rank must not mutate the input slice")
	}
}

func TestRankUsesConfidenceAsTieBreaker(t *testing.T) {
	ranked := Rank([]Anomaly{
		{MeterID: "A", Type: TypeRealAnomaly, Severity: SeverityHigh, Confidence: 0.70},
		{MeterID: "B", Type: TypeRealAnomaly, Severity: SeverityHigh, Confidence: 0.95},
	})
	if ranked[0].MeterID != "B" {
		t.Fatalf("expected higher confidence first, got %s", ranked[0].MeterID)
	}
}

func TestStatusIsValid(t *testing.T) {
	for _, s := range []Status{StatusOpen, StatusInvestigating, StatusResolved, StatusDismissed} {
		if !s.IsValid() {
			t.Errorf("%s should be valid", s)
		}
	}
	if Status("WHATEVER").IsValid() {
		t.Error("unknown status must be invalid")
	}
}

func TestInitialStatus(t *testing.T) {
	if got := InitialStatus(TypeFalsePositive); got != StatusDismissed {
		t.Errorf("false positive initial status = %s, want %s", got, StatusDismissed)
	}
	if got := InitialStatus(TypeRealAnomaly); got != StatusOpen {
		t.Errorf("real anomaly initial status = %s, want %s", got, StatusOpen)
	}
}
