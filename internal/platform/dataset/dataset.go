package dataset

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

const (
	ReadingsFile = "readings.csv"
	EventsFile   = "events.csv"
	MetersFile   = "meters.csv"
)

var bom = string(rune(0xFEFF))

var timeLayouts = []string{"2006-01-02 15:04:05", "2006-01-02 15:04", time.RFC3339}

type Data struct {
	Meters   []meter.Meter
	Readings []reading.Reading
	Events   []event.Event
}

type table struct {
	columns map[string]int
	rows    [][]string
}

func (t table) value(row []string, column string) string {
	return strings.TrimSpace(row[t.columns[column]])
}

func readTable(r io.Reader, required ...string) (table, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return table{}, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return table{}, errors.New("csv is empty")
	}
	columns := map[string]int{}
	for i, name := range records[0] {
		columns[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, bom)))] = i
	}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return table{}, fmt.Errorf("missing required column %q", name)
		}
	}
	return table{columns: columns, rows: records[1:]}, nil
}

func parseTime(value string) (time.Time, error) {
	for _, layout := range timeLayouts {
		if ts, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q", value)
}

func parseFloats(t table, row []string, columns ...string) ([]float64, error) {
	out := make([]float64, len(columns))
	for i, column := range columns {
		v, err := strconv.ParseFloat(t.value(row, column), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid %s %q", column, t.value(row, column))
		}
		out[i] = v
	}
	return out, nil
}

func ParseReadings(r io.Reader) ([]reading.Reading, error) {
	t, err := readTable(r, "meter_id", "timestamp", "consumption_kwh", "voltage_v", "current_a", "power_factor")
	if err != nil {
		return nil, err
	}
	_, hasStatus := t.columns["status"]
	out := make([]reading.Reading, 0, len(t.rows))
	for i, row := range t.rows {
		line := i + 2
		ts, err := parseTime(t.value(row, "timestamp"))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		values, err := parseFloats(t, row, "consumption_kwh", "voltage_v", "current_a", "power_factor")
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		status := "OK"
		if hasStatus && t.value(row, "status") != "" {
			status = t.value(row, "status")
		}
		out = append(out, reading.Reading{
			MeterID:        t.value(row, "meter_id"),
			Timestamp:      ts,
			ConsumptionKWh: values[0],
			VoltageV:       values[1],
			CurrentA:       values[2],
			PowerFactor:    values[3],
			Status:         status,
		})
	}
	return out, nil
}

func ParseEvents(r io.Reader) ([]event.Event, error) {
	t, err := readTable(r, "meter_id", "event_timestamp", "event_type", "description")
	if err != nil {
		return nil, err
	}
	out := make([]event.Event, 0, len(t.rows))
	for i, row := range t.rows {
		ts, err := parseTime(t.value(row, "event_timestamp"))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+2, err)
		}
		out = append(out, event.Event{
			MeterID:     t.value(row, "meter_id"),
			Timestamp:   ts,
			Type:        event.Type(strings.ToUpper(t.value(row, "event_type"))),
			Description: t.value(row, "description"),
		})
	}
	return out, nil
}

func ParseMeters(r io.Reader) ([]meter.Meter, error) {
	t, err := readTable(r, "meter_id", "name", "location")
	if err != nil {
		return nil, err
	}
	out := make([]meter.Meter, 0, len(t.rows))
	for _, row := range t.rows {
		out = append(out, meter.Meter{
			MeterID:  t.value(row, "meter_id"),
			Name:     t.value(row, "name"),
			Location: t.value(row, "location"),
			Status:   meter.StatusPending,
		})
	}
	return out, nil
}

func parseFile[T any](path string, parse func(io.Reader) (T, error)) (T, error) {
	var zero T
	f, err := os.Open(path)
	if err != nil {
		return zero, err
	}
	defer f.Close()
	out, err := parse(f)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return out, nil
}

func LoadDir(dir string) (Data, error) {
	readings, err := parseFile(filepath.Join(dir, ReadingsFile), ParseReadings)
	if err != nil {
		return Data{}, err
	}
	events, err := parseFile(filepath.Join(dir, EventsFile), ParseEvents)
	if err != nil {
		return Data{}, err
	}
	meters, err := parseFile(filepath.Join(dir, MetersFile), ParseMeters)
	if errors.Is(err, os.ErrNotExist) {
		meters, err = nil, nil
	}
	if err != nil {
		return Data{}, err
	}
	return Data{Meters: mergeMeters(meters, readings), Readings: readings, Events: events}, nil
}

func mergeMeters(catalog []meter.Meter, readings []reading.Reading) []meter.Meter {
	index := map[string]meter.Meter{}
	for _, m := range catalog {
		index[m.MeterID] = m
	}
	for _, r := range readings {
		if _, ok := index[r.MeterID]; !ok {
			index[r.MeterID] = meter.Meter{MeterID: r.MeterID, Name: "Medidor " + r.MeterID, Location: "Sin ubicación", Status: meter.StatusPending}
		}
	}
	out := make([]meter.Meter, 0, len(index))
	for _, m := range index {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MeterID < out[j].MeterID })
	return out
}
