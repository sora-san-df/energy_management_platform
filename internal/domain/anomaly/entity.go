package anomaly

import (
	"sort"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
)

type Type string

const (
	TypeRealAnomaly   Type = "REAL_ANOMALY"
	TypeExplainable   Type = "EXPLAINABLE_ANOMALY"
	TypeFalsePositive Type = "FALSE_POSITIVE"
	TypeDataQuality   Type = "DATA_QUALITY"
)

func (t Type) Weight() int {
	switch t {
	case TypeRealAnomaly:
		return 4
	case TypeDataQuality:
		return 3
	case TypeExplainable:
		return 2
	case TypeFalsePositive:
		return 1
	}
	return 0
}

func (t Type) IsEscalable() bool {
	return t == TypeRealAnomaly || t == TypeDataQuality || t == TypeExplainable
}

type Severity string

const (
	SeverityHigh   Severity = "HIGH"
	SeverityMedium Severity = "MEDIUM"
	SeverityLow    Severity = "LOW"
)

func (s Severity) Weight() int {
	switch s {
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	}
	return 0
}

type Status string

const (
	StatusOpen          Status = "OPEN"
	StatusInvestigating Status = "INVESTIGATING"
	StatusResolved      Status = "RESOLVED"
	StatusDismissed     Status = "DISMISSED"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusOpen, StatusInvestigating, StatusResolved, StatusDismissed:
		return true
	}
	return false
}

func (s Status) IsActive() bool {
	return s == "" || s == StatusOpen || s == StatusInvestigating
}

func InitialStatus(t Type) Status {
	if t == TypeFalsePositive {
		return StatusDismissed
	}
	return StatusOpen
}

type Source string

const (
	SourceGemini Source = "gemini"
	SourceRules  Source = "rules"
)

type Evidence struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail,omitempty"`
}

type Anomaly struct {
	ID                int64        `json:"id"`
	AnalysisID        int64        `json:"analysis_id"`
	MeterID           string       `json:"meter_id"`
	DetectedAt        time.Time    `json:"detected_at"`
	Type              Type         `json:"type"`
	Severity          Severity     `json:"severity"`
	Confidence        float64      `json:"confidence"`
	Reason            string       `json:"reason"`
	RecommendedAction string       `json:"recommended_action"`
	Status            Status       `json:"status"`
	Priority          int          `json:"priority"`
	Source            Source       `json:"source"`
	AIError           string       `json:"ai_error,omitempty"`
	AIInsights        []string     `json:"ai_insights"`
	BaselineDailyKWh  float64      `json:"baseline_daily_kwh"`
	CurrentDailyKWh   float64      `json:"current_daily_kwh"`
	VariationPct      float64      `json:"variation_pct"`
	WindowStart       *time.Time   `json:"window_start,omitempty"`
	WindowEnd         *time.Time   `json:"window_end,omitempty"`
	AffectedVariables []string     `json:"affected_variables"`
	Evidence          []Evidence   `json:"evidence"`
	RelatedEvent      *event.Event `json:"related_event,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
}

func (a Anomaly) IsHighPriority() bool {
	return a.Type.IsEscalable() && a.Severity == SeverityHigh
}

func (a Anomaly) score() float64 {
	return float64(a.Severity.Weight())*100 + float64(a.Type.Weight())*10 + a.Confidence
}

func ConfidenceLevel(confidence float64) string {
	switch {
	case confidence >= 0.85:
		return "ALTA"
	case confidence >= 0.65:
		return "MEDIA"
	}
	return "BAJA"
}

func Rank(items []Anomaly) []Anomaly {
	ranked := make([]Anomaly, len(items))
	copy(ranked, items)
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score() > ranked[j].score()
	})
	for i := range ranked {
		ranked[i].Priority = i + 1
	}
	return ranked
}
