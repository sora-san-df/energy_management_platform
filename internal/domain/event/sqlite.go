package event

import (
	"context"
	"database/sql"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

const selectEvents = `SELECT id, meter_id, timestamp, type, description FROM events`

func (r *SQLiteRepository) query(ctx context.Context, query string, args ...any) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var ts string
		if err := rows.Scan(&e.ID, &e.MeterID, &ts, &e.Type, &e.Description); err != nil {
			return nil, err
		}
		if e.Timestamp, err = database.ParseTime(ts); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Event, error) {
	return r.query(ctx, selectEvents+` ORDER BY timestamp`)
}

func (r *SQLiteRepository) ListByMeter(ctx context.Context, meterID string) ([]Event, error) {
	return r.query(ctx, selectEvents+` WHERE meter_id = ? ORDER BY timestamp`, meterID)
}

func (r *SQLiteRepository) InsertMany(ctx context.Context, events []Event) error {
	return database.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO events (meter_id, timestamp, type, description) VALUES (?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, e := range events {
			if _, err := stmt.ExecContext(ctx, e.MeterID, database.FormatTime(e.Timestamp), e.Type, e.Description); err != nil {
				return err
			}
		}
		return nil
	})
}
