package analysis

import (
	"fmt"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/analytics"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
)

type RunStatus string

const (
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusCompleted RunStatus = "COMPLETED"
	RunStatusFailed    RunStatus = "FAILED"
)

type StepStatus string

const (
	StepPending StepStatus = "PENDING"
	StepRunning StepStatus = "RUNNING"
	StepDone    StepStatus = "DONE"
	StepFailed  StepStatus = "FAILED"
)

type AIStatus string

const (
	AIStatusOK       AIStatus = "ok"
	AIStatusPartial  AIStatus = "partial"
	AIStatusFallback AIStatus = "fallback"
	AIStatusDisabled AIStatus = "disabled"
)

const (
	StepReadings       = "readings"
	StepBaseline       = "baseline"
	StepDetection      = "detection"
	StepCorrelation    = "correlation"
	StepEvents         = "events"
	StepExplanation    = "explanation"
	StepRecommendation = "recommendation"
)

var pipeline = []struct{ key, label string }{
	{StepReadings, "Lecturas"},
	{StepBaseline, "Baseline"},
	{StepDetection, "Detección"},
	{StepCorrelation, "Correlación"},
	{StepEvents, "Eventos"},
	{StepExplanation, "Explicación"},
	{StepRecommendation, "Recomendación"},
}

type Step struct {
	Key        string     `json:"key"`
	Label      string     `json:"label"`
	Status     StepStatus `json:"status"`
	Detail     string     `json:"detail,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type Run struct {
	ID                int64      `json:"id"`
	Status            RunStatus  `json:"status"`
	Scope             []string   `json:"scope"`
	Steps             []Step     `json:"steps"`
	CurrentStep       string     `json:"current_step"`
	MetersAnalyzed    int        `json:"meters_analyzed"`
	AnomaliesDetected int        `json:"anomalies_detected"`
	HighPriority      int        `json:"high_priority"`
	AvgConfidence     float64    `json:"avg_confidence"`
	AIProvider        string     `json:"ai_provider"`
	AIStatus          AIStatus   `json:"ai_status"`
	Summary           string     `json:"summary"`
	Error             string     `json:"error,omitempty"`
	DurationMs        int64      `json:"duration_ms"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}

func NewRun(scope []string, now time.Time) Run {
	steps := make([]Step, len(pipeline))
	for i, p := range pipeline {
		steps[i] = Step{Key: p.key, Label: p.label, Status: StepPending}
	}
	if scope == nil {
		scope = []string{}
	}
	return Run{Status: RunStatusRunning, Scope: scope, Steps: steps, StartedAt: now.UTC()}
}

func (r *Run) step(key string) *Step {
	for i := range r.Steps {
		if r.Steps[i].Key == key {
			return &r.Steps[i]
		}
	}
	return nil
}

func (r *Run) StartStep(key string, now time.Time) {
	if s := r.step(key); s != nil {
		ts := now.UTC()
		s.Status = StepRunning
		s.StartedAt = &ts
		r.CurrentStep = key
	}
}

func (r *Run) CompleteStep(key, detail string, now time.Time) {
	if s := r.step(key); s != nil {
		ts := now.UTC()
		s.Status = StepDone
		s.Detail = detail
		s.FinishedAt = &ts
	}
}

func (r *Run) finish(now time.Time) {
	ts := now.UTC()
	r.FinishedAt = &ts
	r.DurationMs = ts.Sub(r.StartedAt).Milliseconds()
}

func (r *Run) Fail(err error, now time.Time) {
	if s := r.step(r.CurrentStep); s != nil && s.Status == StepRunning {
		s.Status = StepFailed
	}
	r.Status = RunStatusFailed
	r.Error = err.Error()
	r.Summary = "El análisis falló"
	r.finish(now)
}

func (r *Run) Complete(anomalies []anomaly.Anomaly, metersAnalyzed int, aiStatus AIStatus, now time.Time) {
	r.Status = RunStatusCompleted
	r.MetersAnalyzed = metersAnalyzed
	r.AIStatus = aiStatus
	r.AnomaliesDetected = len(anomalies)
	r.HighPriority = 0
	var confidences []float64
	for _, a := range anomalies {
		if a.IsHighPriority() {
			r.HighPriority++
		}
		confidences = append(confidences, a.Confidence)
	}
	r.AvgConfidence = analytics.Round(analytics.Mean(confidences), 2)
	r.Summary = summarize(r.AnomaliesDetected, r.HighPriority)
	r.CurrentStep = ""
	r.finish(now)
}

func summarize(detected, high int) string {
	if detected == 0 {
		return "Sin anomalías detectadas"
	}
	noun := "anomalías detectadas"
	if detected == 1 {
		noun = "anomalía detectada"
	}
	verb := "requieren"
	if high == 1 {
		verb = "requiere"
	}
	return fmt.Sprintf("%d %s · %d %s atención prioritaria", detected, noun, high, verb)
}
