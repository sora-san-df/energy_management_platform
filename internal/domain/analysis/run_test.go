package analysis

import (
	"errors"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
)

func TestNewRunHasPipelineSteps(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	run := NewRun([]string{"M-109"}, now)

	if run.Status != RunStatusRunning || !run.StartedAt.Equal(now) {
		t.Fatalf("unexpected run %+v", run)
	}
	want := []string{StepReadings, StepBaseline, StepDetection, StepCorrelation, StepEvents, StepExplanation, StepRecommendation}
	if len(run.Steps) != len(want) {
		t.Fatalf("expected %d steps, got %d", len(want), len(run.Steps))
	}
	for i, key := range want {
		if run.Steps[i].Key != key || run.Steps[i].Status != StepPending || run.Steps[i].Label == "" {
			t.Fatalf("unexpected step %d: %+v", i, run.Steps[i])
		}
	}
}

func TestRunStepLifecycle(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	run := NewRun(nil, now)

	run.StartStep(StepReadings, now)
	if run.CurrentStep != StepReadings || run.Steps[0].Status != StepRunning || run.Steps[0].StartedAt == nil {
		t.Fatalf("step not started: %+v", run.Steps[0])
	}

	run.CompleteStep(StepReadings, "4.032 lecturas", now.Add(time.Second))
	if run.Steps[0].Status != StepDone || run.Steps[0].Detail != "4.032 lecturas" || run.Steps[0].FinishedAt == nil {
		t.Fatalf("step not completed: %+v", run.Steps[0])
	}
}

func TestRunComplete(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	run := NewRun(nil, now)
	anomalies := []anomaly.Anomaly{
		{Type: anomaly.TypeRealAnomaly, Severity: anomaly.SeverityHigh, Confidence: 0.96},
		{Type: anomaly.TypeDataQuality, Severity: anomaly.SeverityHigh, Confidence: 0.94},
		{Type: anomaly.TypeExplainable, Severity: anomaly.SeverityMedium, Confidence: 0.90},
		{Type: anomaly.TypeFalsePositive, Severity: anomaly.SeverityLow, Confidence: 0.80},
	}

	run.Complete(anomalies, 12, AIStatusOK, now.Add(3*time.Second))

	if run.Status != RunStatusCompleted || run.FinishedAt == nil {
		t.Fatalf("run not completed: %+v", run)
	}
	if run.AnomaliesDetected != 4 || run.HighPriority != 2 || run.MetersAnalyzed != 12 {
		t.Fatalf("unexpected counters %+v", run)
	}
	if run.AvgConfidence != 0.9 {
		t.Fatalf("avg confidence = %v, want 0.9", run.AvgConfidence)
	}
	if run.Summary != "4 anomalías detectadas · 2 requieren atención prioritaria" {
		t.Fatalf("unexpected summary %q", run.Summary)
	}
	if run.DurationMs != 3000 {
		t.Fatalf("duration = %d", run.DurationMs)
	}
}

func TestRunFail(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	run := NewRun(nil, now)
	run.StartStep(StepDetection, now)

	run.Fail(errors.New("boom"), now.Add(time.Second))

	if run.Status != RunStatusFailed || run.Error != "boom" || run.FinishedAt == nil {
		t.Fatalf("run not failed: %+v", run)
	}
	for _, s := range run.Steps {
		if s.Key == StepDetection && s.Status != StepFailed {
			t.Fatalf("current step must be marked as failed: %+v", s)
		}
	}
}

func TestSummaryWithoutAnomalies(t *testing.T) {
	run := NewRun(nil, time.Now())
	run.Complete(nil, 12, AIStatusDisabled, time.Now())
	if run.Summary != "Sin anomalías detectadas" {
		t.Fatalf("unexpected summary %q", run.Summary)
	}
}
