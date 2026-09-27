package analysis

import (
	"math"
	"sort"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/analytics"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

type Direction string

const (
	DirectionUp   Direction = "UP"
	DirectionDown Direction = "DOWN"
)

type Baseline struct {
	Profile [24]float64                `json:"hourly_profile"`
	Stats   analytics.ConsumptionStats `json:"stats"`
}

type Shift struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Hours        int       `json:"hours"`
	FlaggedHours int       `json:"flagged_hours"`
	Direction    Direction `json:"direction"`
	DeviationPct float64   `json:"deviation_pct"`
	ExpectedKWh  float64   `json:"expected_kwh_per_hour"`
	ObservedKWh  float64   `json:"observed_kwh_per_hour"`
	Persistent   bool      `json:"persistent"`
}

type ElectricalChange struct {
	Before               analytics.ElectricalAverages `json:"before"`
	During               analytics.ElectricalAverages `json:"during"`
	ConsumptionChangePct float64                      `json:"consumption_change_pct"`
	CurrentChangePct     float64                      `json:"current_change_pct"`
	VoltageChangePct     float64                      `json:"voltage_change_pct"`
	PowerFactorDelta     float64                      `json:"power_factor_delta"`
	Significant          bool                         `json:"significant"`
}

type IssueKind string

const (
	IssueVoltageOutOfRange IssueKind = "VOLTAGE_OUT_OF_RANGE"
	IssueVoltageJump       IssueKind = "VOLTAGE_JUMP"
	IssuePowerFactorJump   IssueKind = "POWER_FACTOR_JUMP"
	IssueEnergyMismatch    IssueKind = "ENERGY_MISMATCH"
)

type QualityIssue struct {
	Timestamp      time.Time   `json:"timestamp"`
	Kinds          []IssueKind `json:"kinds"`
	VoltageV       float64     `json:"voltage_v"`
	CurrentA       float64     `json:"current_a"`
	PowerFactor    float64     `json:"power_factor"`
	ConsumptionKWh float64     `json:"consumption_kwh"`
}

type QualityReport struct {
	AffectedReadings  int            `json:"affected_readings"`
	VoltageOutOfRange int            `json:"voltage_out_of_range"`
	VoltageJumps      int            `json:"voltage_jumps"`
	PowerFactorJumps  int            `json:"power_factor_jumps"`
	EnergyMismatches  int            `json:"energy_mismatches"`
	FirstIssueAt      *time.Time     `json:"first_issue_at,omitempty"`
	LastIssueAt       *time.Time     `json:"last_issue_at,omitempty"`
	VoltageMin        float64        `json:"voltage_min"`
	VoltageMax        float64        `json:"voltage_max"`
	PowerFactorMin    float64        `json:"power_factor_min"`
	PowerFactorMax    float64        `json:"power_factor_max"`
	Samples           []QualityIssue `json:"samples"`
}

const maxQualitySamples = 8

type Detector struct {
	cfg Config
}

func NewDetector(cfg Config) *Detector {
	return &Detector{cfg: cfg}
}

func (d *Detector) Config() Config {
	return d.cfg
}

