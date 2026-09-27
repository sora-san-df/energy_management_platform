package analytics

import (
	"math"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

func approx(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func hourly(meterID string, start time.Time, values ...float64) []reading.Reading {
	out := make([]reading.Reading, len(values))
	for i, v := range values {
		out[i] = reading.Reading{MeterID: meterID, Timestamp: start.Add(time.Duration(i) * time.Hour), ConsumptionKWh: v, VoltageV: 220, CurrentA: v * 1000 / 220 / 0.95, PowerFactor: 0.95}
	}
	return out
}

func flatDays(days int, perHour float64) []float64 {
	out := make([]float64, days*24)
	for i := range out {
		out[i] = perHour
	}
	return out
}

var start = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func TestMedian(t *testing.T) {
	cases := []struct {
		in   []float64
		want float64
	}{
		{nil, 0},
		{[]float64{5}, 5},
		{[]float64{3, 1, 2}, 2},
		{[]float64{4, 1, 3, 2}, 2.5},
	}
	for _, c := range cases {
		if got := Median(c.in); got != c.want {
			t.Errorf("Median(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMedianDoesNotMutateInput(t *testing.T) {
	in := []float64{3, 1, 2}
	Median(in)
	if in[0] != 3 || in[1] != 1 || in[2] != 2 {
		t.Fatalf("input mutated: %v", in)
	}
}

func TestMean(t *testing.T) {
	if got := Mean(nil); got != 0 {
		t.Fatalf("Mean(nil) = %v", got)
	}
	if got := Mean([]float64{1, 2, 3, 4}); got != 2.5 {
		t.Fatalf("Mean = %v, want 2.5", got)
	}
}

func TestPercentChange(t *testing.T) {
	if got := PercentChange(100, 203.7); !approx(got, 103.7, 1e-9) {
		t.Fatalf("PercentChange = %v", got)
	}
	if got := PercentChange(0, 10); got != 0 {
		t.Fatalf("PercentChange with zero baseline = %v, want 0", got)
	}
}

func TestRound(t *testing.T) {
	if got := Round(103.66666, 1); got != 103.7 {
		t.Fatalf("Round = %v", got)
	}
	if got := Round(0.95555, 2); got != 0.96 {
		t.Fatalf("Round = %v", got)
	}
}

func TestDailyTotals(t *testing.T) {
	values := append(flatDays(1, 10), flatDays(1, 20)...)
	totals := DailyTotals(hourly("M-1", start, values...))

	if len(totals) != 2 {
		t.Fatalf("expected 2 days, got %d", len(totals))
	}
	if totals[0].KWh != 240 || totals[1].KWh != 480 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
	if !totals[0].Date.Equal(start) || totals[0].Hours != 24 {
		t.Fatalf("unexpected first day: %+v", totals[0])
	}
}

func TestDailyTotalsAreSortedEvenWithUnsortedInput(t *testing.T) {
	rs := hourly("M-1", start, append(flatDays(1, 1), flatDays(1, 2)...)...)
	rs[0], rs[len(rs)-1] = rs[len(rs)-1], rs[0]
	totals := DailyTotals(rs)
	if !totals[0].Date.Before(totals[1].Date) {
		t.Fatal("daily totals must be sorted by date")
	}
}

func TestHourlyProfileUsesMedianPerHourOfDay(t *testing.T) {
	values := flatDays(5, 10)
	for i := 0; i < 24; i++ {
		values[4*24+i] = 100
	}
	values[3] = 4
	profile := HourlyProfile(hourly("M-1", start, values...))

	for h := 0; h < 24; h++ {
		if profile[h] != 10 {
			t.Fatalf("hour %d expected 10, got %v", h, profile[h])
		}
	}
}

func TestConsumptionStats(t *testing.T) {
	values := append(flatDays(10, 43.75), flatDays(2, 91.75)...)
	stats := ComputeConsumptionStats(hourly("M-109", start, values...))

	if !approx(stats.TotalKWh, 10*1050+2*2202, 1e-6) {
		t.Fatalf("total = %v", stats.TotalKWh)
	}
	if !approx(stats.BaselineDailyKWh, 1050, 1e-6) {
		t.Fatalf("baseline = %v", stats.BaselineDailyKWh)
	}
	if !approx(stats.CurrentDailyKWh, 2202, 1e-6) {
		t.Fatalf("current = %v", stats.CurrentDailyKWh)
	}
	if !approx(stats.VariationPct, 109.7, 0.05) {
		t.Fatalf("variation = %v", stats.VariationPct)
	}
	if stats.Days != 12 {
		t.Fatalf("days = %d", stats.Days)
	}
}

func TestConsumptionStatsEmpty(t *testing.T) {
	stats := ComputeConsumptionStats(nil)
	if stats != (ConsumptionStats{}) {
		t.Fatalf("expected zero stats, got %+v", stats)
	}
}

func TestConsumptionStatsIgnoresIncompleteLastDay(t *testing.T) {
	values := append(flatDays(3, 10), 50, 50)
	stats := ComputeConsumptionStats(hourly("M-1", start, values...))
	if stats.CurrentDailyKWh != 240 {
		t.Fatalf("current should use last complete day, got %v", stats.CurrentDailyKWh)
	}
}

func TestElectricalAverages(t *testing.T) {
	rs := []reading.Reading{
		{VoltageV: 220, CurrentA: 100, PowerFactor: 0.9, ConsumptionKWh: 20},
		{VoltageV: 222, CurrentA: 120, PowerFactor: 0.8, ConsumptionKWh: 22},
	}
	avg := Averages(rs)
	if avg.VoltageV != 221 || avg.CurrentA != 110 || !approx(avg.PowerFactor, 0.85, 1e-9) || avg.ConsumptionKWh != 21 {
		t.Fatalf("unexpected averages %+v", avg)
	}
	if Averages(nil) != (ElectricalAverages{}) {
		t.Fatal("empty averages must be zero")
	}
}

func TestApparentEnergyRatio(t *testing.T) {
	r := reading.Reading{ConsumptionKWh: 22, VoltageV: 220, CurrentA: 100, PowerFactor: 1}
	if got := EnergyCoherenceRatio(r); !approx(got, 1.0, 1e-9) {
		t.Fatalf("ratio = %v", got)
	}
	if got := EnergyCoherenceRatio(reading.Reading{ConsumptionKWh: 5}); !math.IsInf(got, 1) {
		t.Fatalf("zero electrical power must return +Inf, got %v", got)
	}
}
