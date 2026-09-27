package dashboard

import (
	"context"
	"errors"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/analytics"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/analysis"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

const topAnomalies = 5

type MeterLister interface {
	List(ctx context.Context, q meter.ListQuery) ([]meter.Overview, error)
}

type ReadingLister interface {
	ListAll(ctx context.Context) ([]reading.Reading, error)
}

type AnomalyLister interface {
	List(ctx context.Context, q anomaly.ListQuery) ([]anomaly.Anomaly, error)
}

type RunReader interface {
	Latest(ctx context.Context) (analysis.Run, error)
}

type DailyPoint struct {
	Date        time.Time `json:"date"`
	KWh         float64   `json:"kwh"`
	BaselineKWh float64   `json:"baseline_kwh"`
}

type Summary struct {
	MetersTotal         int                  `json:"meters_total"`
	MetersByStatus      map[meter.Status]int `json:"meters_by_status"`
	ReadingsTotal       int                  `json:"readings_total"`
	TotalConsumptionKWh float64              `json:"total_consumption_kwh"`
	BaselineDailyKWh    float64              `json:"baseline_daily_kwh"`
	CurrentDailyKWh     float64              `json:"current_daily_kwh"`
	VariationPct        float64              `json:"variation_pct"`
	PeriodStart         *time.Time           `json:"period_start,omitempty"`
	PeriodEnd           *time.Time           `json:"period_end,omitempty"`
	AnomaliesDetected   int                  `json:"anomalies_detected"`
	HighPriority        int                  `json:"high_priority"`
	AvgConfidence       float64              `json:"avg_confidence"`
	ConfidenceLevel     string               `json:"confidence_level"`
	AnomaliesByType     map[anomaly.Type]int `json:"anomalies_by_type"`
	TopAnomalies        []anomaly.Anomaly    `json:"top_anomalies"`
	DailyConsumption    []DailyPoint         `json:"daily_consumption"`
	LastAnalysis        *analysis.Run        `json:"last_analysis"`
}

type Service struct {
	meters    MeterLister
	readings  ReadingLister
	anomalies AnomalyLister
	runs      RunReader
}

func NewService(meters MeterLister, readings ReadingLister, anomalies AnomalyLister, runs RunReader) *Service {
	return &Service{meters: meters, readings: readings, anomalies: anomalies, runs: runs}
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	meters, err := s.meters.List(ctx, meter.ListQuery{})
	if err != nil {
		return Summary{}, err
	}
	readings, err := s.readings.ListAll(ctx)
	if err != nil {
		return Summary{}, err
	}
	anomalies, err := s.anomalies.List(ctx, anomaly.ListQuery{})
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{
		MetersTotal:     len(meters),
		MetersByStatus:  map[meter.Status]int{},
		ReadingsTotal:   len(readings),
		AnomaliesByType: map[anomaly.Type]int{},
		TopAnomalies:    []anomaly.Anomaly{},
	}

	for _, m := range meters {
		sum.MetersByStatus[m.Status]++
		sum.BaselineDailyKWh += m.BaselineDailyKWh
		sum.CurrentDailyKWh += m.CurrentDailyKWh
	}
	sum.BaselineDailyKWh = analytics.Round(sum.BaselineDailyKWh, 1)
	sum.CurrentDailyKWh = analytics.Round(sum.CurrentDailyKWh, 1)
	sum.VariationPct = analytics.Round(analytics.PercentChange(sum.BaselineDailyKWh, sum.CurrentDailyKWh), 1)

	var total float64
	for _, r := range readings {
		total += r.ConsumptionKWh
		ts := r.Timestamp
		if sum.PeriodStart == nil || ts.Before(*sum.PeriodStart) {
			sum.PeriodStart = &ts
		}
		if sum.PeriodEnd == nil || ts.After(*sum.PeriodEnd) {
			sum.PeriodEnd = &ts
		}
	}
	sum.TotalConsumptionKWh = analytics.Round(total, 1)
	for _, d := range analytics.DailyTotals(readings) {
		sum.DailyConsumption = append(sum.DailyConsumption, DailyPoint{Date: d.Date, KWh: analytics.Round(d.KWh, 1), BaselineKWh: sum.BaselineDailyKWh})
	}
	if sum.DailyConsumption == nil {
		sum.DailyConsumption = []DailyPoint{}
	}

	var confidences []float64
	for _, a := range anomalies {
		sum.AnomaliesDetected++
		sum.AnomaliesByType[a.Type]++
		if a.IsHighPriority() {
			sum.HighPriority++
		}
		confidences = append(confidences, a.Confidence)
	}
	sum.AvgConfidence = analytics.Round(analytics.Mean(confidences), 2)
	if len(confidences) > 0 {
		sum.ConfidenceLevel = anomaly.ConfidenceLevel(sum.AvgConfidence)
	}
	ranked := anomaly.Rank(anomalies)
	if len(ranked) > topAnomalies {
		ranked = ranked[:topAnomalies]
	}
	sum.TopAnomalies = append(sum.TopAnomalies, ranked...)

	run, err := s.runs.Latest(ctx)
	switch {
	case errors.Is(err, analysis.ErrRunNotFound):
	case err != nil:
		return Summary{}, err
	default:
		sum.LastAnalysis = &run
	}
	return sum, nil
}
