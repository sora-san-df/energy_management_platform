package reading

import "time"

type Reading struct {
	ID             int64     `json:"id"`
	MeterID        string    `json:"meter_id"`
	Timestamp      time.Time `json:"timestamp"`
	ConsumptionKWh float64   `json:"consumption_kwh"`
	VoltageV       float64   `json:"voltage_v"`
	CurrentA       float64   `json:"current_a"`
	PowerFactor    float64   `json:"power_factor"`
	Status         string    `json:"status"`
}
