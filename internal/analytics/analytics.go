package analytics

import (
	"math"
	"sort"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

type DailyPoint struct {
	Date  time.Time `json:"date"`
	KWh   float64   `json:"kwh"`
	Hours int       `json:"hours"`
}

type ConsumptionStats struct {
	TotalKWh         float64 `json:"total_kwh"`
	BaselineDailyKWh float64 `json:"baseline_daily_kwh"`
	CurrentDailyKWh  float64 `json:"current_daily_kwh"`
	VariationPct     float64 `json:"variation_pct"`
	Days             int     `json:"days"`
}

type ElectricalAverages struct {
	ConsumptionKWh float64 `json:"consumption_kwh"`
	VoltageV       float64 `json:"voltage_v"`
	CurrentA       float64 `json:"current_a"`
	PowerFactor    float64 `json:"power_factor"`
}

func Median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func PercentChange(baseline, current float64) float64 {
	if baseline == 0 {
		return 0
	}
	return (current - baseline) / baseline * 100
}

func Round(value float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(value*factor) / factor
}

func DayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func DailyTotals(readings []reading.Reading) []DailyPoint {
	index := map[time.Time]*DailyPoint{}
	for _, r := range readings {
		day := DayOf(r.Timestamp)
		point, ok := index[day]
		if !ok {
			point = &DailyPoint{Date: day}
			index[day] = point
		}
		point.KWh += r.ConsumptionKWh
		point.Hours++
	}
	out := make([]DailyPoint, 0, len(index))
	for _, p := range index {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

func HourlyProfile(readings []reading.Reading) [24]float64 {
	var buckets [24][]float64
	for _, r := range readings {
		h := r.Timestamp.Hour()
		buckets[h] = append(buckets[h], r.ConsumptionKWh)
	}
	var profile [24]float64
	for h := range buckets {
		profile[h] = Median(buckets[h])
	}
	return profile
}

func ComputeConsumptionStats(readings []reading.Reading) ConsumptionStats {
	if len(readings) == 0 {
		return ConsumptionStats{}
	}
	var total float64
	for _, r := range readings {
		total += r.ConsumptionKWh
	}
	var complete []float64
	for _, d := range DailyTotals(readings) {
		if d.Hours == 24 {
			complete = append(complete, d.KWh)
		}
	}
	stats := ConsumptionStats{TotalKWh: total, Days: len(complete)}
	if len(complete) == 0 {
		return stats
	}
	stats.BaselineDailyKWh = Median(complete)
	stats.CurrentDailyKWh = complete[len(complete)-1]
	stats.VariationPct = PercentChange(stats.BaselineDailyKWh, stats.CurrentDailyKWh)
	return stats
}

func Averages(readings []reading.Reading) ElectricalAverages {
	if len(readings) == 0 {
		return ElectricalAverages{}
	}
	var avg ElectricalAverages
	for _, r := range readings {
		avg.ConsumptionKWh += r.ConsumptionKWh
		avg.VoltageV += r.VoltageV
		avg.CurrentA += r.CurrentA
		avg.PowerFactor += r.PowerFactor
	}
	n := float64(len(readings))
	avg.ConsumptionKWh /= n
	avg.VoltageV /= n
	avg.CurrentA /= n
	avg.PowerFactor /= n
	return avg
}

func EnergyCoherenceRatio(r reading.Reading) float64 {
	electrical := r.VoltageV * r.CurrentA * r.PowerFactor / 1000
	if electrical <= 0 {
		return math.Inf(1)
	}
	return r.ConsumptionKWh / electrical
}
