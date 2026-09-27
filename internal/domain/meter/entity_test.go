package meter

import (
	"testing"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
)

func TestStatusFromAnomalies(t *testing.T) {
	cases := []struct {
		name      string
		anomalies []anomaly.Anomaly
		want      Status
	}{
		{"no anomalies", nil, StatusOK},
		{"false positive only", []anomaly.Anomaly{{Type: anomaly.TypeFalsePositive, Severity: anomaly.SeverityLow}}, StatusOK},
		{"explainable medium", []anomaly.Anomaly{{Type: anomaly.TypeExplainable, Severity: anomaly.SeverityMedium}}, StatusAlert},
		{"data quality high", []anomaly.Anomaly{{Type: anomaly.TypeDataQuality, Severity: anomaly.SeverityHigh}}, StatusAlert},
		{"real anomaly high", []anomaly.Anomaly{{Type: anomaly.TypeRealAnomaly, Severity: anomaly.SeverityHigh}}, StatusCritical},
		{"real anomaly medium", []anomaly.Anomaly{{Type: anomaly.TypeRealAnomaly, Severity: anomaly.SeverityMedium}}, StatusAlert},
		{"resolved real anomaly", []anomaly.Anomaly{{Type: anomaly.TypeRealAnomaly, Severity: anomaly.SeverityHigh, Status: anomaly.StatusResolved}}, StatusOK},
		{"worst wins", []anomaly.Anomaly{
			{Type: anomaly.TypeExplainable, Severity: anomaly.SeverityMedium},
			{Type: anomaly.TypeRealAnomaly, Severity: anomaly.SeverityHigh},
		}, StatusCritical},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StatusFromAnomalies(c.anomalies); got != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}

func TestStatusMatchesFilter(t *testing.T) {
	cases := []struct {
		status Status
		filter string
		want   bool
	}{
		{StatusOK, "", true},
		{StatusOK, "all", true},
		{StatusOK, "normal", true},
		{StatusAlert, "normal", false},
		{StatusAlert, "alert", true},
		{StatusCritical, "alert", false},
		{StatusCritical, "critical", true},
		{StatusPending, "normal", false},
	}
	for _, c := range cases {
		if got := c.status.MatchesFilter(c.filter); got != c.want {
			t.Errorf("%s.MatchesFilter(%q) = %v, want %v", c.status, c.filter, got, c.want)
		}
	}
}
