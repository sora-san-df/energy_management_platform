package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrateCreatesSchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, table := range []string{"meters", "readings", "events", "anomalies", "analyses", "schema_migrations"} {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second migrate should be a no-op: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected applied migrations to be recorded")
	}
}

func TestTimeRoundTrip(t *testing.T) {
	ts, err := ParseTime(FormatTime(mustTime(t, "2026-09-12T14:00:00Z")))
	if err != nil {
		t.Fatal(err)
	}
	if !ts.Equal(mustTime(t, "2026-09-12T14:00:00Z")) {
		t.Fatalf("round trip mismatch: %s", ts)
	}
}
