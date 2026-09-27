package dbtest

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database"
)

func New(tb testing.TB) *sql.DB {
	tb.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		tb.Fatalf("open test database: %v", err)
	}
	tb.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db); err != nil {
		tb.Fatalf("migrate test database: %v", err)
	}
	return db
}
