package meter

import (
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/analytics"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

type AnomalyRef struct {
	ID         int64            `json:"id"`
	Type       anomaly.Type     `json:"type"`
	Severity   anomaly.Severity `json:"severity"`
	Confidence float64          `json:"confidence"`
	Status     anomaly.Status   `json:"status"`
}

type Overview struct {
	Meter
	TotalKWh         float64     `json:"total_kwh"`
	CurrentDailyKWh  float64     `json:"current_daily_kwh"`
	BaselineDailyKWh float64     `json:"baseline_daily_kwh"`
	VariationPct     float64     `json:"variation_pct"`
	ReadingsCount    int         `json:"readings_count"`
	LastReadingAt    *time.Time  `json:"last_reading_at,omitempty"`
	Anomaly          *AnomalyRef `json:"anomaly,omitempty"`
}

type DailyPoint struct {
	Date        time.Time `json:"date"`
	KWh         float64   `json:"kwh"`
	BaselineKWh float64   `json:"baseline_kwh"`
}

type Detail struct {
	Overview
	Electrical analytics.ElectricalAverages `json:"electrical"`
	Latest     *reading.Reading             `json:"latest_reading,omitempty"`
	Daily      []DailyPoint                 `json:"daily"`
	Events     []event.Event                `json:"events"`
	Anomalies  []anomaly.Anomaly            `json:"anomalies"`
}

type ListQuery struct {
	Status string
	Search string
	Sort   string
	Order  string
}

type ReadingsQuery struct {
	From        time.Time
	To          time.Time
	Granularity string
}

type SeriesPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	ConsumptionKWh float64   `json:"consumption_kwh"`
	ExpectedKWh    float64   `json:"expected_kwh"`
	VoltageV       float64   `json:"voltage_v"`
	CurrentA       float64   `json:"current_a"`
	PowerFactor    float64   `json:"power_factor"`
	Status         string    `json:"status,omitempty"`
}

type ReadingsResult struct {
	MeterID     string        `json:"meter_id"`
	Granularity string        `json:"granularity"`
	Points      []SeriesPoint `json:"points"`
}
