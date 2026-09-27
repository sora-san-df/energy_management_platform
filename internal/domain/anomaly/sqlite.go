package anomaly

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database"
)

var ErrNotFound = errors.New("anomaly not found")

type Filter struct {
	MeterID  string
	Type     Type
	Severity Severity
	Status   Status
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

const selectAnomalies = `SELECT id, analysis_id, meter_id, detected_at, type, severity, confidence, reason, recommended_action,
	status, source, ai_error, ai_insights, baseline_daily_kwh, current_daily_kwh, variation_pct, window_start, window_end,
	affected_variables, evidence, related_event, created_at FROM anomalies`

func scanAnomaly(row interface{ Scan(...any) error }) (Anomaly, error) {
	var a Anomaly
	var detectedAt, createdAt, insights, variables, evidence string
	var windowStart, windowEnd, related sql.NullString
	err := row.Scan(&a.ID, &a.AnalysisID, &a.MeterID, &detectedAt, &a.Type, &a.Severity, &a.Confidence, &a.Reason,
		&a.RecommendedAction, &a.Status, &a.Source, &a.AIError, &insights, &a.BaselineDailyKWh, &a.CurrentDailyKWh,
		&a.VariationPct, &windowStart, &windowEnd, &variables, &evidence, &related, &createdAt)
	if err != nil {
		return Anomaly{}, err
	}
	if a.DetectedAt, err = database.ParseTime(detectedAt); err != nil {
		return Anomaly{}, err
	}
	if a.CreatedAt, err = database.ParseTime(createdAt); err != nil {
		return Anomaly{}, err
	}
	if a.WindowStart, err = database.ParseNullableTime(windowStart); err != nil {
		return Anomaly{}, err
	}
	if a.WindowEnd, err = database.ParseNullableTime(windowEnd); err != nil {
		return Anomaly{}, err
	}
	a.AIInsights, a.AffectedVariables, a.Evidence = []string{}, []string{}, []Evidence{}
	if err := database.FromJSON(insights, &a.AIInsights); err != nil {
		return Anomaly{}, err
	}
	if err := database.FromJSON(variables, &a.AffectedVariables); err != nil {
		return Anomaly{}, err
	}
	if err := database.FromJSON(evidence, &a.Evidence); err != nil {
		return Anomaly{}, err
	}
	if related.Valid && related.String != "" {
		var e event.Event
		if err := database.FromJSON(related.String, &e); err != nil {
			return Anomaly{}, err
		}
		a.RelatedEvent = &e
	}
	return a, nil
}

func (r *SQLiteRepository) List(ctx context.Context, f Filter) ([]Anomaly, error) {
	var clauses []string
	var args []any
	add := func(column, value string) {
		if value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	add("meter_id", f.MeterID)
	add("type", string(f.Type))
	add("severity", string(f.Severity))
	add("status", string(f.Status))
	query := selectAnomalies
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	rows, err := r.db.QueryContext(ctx, query+" ORDER BY id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Anomaly
	for rows.Next() {
		a, err := scanAnomaly(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) Get(ctx context.Context, id int64) (Anomaly, error) {
	a, err := scanAnomaly(r.db.QueryRowContext(ctx, selectAnomalies+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Anomaly{}, ErrNotFound
	}
	return a, err
}

func (r *SQLiteRepository) ReplaceForMeters(ctx context.Context, meterIDs []string, items []Anomaly) ([]Anomaly, error) {
	saved := make([]Anomaly, 0, len(items))
	err := database.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if len(meterIDs) > 0 {
			placeholders := strings.TrimSuffix(strings.Repeat("?,", len(meterIDs)), ",")
			args := make([]any, len(meterIDs))
			for i, id := range meterIDs {
				args[i] = id
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM anomalies WHERE meter_id IN ("+placeholders+")", args...); err != nil {
				return err
			}
		}
		now := time.Now().UTC().Truncate(time.Second)
		for _, a := range items {
			a.CreatedAt = now
			id, err := insert(ctx, tx, a)
			if err != nil {
				return err
			}
			a.ID = id
			saved = append(saved, a)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func insert(ctx context.Context, tx *sql.Tx, a Anomaly) (int64, error) {
	insights, err := database.ToJSON(nonNilStrings(a.AIInsights))
	if err != nil {
		return 0, err
	}
	variables, err := database.ToJSON(nonNilStrings(a.AffectedVariables))
	if err != nil {
		return 0, err
	}
	evidence := a.Evidence
	if evidence == nil {
		evidence = []Evidence{}
	}
	evidenceJSON, err := database.ToJSON(evidence)
	if err != nil {
		return 0, err
	}
	var related sql.NullString
	if a.RelatedEvent != nil {
		raw, err := database.ToJSON(a.RelatedEvent)
		if err != nil {
			return 0, err
		}
		related = sql.NullString{String: raw, Valid: true}
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO anomalies (analysis_id, meter_id, detected_at, type, severity, confidence, reason,
		recommended_action, status, source, ai_error, ai_insights, baseline_daily_kwh, current_daily_kwh, variation_pct,
		window_start, window_end, affected_variables, evidence, related_event, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.AnalysisID, a.MeterID, database.FormatTime(a.DetectedAt), a.Type, a.Severity, a.Confidence, a.Reason,
		a.RecommendedAction, a.Status, a.Source, a.AIError, insights, a.BaselineDailyKWh, a.CurrentDailyKWh, a.VariationPct,
		database.NullableTime(a.WindowStart), database.NullableTime(a.WindowEnd), variables, evidenceJSON, related,
		database.FormatTime(a.CreatedAt))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func (r *SQLiteRepository) UpdateStatus(ctx context.Context, id int64, status Status) error {
	res, err := r.db.ExecContext(ctx, `UPDATE anomalies SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
