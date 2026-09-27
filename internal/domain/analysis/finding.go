package analysis

import (
	"math"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/analytics"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
)

type Finding struct {
	MeterID           string             `json:"meter_id"`
	Anomaly           bool               `json:"anomaly"`
	Type              anomaly.Type       `json:"type,omitempty"`
	Severity          anomaly.Severity   `json:"severity,omitempty"`
	Confidence        float64            `json:"confidence"`
	Reason            string             `json:"reason"`
	RecommendedAction string             `json:"recommended_action"`
	DetectedAt        time.Time          `json:"detected_at"`
	BaselineDailyKWh  float64            `json:"baseline_daily_kwh"`
	CurrentDailyKWh   float64            `json:"current_daily_kwh"`
	VariationPct      float64            `json:"variation_pct"`
	Shift             *Shift             `json:"shift,omitempty"`
	Electrical        *ElectricalChange  `json:"electrical_change,omitempty"`
	Quality           QualityReport      `json:"quality"`
	RelatedEvent      *event.Event       `json:"related_event,omitempty"`
	Explained         bool               `json:"explained"`
	AffectedVariables []string           `json:"affected_variables"`
	Evidence          []anomaly.Evidence `json:"evidence"`
}

func (d *Detector) Classify(meterID string, b Baseline, shift *Shift, change *ElectricalChange, quality QualityReport, related *event.Event) Finding {
	f := Finding{
		MeterID:          meterID,
		BaselineDailyKWh: analytics.Round(b.Stats.BaselineDailyKWh, 1),
		CurrentDailyKWh:  analytics.Round(b.Stats.CurrentDailyKWh, 1),
		VariationPct:     analytics.Round(b.Stats.VariationPct, 1),
		Shift:            shift,
		Electrical:       change,
		Quality:          quality,
		RelatedEvent:     related,
	}
	switch {
	case shift != nil && explains(related, shift):
		f.Explained = true
		if related.Type == event.TypeOperationalChange {
			d.classifyExplainable(&f)
		} else {
			d.classifyFalsePositive(&f)
		}
	case shift != nil:
		d.classifyReal(&f)
	case d.hasQualityProblem(quality):
		d.classifyDataQuality(&f)
	default:
		return f
	}
	f.Anomaly = true
	f.Confidence = analytics.Round(math.Min(f.Confidence, 0.99), 2)
	f.AffectedVariables = d.affectedVariables(f)
	f.Evidence = d.buildEvidence(f)
	f.Reason, f.RecommendedAction = d.narrative(f)
	return f
}

func (d *Detector) classifyReal(f *Finding) {
	f.Type = anomaly.TypeRealAnomaly
	f.DetectedAt = f.Shift.Start
	magnitude := math.Abs(f.Shift.DeviationPct)
	electrical := f.Electrical != nil && f.Electrical.Significant
	f.Severity = anomaly.SeverityMedium
	if magnitude >= d.cfg.HighSeverityPct || electrical {
		f.Severity = anomaly.SeverityHigh
	}
	c := 0.72
	switch {
	case magnitude >= 100:
		c += 0.10
	case magnitude >= d.cfg.HighSeverityPct:
		c += 0.06
	}
	if f.Shift.Persistent {
		c += 0.05
	}
	if electrical {
		c += 0.05
	}
	if f.Shift.FlaggedHours >= 24 {
		c += 0.03
	}
	if f.RelatedEvent != nil && f.RelatedEvent.Type == event.TypeUnknown {
		c += 0.02
	}
	f.Confidence = c
}

func (d *Detector) eventAlignment(f *Finding) float64 {
	if absDuration(f.RelatedEvent.Timestamp.Sub(f.Shift.Start)) <= time.Hour {
		return 0.12
	}
	return 0.06
}

func (d *Detector) classifyExplainable(f *Finding) {
	f.Type = anomaly.TypeExplainable
	f.Severity = anomaly.SeverityMedium
	f.DetectedAt = f.Shift.Start
	c := 0.70 + d.eventAlignment(f)
	if f.Shift.Persistent {
		c += 0.05
	}
	if f.Electrical == nil || !f.Electrical.Significant {
		c += 0.05
	}
	f.Confidence = c
}

func (d *Detector) classifyFalsePositive(f *Finding) {
	f.Type = anomaly.TypeFalsePositive
	f.Severity = anomaly.SeverityLow
	f.DetectedAt = f.Shift.Start
	c := 0.68 + d.eventAlignment(f)
	if !f.Shift.Persistent {
		c += 0.08
	}
	f.Confidence = c
}

func (d *Detector) classifyDataQuality(f *Finding) {
	f.Type = anomaly.TypeDataQuality
	f.Severity = anomaly.SeverityHigh
	f.DetectedAt = *f.Quality.FirstIssueAt
	c := 0.70 + math.Min(0.15, float64(f.Quality.AffectedReadings)/200)
	if f.RelatedEvent != nil && f.RelatedEvent.Type == event.TypeDataQuality {
		c += 0.08
	}
	if math.Abs(f.VariationPct) < 5 {
		c += 0.04
	}
	f.Confidence = c
}

func (d *Detector) affectedVariables(f Finding) []string {
	set := map[string]bool{}
	var out []string
	add := func(v string) {
		if !set[v] {
			set[v] = true
			out = append(out, v)
		}
	}
	if f.Shift != nil {
		add("consumption_kwh")
		if c := f.Electrical; c != nil {
			if math.Abs(c.CurrentChangePct) >= d.cfg.CurrentShiftPct {
				add("current_a")
			}
			if math.Abs(c.PowerFactorDelta) >= d.cfg.PowerFactorShift {
				add("power_factor")
			}
			if math.Abs(c.VoltageChangePct) >= d.cfg.VoltageShiftPct {
				add("voltage_v")
			}
		}
	}
	if f.Type == anomaly.TypeDataQuality {
		q := f.Quality
		if q.VoltageOutOfRange > 0 || q.VoltageJumps > 0 {
			add("voltage_v")
		}
		if q.PowerFactorJumps > 0 {
			add("power_factor")
		}
		if q.EnergyMismatches > 0 {
			add("current_a")
		}
	}
	return out
}

func (f Finding) window() (*time.Time, *time.Time) {
	if f.Shift != nil {
		start, end := f.Shift.Start, f.Shift.End
		return &start, &end
	}
	if f.Quality.FirstIssueAt != nil && f.Quality.LastIssueAt != nil {
		start, end := *f.Quality.FirstIssueAt, f.Quality.LastIssueAt.Add(time.Hour)
		return &start, &end
	}
	return nil, nil
}

func (f Finding) ToAnomaly(analysisID int64) anomaly.Anomaly {
	start, end := f.window()
	return anomaly.Anomaly{
		AnalysisID:        analysisID,
		MeterID:           f.MeterID,
		DetectedAt:        f.DetectedAt,
		Type:              f.Type,
		Severity:          f.Severity,
		Confidence:        f.Confidence,
		Reason:            f.Reason,
		RecommendedAction: f.RecommendedAction,
		Status:            anomaly.InitialStatus(f.Type),
		Source:            anomaly.SourceRules,
		AIInsights:        []string{},
		BaselineDailyKWh:  f.BaselineDailyKWh,
		CurrentDailyKWh:   f.CurrentDailyKWh,
		VariationPct:      f.VariationPct,
		WindowStart:       start,
		WindowEnd:         end,
		AffectedVariables: append([]string{}, f.AffectedVariables...),
		Evidence:          append([]anomaly.Evidence{}, f.Evidence...),
		RelatedEvent:      f.RelatedEvent,
	}
}
