package anomaly

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database/dbtest"
)

func seedAnalysis(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO analyses (status, scope, steps, started_at) VALUES ('COMPLETED', '[]', '[]', '2026-09-15T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func sample(meterID string, typ Type, sev Severity) Anomaly {
	start := time.Date(2026, 9, 12, 14, 0, 0, 0, time.UTC)
	end := start.Add(58 * time.Hour)
	return Anomaly{
		MeterID:           meterID,
		DetectedAt:        start,
		Type:              typ,
		Severity:          sev,
		Confidence:        0.96,
		Reason:            "Consumo 103,7% por encima del baseline",
		RecommendedAction: "Investigar medidor e instalación",
		Status:            InitialStatus(typ),
		Source:            SourceGemini,
		AIInsights:        []string{"La corriente se duplicó"},
		BaselineDailyKWh:  1070,
		CurrentDailyKWh:   2180,
		VariationPct:      103.7,
		WindowStart:       &start,
		WindowEnd:         &end,
		AffectedVariables: []string{"consumption_kwh", "power_factor"},
		Evidence:          []Evidence{{Label: "Variación", Value: "+103,7%"}},
		RelatedEvent:      &event.Event{ID: 3, MeterID: meterID, Timestamp: start, Type: event.TypeUnknown, Description: "No operational event reported"},
	}
}

func TestSQLiteRepositoryReplaceAndGet(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t)
	repo := NewSQLiteRepository(db)
	analysisID := seedAnalysis(t, db)

	a := sample("M-109", TypeRealAnomaly, SeverityHigh)
	a.AnalysisID = analysisID
	saved, err := repo.ReplaceForMeters(ctx, []string{"M-109"}, []Anomaly{a})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if len(saved) != 1 || saved[0].ID == 0 || saved[0].CreatedAt.IsZero() {
		t.Fatalf("expected saved anomaly with id, got %+v", saved)
	}

	got, err := repo.Get(ctx, saved[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.MeterID != "M-109" || got.Type != TypeRealAnomaly || got.Severity != SeverityHigh || got.Confidence != 0.96 {
		t.Fatalf("unexpected anomaly %+v", got)
	}
	if got.WindowStart == nil || !got.WindowStart.Equal(*a.WindowStart) || got.WindowEnd == nil {
		t.Fatalf("window not persisted: %+v", got)
	}
	if len(got.Evidence) != 1 || got.Evidence[0].Value != "+103,7%" {
		t.Fatalf("evidence not persisted: %+v", got.Evidence)
	}
	if len(got.AffectedVariables) != 2 || len(got.AIInsights) != 1 {
		t.Fatalf("json columns not persisted: %+v", got)
	}
	if got.RelatedEvent == nil || got.RelatedEvent.Type != event.TypeUnknown {
		t.Fatalf("related event not persisted: %+v", got.RelatedEvent)
	}
	if got.Source != SourceGemini {
		t.Fatalf("source not persisted: %s", got.Source)
	}

	if _, err := repo.Get(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryReplaceOnlyTouchesGivenMeters(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t)
	repo := NewSQLiteRepository(db)
	analysisID := seedAnalysis(t, db)

	first := []Anomaly{sample("M-109", TypeRealAnomaly, SeverityHigh), sample("M-112", TypeDataQuality, SeverityHigh)}
	for i := range first {
		first[i].AnalysisID = analysisID
	}
	if _, err := repo.ReplaceForMeters(ctx, []string{"M-109", "M-112"}, first); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.ReplaceForMeters(ctx, []string{"M-109"}, nil); err != nil {
		t.Fatal(err)
	}

	all, err := repo.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].MeterID != "M-112" {
		t.Fatalf("expected only M-112 to remain, got %+v", all)
	}
}

func TestSQLiteRepositoryListFilters(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t)
	repo := NewSQLiteRepository(db)
	analysisID := seedAnalysis(t, db)

	items := []Anomaly{
		sample("M-109", TypeRealAnomaly, SeverityHigh),
		sample("M-112", TypeDataQuality, SeverityHigh),
		sample("M-104", TypeExplainable, SeverityMedium),
		sample("M-106", TypeFalsePositive, SeverityLow),
	}
	for i := range items {
		items[i].AnalysisID = analysisID
	}
	if _, err := repo.ReplaceForMeters(ctx, []string{"M-109", "M-112", "M-104", "M-106"}, items); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		filter Filter
		want   int
	}{
		{"all", Filter{}, 4},
		{"by meter", Filter{MeterID: "M-104"}, 1},
		{"by severity", Filter{Severity: SeverityHigh}, 2},
		{"by type", Filter{Type: TypeFalsePositive}, 1},
		{"by status", Filter{Status: StatusDismissed}, 1},
		{"combined", Filter{Severity: SeverityHigh, Type: TypeDataQuality}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := repo.List(ctx, c.filter)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.want {
				t.Fatalf("got %d anomalies, want %d", len(got), c.want)
			}
		})
	}
}

func TestSQLiteRepositoryUpdateStatus(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t)
	repo := NewSQLiteRepository(db)
	a := sample("M-109", TypeRealAnomaly, SeverityHigh)
	a.AnalysisID = seedAnalysis(t, db)
	saved, err := repo.ReplaceForMeters(ctx, []string{"M-109"}, []Anomaly{a})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.UpdateStatus(ctx, saved[0].ID, StatusInvestigating); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, saved[0].ID)
	if got.Status != StatusInvestigating {
		t.Fatalf("status = %s", got.Status)
	}
	if err := repo.UpdateStatus(ctx, 999, StatusResolved); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