func SortReadings(readings []reading.Reading) []reading.Reading {
	out := append([]reading.Reading(nil), readings...)
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

func (d *Detector) Detect(meterID string, readings []reading.Reading, events []event.Event) Finding {
	rs := SortReadings(readings)
	if len(rs) == 0 {
		return Finding{MeterID: meterID}
	}
	baseline := d.Baseline(rs)
	shift := d.DetectShift(rs, baseline)
	change := d.Correlate(rs, shift)
	quality := d.CheckQuality(rs)
	related := d.MatchEvent(meterID, events, shift, quality)
	return d.Classify(meterID, baseline, shift, change, quality, related)
}

func (d *Detector) Baseline(rs []reading.Reading) Baseline {
	return Baseline{Profile: analytics.HourlyProfile(rs), Stats: analytics.ComputeConsumptionStats(rs)}
}

type segment struct {
	startIdx, endIdx int
	direction        Direction
	flagged          int
	devSum           float64
	expectedSum      float64
	observedSum      float64
}

func (s segment) meanDeviation() float64 {
	return s.devSum / float64(s.flagged)
}

func (s segment) score() float64 {
	return float64(s.flagged) * math.Abs(s.meanDeviation())
}

func (d *Detector) DetectShift(rs []reading.Reading, b Baseline) *Shift {
	var segments []segment
	var current *segment
	gap := 0
	flush := func() {
		if current != nil {
			segments = append(segments, *current)
			current = nil
		}
		gap = 0
	}
	for i, r := range rs {
		expected := b.Profile[r.Timestamp.Hour()]
		var direction Direction
		var dev float64
		if expected > 0 {
			dev = (r.ConsumptionKWh - expected) / expected
			switch {
			case dev >= d.cfg.DeviationThreshold:
				direction = DirectionUp
			case dev <= -d.cfg.DeviationThreshold:
				direction = DirectionDown
			}
		}
		if direction == "" {
			if current != nil {
				gap++
				if gap > d.cfg.MaxGapHours {
					flush()
				}
			}
			continue
		}
		if current == nil || current.direction != direction {
			flush()
			current = &segment{startIdx: i, direction: direction}
		}
		current.endIdx = i
		current.flagged++
		current.devSum += dev
		current.expectedSum += expected
		current.observedSum += r.ConsumptionKWh
		gap = 0
	}
	flush()

	var best *segment
	for i := range segments {
		s := segments[i]
		if s.flagged < d.cfg.MinShiftHours {
			continue
		}
		if best == nil || s.score() > best.score() {
			best = &segments[i]
		}
	}
	if best == nil {
		return nil
	}
	start := rs[best.startIdx].Timestamp
	endTs := rs[best.endIdx].Timestamp.Add(time.Hour)
	last := rs[len(rs)-1].Timestamp
	return &Shift{
		Start:        start,
		End:          endTs,
		Hours:        int(endTs.Sub(start) / time.Hour),
		FlaggedHours: best.flagged,
		Direction:    best.direction,
		DeviationPct: best.meanDeviation() * 100,
		ExpectedKWh:  best.expectedSum / float64(best.flagged),
		ObservedKWh:  best.observedSum / float64(best.flagged),
		Persistent:   !rs[best.endIdx].Timestamp.Before(last.Add(-time.Duration(d.cfg.PersistentTailHours) * time.Hour)),
	}
}

func (d *Detector) Correlate(rs []reading.Reading, shift *Shift) *ElectricalChange {
	if shift == nil {
		return nil
	}
	windowStart := shift.Start.Add(-time.Duration(d.cfg.ComparisonWindowHours) * time.Hour)
	var before, during, outside []reading.Reading
	for _, r := range rs {
		inShift := !r.Timestamp.Before(shift.Start) && r.Timestamp.Before(shift.End)
		switch {
		case inShift:
			during = append(during, r)
		case !r.Timestamp.Before(windowStart) && r.Timestamp.Before(shift.Start):
			before = append(before, r)
			outside = append(outside, r)
		default:
			outside = append(outside, r)
		}
	}
	if len(before) == 0 {
		before = outside
	}
	change := &ElectricalChange{Before: analytics.Averages(before), During: analytics.Averages(during)}
	change.ConsumptionChangePct = analytics.PercentChange(change.Before.ConsumptionKWh, change.During.ConsumptionKWh)
	change.CurrentChangePct = analytics.PercentChange(change.Before.CurrentA, change.During.CurrentA)
	change.VoltageChangePct = analytics.PercentChange(change.Before.VoltageV, change.During.VoltageV)
	change.PowerFactorDelta = change.During.PowerFactor - change.Before.PowerFactor
	change.Significant = math.Abs(change.PowerFactorDelta) >= d.cfg.PowerFactorShift || math.Abs(change.VoltageChangePct) >= d.cfg.VoltageShiftPct
	return change
}

func (d *Detector) CheckQuality(rs []reading.Reading) QualityReport {
	report := QualityReport{}
	if len(rs) == 0 {
		return report
	}
	low := d.cfg.NominalVoltage * (1 - d.cfg.VoltageTolerancePct/100)
	high := d.cfg.NominalVoltage * (1 + d.cfg.VoltageTolerancePct/100)
	report.VoltageMin, report.VoltageMax = rs[0].VoltageV, rs[0].VoltageV
	report.PowerFactorMin, report.PowerFactorMax = rs[0].PowerFactor, rs[0].PowerFactor
	for i, r := range rs {
		report.VoltageMin = math.Min(report.VoltageMin, r.VoltageV)
		report.VoltageMax = math.Max(report.VoltageMax, r.VoltageV)
		report.PowerFactorMin = math.Min(report.PowerFactorMin, r.PowerFactor)
		report.PowerFactorMax = math.Max(report.PowerFactorMax, r.PowerFactor)

		var kinds []IssueKind
		if r.VoltageV < low || r.VoltageV > high {
			kinds = append(kinds, IssueVoltageOutOfRange)
			report.VoltageOutOfRange++
		}
		if i > 0 {
			prev := rs[i-1]
			if math.Abs(r.VoltageV-prev.VoltageV) > d.cfg.VoltageJumpV {
				kinds = append(kinds, IssueVoltageJump)
				report.VoltageJumps++
			}
			if math.Abs(r.PowerFactor-prev.PowerFactor) > d.cfg.PowerFactorJump {
				kinds = append(kinds, IssuePowerFactorJump)
				report.PowerFactorJumps++
			}
		}
		ratio := analytics.EnergyCoherenceRatio(r)
		if ratio < d.cfg.EnergyRatioMin || ratio > d.cfg.EnergyRatioMax {
			kinds = append(kinds, IssueEnergyMismatch)
			report.EnergyMismatches++
		}
		if len(kinds) == 0 {
			continue
		}
		report.AffectedReadings++
		ts := r.Timestamp
		if report.FirstIssueAt == nil {
			report.FirstIssueAt = &ts
		}
		report.LastIssueAt = &ts
		if len(report.Samples) < maxQualitySamples {
			report.Samples = append(report.Samples, QualityIssue{
				Timestamp:      ts,
				Kinds:          kinds,
				VoltageV:       r.VoltageV,
				CurrentA:       r.CurrentA,
				PowerFactor:    r.PowerFactor,
				ConsumptionKWh: r.ConsumptionKWh,
			})
		}
	}
	return report
}

func (d *Detector) hasQualityProblem(q QualityReport) bool {
	return q.AffectedReadings >= d.cfg.MinQualityReadings
}

func (d *Detector) MatchEvent(meterID string, events []event.Event, shift *Shift, quality QualityReport) *event.Event {
	var anchor time.Time
	var from, to time.Time
	switch {
	case shift != nil:
		anchor = shift.Start
		from, to = anchor.Add(-d.cfg.EventMatchWindow), anchor.Add(d.cfg.EventMatchWindow)
	case d.hasQualityProblem(quality):
		anchor = *quality.FirstIssueAt
		from, to = anchor.Add(-d.cfg.QualityEventWindow), quality.LastIssueAt.Add(d.cfg.QualityEventWindow)
	default:
		return nil
	}
	var best *event.Event
	for i := range events {
		e := events[i]
		if e.MeterID != meterID || e.Timestamp.Before(from) || e.Timestamp.After(to) {
			continue
		}
		if best == nil || absDuration(e.Timestamp.Sub(anchor)) < absDuration(best.Timestamp.Sub(anchor)) {
			match := e
			best = &match
		}
	}
	return best
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func explains(e *event.Event, shift *Shift) bool {
	if e == nil || shift == nil {
		return false
	}
	switch e.Type {
	case event.TypeOperationalChange:
		return true
	case event.TypeScheduledOutage:
		return shift.Direction == DirectionDown && !shift.Persistent
	}
	return false
}
