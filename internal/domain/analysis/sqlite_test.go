package analysis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database/dbtest"
)

func TestSQLiteRunRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLiteRunRepository(dbtest.New(t))
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)

	if _, err := repo.Latest(ctx); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("expected ErrRunNotFound on empty repo, got %v", err)
	}

	run := NewRun([]string{"M-109"}, now)
	if err := repo.Create(ctx, &run); err != nil {
		t.Fatalf("create: %v", err)
	}
	if run.ID == 0 {
		t.Fatal("expected id to be assigned")
	}

	run.StartStep(StepReadings, now)
	run.CompleteStep(StepReadings, "336 lecturas", now)
	run.Complete(nil, 1, AIStatusFallback, now.Add(2*time.Second))
	if err := repo.Update(ctx, run); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != RunStatusCompleted || got.AIStatus != AIStatusFallback || got.MetersAnalyzed != 1 {
		t.Fatalf("unexpected run %+v", got)
	}
	if len(got.Scope) != 1 || got.Scope[0] != "M-109" {
		t.Fatalf("scope not persisted: %+v", got.Scope)
	}
	if got.Steps[0].Status != StepDone || got.Steps[0].Detail != "336 lecturas" {
		t.Fatalf("steps not persisted: %+v", got.Steps[0])
	}
	if got.FinishedAt == nil || got.DurationMs != 2000 || got.Summary == "" {
		t.Fatalf("completion data not persisted: %+v", got)
	}

	second := NewRun(nil, now.Add(time.Minute))
	if err := repo.Create(ctx, &second); err != nil {
		t.Fatal(err)
	}
	latest, err := repo.Latest(ctx)
	if err != nil || latest.ID != second.ID {
		t.Fatalf("expected latest run %d, got %d (%v)", second.ID, latest.ID, err)
	}

	if _, err := repo.Get(ctx, 999); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("expected ErrRunNotFound, got %v", err)
	}
}
