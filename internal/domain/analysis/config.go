package analysis

import "time"

type Config struct {
	DeviationThreshold    float64
	MinShiftHours         int
	MaxGapHours           int
	PersistentTailHours   int
	ComparisonWindowHours int
	HighSeverityPct       float64
	NominalVoltage        float64
	VoltageTolerancePct   float64
	VoltageJumpV          float64
	PowerFactorJump       float64
	EnergyRatioMin        float64
	EnergyRatioMax        float64
	MinQualityReadings    int
	PowerFactorShift      float64
	VoltageShiftPct       float64
	CurrentShiftPct       float64
	EventMatchWindow      time.Duration
	QualityEventWindow    time.Duration
}

func DefaultConfig() Config {
	return Config{
		DeviationThreshold:    0.30,
		MinShiftHours:         3,
		MaxGapHours:           2,
		PersistentTailHours:   6,
		ComparisonWindowHours: 72,
		HighSeverityPct:       50,
		NominalVoltage:        220,
		VoltageTolerancePct:   5,
		VoltageJumpV:          10,
		PowerFactorJump:       0.15,
		EnergyRatioMin:        0.6,
		EnergyRatioMax:        1.6,
		MinQualityReadings:    6,
		PowerFactorShift:      0.08,
		VoltageShiftPct:       3,
		CurrentShiftPct:       20,
		EventMatchWindow:      6 * time.Hour,
		QualityEventWindow:    24 * time.Hour,
	}
}
