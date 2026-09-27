package meter

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database"
)

var ErrNotFound = errors.New("meter not found")

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

const selectMeters = `SELECT id, meter_id, name, location, status, created_at FROM meters`

func scanMeter(row interface{ Scan(...any) error }) (Meter, error) {
	var m Meter
	var createdAt string
	if err := row.Scan(&m.ID, &m.MeterID, &m.Name, &m.Location, &m.Status, &createdAt); err != nil {
		return Meter{}, err
	}
	ts, err := database.ParseTime(createdAt)
	if err != nil {
		return Meter{}, err
	}
	m.CreatedAt = ts
	return m, nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Meter, error) {
	rows, err := r.db.QueryContext(ctx, selectMeters+` ORDER BY meter_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Meter
	for rows.Next() {
		m, err := scanMeter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) Get(ctx context.Context, meterID string) (Meter, error) {
	m, err := scanMeter(r.db.QueryRowContext(ctx, selectMeters+` WHERE meter_id = ?`, meterID))
	if errors.Is(err, sql.ErrNoRows) {
		return Meter{}, ErrNotFound
	}
	return m, err
}

func (r *SQLiteRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM meters`).Scan(&n)
	return n, err
}

func (r *SQLiteRepository) InsertMany(ctx context.Context, meters []Meter) error {
	now := database.FormatTime(time.Now())
	return database.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO meters (meter_id, name, location, status, created_at) VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, m := range meters {
			status := m.Status
			if status == "" {
				status = StatusPending
			}
			if _, err := stmt.ExecContext(ctx, m.MeterID, m.Name, m.Location, status, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SQLiteRepository) UpdateStatus(ctx context.Context, meterID string, status Status) error {
	res, err := r.db.ExecContext(ctx, `UPDATE meters SET status = ? WHERE meter_id = ?`, status, meterID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
