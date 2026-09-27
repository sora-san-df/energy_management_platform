package meter

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sora-san-df/ai-energy-management-platform/internal/analytics"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

var ErrInvalidQuery = errors.New("invalid query")

var validFilters = map[string]bool{"": true, "all": true, "normal": true, "alert": true, "critical": true, "pending": true}

var validSorts = map[string]bool{"": true, "meter_id": true, "consumption": true, "variation": true, "severity": true}

type Service struct {
	meters    Repository
	readings  ReadingSource
	anomalies AnomalySource
	events    EventSource
}

func NewService(meters Repository, readings ReadingSource, anomalies AnomalySource, events EventSource) *Service {
	return &Service{meters: meters, readings: readings, anomalies: anomalies, events: events}
}

func buildOverview(m Meter, rs []reading.Reading, anomalies []anomaly.Anomaly) Overview {
	stats := analytics.ComputeConsumptionStats(rs)
	o := Overview{
		Meter:            m,
		TotalKWh:         analytics.Round(stats.TotalKWh, 1),
		CurrentDailyKWh:  analytics.Round(stats.CurrentDailyKWh, 1),
		BaselineDailyKWh: analytics.Round(stats.BaselineDailyKWh, 1),
		VariationPct:     analytics.Round(stats.VariationPct, 1),
		ReadingsCount:    len(rs),
	}
	for _, r := range rs {
		if o.LastReadingAt == nil || r.Timestamp.After(*o.LastReadingAt) {
			ts := r.Timestamp
			o.LastReadingAt = &ts
		}
	}
	if m.Status != StatusPending {
		o.Status = StatusFromAnomalies(anomalies)
	}
	if ranked := anomaly.Rank(anomalies); len(ranked) > 0 {
		top := ranked[0]
		o.Anomaly = &AnomalyRef{ID: top.ID, Type: top.Type, Severity: top.Severity, Confidence: top.Confidence, Status: top.Status}
	}
	return o
}

func (s *Service) List(ctx context.Context, q ListQuery) ([]Overview, error) {
	if !validFilters[q.Status] {
		return nil, fmt.Errorf("%w: status %q", ErrInvalidQuery, q.Status)
	}
	if !validSorts[q.Sort] {
		return nil, fmt.Errorf("%w: sort %q", ErrInvalidQuery, q.Sort)
	}
	meters, err := s.meters.List(ctx)
	if err != nil {
		return nil, err
	}
	readings, err := s.readings.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	anomalies, err := s.anomalies.List(ctx, anomaly.Filter{})
	if err != nil {
		return nil, err
	}
	readingsByMeter := map[string][]reading.Reading{}
	for _, r := range readings {
		readingsByMeter[r.MeterID] = append(readingsByMeter[r.MeterID], r)
	}
	anomaliesByMeter := map[string][]anomaly.Anomaly{}
	for _, a := range anomalies {
		anomaliesByMeter[a.MeterID] = append(anomaliesByMeter[a.MeterID], a)
	}

	search := strings.ToLower(strings.TrimSpace(q.Search))
	out := make([]Overview, 0, len(meters))
	for _, m := range meters {
		o := buildOverview(m, readingsByMeter[m.MeterID], anomaliesByMeter[m.MeterID])
		if !o.Status.MatchesFilter(q.Status) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(m.MeterID), search) && !strings.Contains(strings.ToLower(m.Name), search) {
			continue
		}
		out = append(out, o)
	}
	sortOverviews(out, q.Sort, q.Order)
	return out, nil
}

func severityKey(o Overview) float64 {
	if o.Anomaly == nil || !o.Anomaly.Type.IsEscalable() {
		return 0
	}
	return float64(o.Anomaly.Severity.Weight())*10 + float64(o.Anomaly.Type.Weight())
}

func sortOverviews(items []Overview, by, order string) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].MeterID < items[j].MeterID })
	var key func(Overview) float64
	desc := order != "asc"
	switch by {
	case "consumption":
		key = func(o Overview) float64 { return o.CurrentDailyKWh }
	case "variation":
		key = func(o Overview) float64 { return o.VariationPct }
	case "severity":
		key = severityKey
	default:
		if order == "desc" {
			sort.SliceStable(items, func(i, j int) bool { return items[i].MeterID > items[j].MeterID })
		}
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		if desc {
			return key(items[i]) > key(items[j])
		}
		return key(items[i]) < key(items[j])
	})
}

