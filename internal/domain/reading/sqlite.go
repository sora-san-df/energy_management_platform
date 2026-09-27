package reading

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

const selectReadings = `SELECT id, meter_id, timestamp, consumption_kwh, voltage_v, current_a, power_factor, status FROM readings`

func (r *SQLiteRepository) query(ctx context.Context, query string, args ...any) ([]Reading, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Reading
	for rows.Next() {
		var rd Reading
		var ts string
		if err := rows.Scan(&rd.ID, &rd.MeterID, &ts, &rd.ConsumptionKWh, &rd.VoltageV, &rd.CurrentA, &rd.PowerFactor, &rd.Status); err != nil {
			return nil, err
		}
		if rd.Timestamp, err = database.ParseTime(ts); err != nil {
			return nil, err
		}
		out = append(out, rd)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) ListByMeter(ctx context.Context, meterID string, from, to time.Time) ([]Reading, error) {
	clauses := []string{"meter_id = ?"}
	args := []any{meterID}
	if !from.IsZero() {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, database.FormatTime(from))
	}
	if !to.IsZero() {
		clauses = append(clauses, "timestamp <= ?")
		args = append(args, database.FormatTime(to))
	}
	return r.query(ctx, selectReadings+` WHERE `+strings.Join(clauses, " AND ")+` ORDER BY timestamp`, args...)
}

func (r *SQLiteRepository) ListAll(ctx context.Context) ([]Reading, error) {
	return r.query(ctx, selectReadings+` ORDER BY meter_id, timestamp`)
}

func (r *SQLiteRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM readings`).Scan(&n)
	return n, err
}

func (r *SQLiteRepository) InsertMany(ctx context.Context, readings []Reading) error {
	return database.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO readings (meter_id, timestamp, consumption_kwh, voltage_v, current_a, power_factor, status) VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, rd := range readings {
			if _, err := stmt.ExecContext(ctx, rd.MeterID, database.FormatTime(rd.Timestamp), rd.ConsumptionKWh, rd.VoltageV, rd.CurrentA, rd.PowerFactor, rd.Status); err != nil {
				return err
			}
		}
		return nil
	})
}
