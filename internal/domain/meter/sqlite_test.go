package meter

import (
	"context"
	"errors"
	"testing"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database/dbtest"
)

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLiteRepository(dbtest.New(t))

	count, err := repo.Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("expected empty repository, got %d (%v)", count, err)
	}

	err = repo.InsertMany(ctx, []Meter{
		{MeterID: "M-102", Name: "Compresores", Location: "Planta Norte", Status: StatusPending},
		{MeterID: "M-101", Name: "Oficinas", Location: "Sede Central", Status: StatusPending},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	meters, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(meters) != 2 || meters[0].MeterID != "M-101" || meters[0].ID == 0 || meters[0].CreatedAt.IsZero() {
		t.Fatalf("unexpected list %+v", meters)
	}

	if err := repo.UpdateStatus(ctx, "M-102", StatusCritical); err != nil {
		t.Fatal(err)
	}
	m, err := repo.Get(ctx, "M-102")
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != StatusCritical || m.Name != "Compresores" {
		t.Fatalf("unexpected meter %+v", m)
	}

	if _, err := repo.Get(ctx, "M-999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := repo.UpdateStatus(ctx, "M-999", StatusOK); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on update, got %v", err)
	}
}