func (s *Service) Get(ctx context.Context, meterID string) (Detail, error) {
	m, err := s.meters.Get(ctx, meterID)
	if err != nil {
		return Detail{}, err
	}
	rs, err := s.readings.ListByMeter(ctx, meterID, zeroTime, zeroTime)
	if err != nil {
		return Detail{}, err
	}
	anomalies, err := s.anomalies.List(ctx, anomaly.Filter{MeterID: meterID})
	if err != nil {
		return Detail{}, err
	}
	events, err := s.events.ListByMeter(ctx, meterID)
	if err != nil {
		return Detail{}, err
	}
	overview := buildOverview(m, rs, anomalies)
	d := Detail{
		Overview:   overview,
		Electrical: roundAverages(analytics.Averages(rs)),
		Events:     nonNilEvents(events),
		Anomalies:  anomaly.Rank(anomalies),
	}
	for _, p := range analytics.DailyTotals(rs) {
		d.Daily = append(d.Daily, DailyPoint{Date: p.Date, KWh: analytics.Round(p.KWh, 1), BaselineKWh: overview.BaselineDailyKWh})
	}
	if d.Daily == nil {
		d.Daily = []DailyPoint{}
	}
	if len(rs) > 0 {
		latest := rs[0]
		for _, r := range rs {
			if r.Timestamp.After(latest.Timestamp) {
				latest = r
			}
		}
		d.Latest = &latest
	}
	return d, nil
}

func (s *Service) Readings(ctx context.Context, meterID string, q ReadingsQuery) (ReadingsResult, error) {
	granularity := q.Granularity
	if granularity == "" {
		granularity = "hour"
	}
	if granularity != "hour" && granularity != "day" {
		return ReadingsResult{}, fmt.Errorf("%w: granularity %q", ErrInvalidQuery, q.Granularity)
	}
	if _, err := s.meters.Get(ctx, meterID); err != nil {
		return ReadingsResult{}, err
	}
	all, err := s.readings.ListByMeter(ctx, meterID, zeroTime, zeroTime)
	if err != nil {
		return ReadingsResult{}, err
	}
	rs := all
	if !q.From.IsZero() || !q.To.IsZero() {
		if rs, err = s.readings.ListByMeter(ctx, meterID, q.From, q.To); err != nil {
			return ReadingsResult{}, err
		}
	}
	res := ReadingsResult{MeterID: meterID, Granularity: granularity}
	if granularity == "hour" {
		profile := analytics.HourlyProfile(all)
		for _, r := range rs {
			res.Points = append(res.Points, SeriesPoint{
				Timestamp:      r.Timestamp,
				ConsumptionKWh: r.ConsumptionKWh,
				ExpectedKWh:    analytics.Round(profile[r.Timestamp.Hour()], 2),
				VoltageV:       r.VoltageV,
				CurrentA:       r.CurrentA,
				PowerFactor:    r.PowerFactor,
				Status:         r.Status,
			})
		}
	} else {
		res.Points = dailySeries(rs, analytics.ComputeConsumptionStats(all).BaselineDailyKWh)
	}
	if res.Points == nil {
		res.Points = []SeriesPoint{}
	}
	return res, nil
}

func dailySeries(rs []reading.Reading, baseline float64) []SeriesPoint {
	groups := map[string][]reading.Reading{}
	for _, r := range rs {
		key := analytics.DayOf(r.Timestamp).Format("2006-01-02")
		groups[key] = append(groups[key], r)
	}
	var out []SeriesPoint
	for _, day := range analytics.DailyTotals(rs) {
		avg := roundAverages(analytics.Averages(groups[day.Date.Format("2006-01-02")]))
		out = append(out, SeriesPoint{
			Timestamp:      day.Date,
			ConsumptionKWh: analytics.Round(day.KWh, 2),
			ExpectedKWh:    analytics.Round(baseline, 2),
			VoltageV:       avg.VoltageV,
			CurrentA:       avg.CurrentA,
			PowerFactor:    avg.PowerFactor,
		})
	}
	return out
}

func roundAverages(a analytics.ElectricalAverages) analytics.ElectricalAverages {
	return analytics.ElectricalAverages{
		ConsumptionKWh: analytics.Round(a.ConsumptionKWh, 2),
		VoltageV:       analytics.Round(a.VoltageV, 2),
		CurrentA:       analytics.Round(a.CurrentA, 2),
		PowerFactor:    analytics.Round(a.PowerFactor, 3),
	}
}

func nonNilEvents(items []event.Event) []event.Event {
	if items == nil {
		return []event.Event{}
	}
	return items
}
