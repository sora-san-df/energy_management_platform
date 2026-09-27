package dataset

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
)

func TestParseReadings(t *testing.T) {
	input := `meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status
M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK
M-101,2026-09-01 01:00:00,24.1,220.4,102.5,0.95,OK
`
	readings, err := ParseReadings(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("expected 2 readings, got %d", len(readings))
	}
	r := readings[0]
	want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if r.MeterID != "M-101" || !r.Timestamp.Equal(want) || r.ConsumptionKWh != 23.5 || r.VoltageV != 221.9 || r.CurrentA != 101.28 || r.PowerFactor != 0.954 || r.Status != "OK" {
		t.Fatalf("unexpected reading %+v", r)
	}
}

func TestParseReadingsRejectsMissingColumns(t *testing.T) {
	_, err := ParseReadings(strings.NewReader("meter_id,timestamp\nM-1,2026-09-01 00:00:00\n"))
	if err == nil {
		t.Fatal("expected error for missing columns")
	}
}

func TestParseReadingsReportsInvalidNumbersWithLine(t *testing.T) {
	input := `meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status
M-101,2026-09-01 00:00:00,abc,221.9,101.28,0.954,OK
`
	_, err := ParseReadings(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("expected error mentioning line 2, got %v", err)
	}
}

func TestParseReadingsSupportsReorderedColumns(t *testing.T) {
	input := `timestamp,meter_id,status,power_factor,current_a,voltage_v,consumption_kwh
2026-09-01 00:00:00,M-7,OK,0.9,10,220,2
`
	readings, err := ParseReadings(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if readings[0].MeterID != "M-7" || readings[0].ConsumptionKWh != 2 || readings[0].PowerFactor != 0.9 {
		t.Fatalf("unexpected %+v", readings[0])
	}
}

func TestParseEvents(t *testing.T) {
	input := `meter_id,event_timestamp,event_type,description
M-104,2026-09-11 00:00,OPERATIONAL_CHANGE,New production line activated
M-106,2026-09-08 00:00,SCHEDULED_OUTAGE,"Scheduled maintenance outage, 12 hours"
`
	events, err := ParseEvents(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != event.TypeOperationalChange || !events[0].Timestamp.Equal(time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected event %+v", events[0])
	}
	if events[1].Description != "Scheduled maintenance outage, 12 hours" {
		t.Fatalf("unexpected description %q", events[1].Description)
	}
}

func TestParseMeters(t *testing.T) {
	input := "meter_id,name,location\nM-101,Oficinas,Sede Central\n"
	meters, err := ParseMeters(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(meters) != 1 || meters[0].MeterID != "M-101" || meters[0].Name != "Oficinas" || meters[0].Location != "Sede Central" {
		t.Fatalf("unexpected meters %+v", meters)
	}
}

func TestLoadDirWithRealDataset(t *testing.T) {
	data, err := LoadDir(filepath.Join("..", "..", "..", "data"))
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}
	if len(data.Readings) != 4032 {
		t.Fatalf("expected 4032 readings, got %d", len(data.Readings))
	}
	if len(data.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(data.Events))
	}
	if len(data.Meters) != 12 {
		t.Fatalf("expected 12 meters, got %d", len(data.Meters))
	}
}

func TestLoadDirDerivesMetersWhenCatalogIsMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "readings.csv"), "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status\nM-2,2026-09-01 00:00:00,1,220,5,0.9,OK\nM-1,2026-09-01 00:00:00,1,220,5,0.9,OK\n")
	writeFile(t, filepath.Join(dir, "events.csv"), "meter_id,event_timestamp,event_type,description\n")

	data, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Meters) != 2 || data.Meters[0].MeterID != "M-1" || data.Meters[1].MeterID != "M-2" {
		t.Fatalf("expected meters derived from readings sorted by id, got %+v", data.Meters)
	}
}
