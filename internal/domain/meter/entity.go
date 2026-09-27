package meter

import (
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
)

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusOK       Status = "OK"
	StatusAlert    Status = "ALERT"
	StatusCritical Status = "CRITICAL"
)

func (s Status) weight() int {
	switch s {
	case StatusCritical:
		return 3
	case StatusAlert:
		return 2
	case StatusOK:
		return 1
	}
	return 0
}

func (s Status) MatchesFilter(filter string) bool {
	switch filter {
	case "", "all":
		return true
	case "normal":
		return s == StatusOK
	case "alert":
		return s == StatusAlert
	case "critical":
		return s == StatusCritical
	case "pending":
		return s == StatusPending
	}
	return false
}

type Meter struct {
	ID        int64     `json:"id"`
	MeterID   string    `json:"meter_id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func StatusFromAnomalies(items []anomaly.Anomaly) Status {
	status := StatusOK
	for _, a := range items {
		candidate := statusFor(a)
		if candidate.weight() > status.weight() {
			status = candidate
		}
	}
	return status
}

func statusFor(a anomaly.Anomaly) Status {
	if !a.Status.IsActive() || !a.Type.IsEscalable() {
		return StatusOK
	}
	if a.Type == anomaly.TypeRealAnomaly && a.Severity == anomaly.SeverityHigh {
		return StatusCritical
	}
	return StatusAlert
}
