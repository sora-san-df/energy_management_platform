package analysis

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database"
)

var ErrRunNotFound = errors.New("analysis run not found")

type SQLiteRunRepository struct {
	db *sql.DB
}

func NewSQLiteRunRepository(db *sql.DB) *SQLiteRunRepository {
	return &SQLiteRunRepository{db: db}
}

const selectRuns = `SELECT id, status, scope, steps, current_step, meters_analyzed, anomalies_detected, high_priority,
	avg_confidence, ai_provider, ai_status, summary, error, duration_ms, started_at, finished_at FROM analyses`

func scanRun(row interface{ Scan(...any) error }) (Run, error) {
	var r Run
	var scope, steps, startedAt string
	var finishedAt sql.NullString
	err := row.Scan(&r.ID, &r.Status, &scope, &steps, &r.CurrentStep, &r.MetersAnalyzed, &r.AnomaliesDetected,
		&r.HighPriority, &r.AvgConfidence, &r.AIProvider, &r.AIStatus, &r.Summary, &r.Error, &r.DurationMs, &startedAt, &finishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, err
	}
	if err := database.FromJSON(scope, &r.Scope); err != nil {
		return Run{}, err
	}
	if err := database.FromJSON(steps, &r.Steps); err != nil {
		return Run{}, err
	}
	if r.StartedAt, err = database.ParseTime(startedAt); err != nil {
		return Run{}, err
	}
	if r.FinishedAt, err = database.ParseNullableTime(finishedAt); err != nil {
		return Run{}, err
	}
	return r, nil
}

func encodeRun(r Run) (string, string, error) {
	scope, err := database.ToJSON(r.Scope)
	if err != nil {
		return "", "", err
	}
	steps, err := database.ToJSON(r.Steps)
	if err != nil {
		return "", "", err
	}
	return scope, steps, nil
}

func (s *SQLiteRunRepository) Create(ctx context.Context, r *Run) error {
	scope, steps, err := encodeRun(*r)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO analyses (status, scope, steps, current_step, ai_provider, ai_status, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, r.Status, scope, steps, r.CurrentStep, r.AIProvider, r.AIStatus, database.FormatTime(r.StartedAt))
	if err != nil {
		return err
	}
	r.ID, err = res.LastInsertId()
	return err
}

func (s *SQLiteRunRepository) Update(ctx context.Context, r Run) error {
	scope, steps, err := encodeRun(r)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE analyses SET status = ?, scope = ?, steps = ?, current_step = ?, meters_analyzed = ?,
		anomalies_detected = ?, high_priority = ?, avg_confidence = ?, ai_provider = ?, ai_status = ?, summary = ?, error = ?,
		duration_ms = ?, finished_at = ? WHERE id = ?`,
		r.Status, scope, steps, r.CurrentStep, r.MetersAnalyzed, r.AnomaliesDetected, r.HighPriority, r.AvgConfidence,
		r.AIProvider, r.AIStatus, r.Summary, r.Error, r.DurationMs, database.NullableTime(r.FinishedAt), r.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRunNotFound
	}
	return nil
}

func (s *SQLiteRunRepository) Get(ctx context.Context, id int64) (Run, error) {
	return scanRun(s.db.QueryRowContext(ctx, selectRuns+` WHERE id = ?`, id))
}

func (s *SQLiteRunRepository) Latest(ctx context.Context) (Run, error) {
	return scanRun(s.db.QueryRowContext(ctx, selectRuns+` ORDER BY id DESC LIMIT 1`))
}
