package event

import "time"

type Type string

const (
	TypeOperationalChange Type = "OPERATIONAL_CHANGE"
	TypeScheduledOutage   Type = "SCHEDULED_OUTAGE"
	TypeDataQuality       Type = "DATA_QUALITY"
	TypeUnknown           Type = "UNKNOWN"
)

type Event struct {
	ID          int64     `json:"id"`
	MeterID     string    `json:"meter_id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        Type      `json:"type"`
	Description string    `json:"description"`
}
