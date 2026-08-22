package store

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateFreshDB(t *testing.T) {
	db := openTestDB(t)

	if err := Migrate(db, 0); err != nil {
		t.Fatalf("migrate fresh db: %v", err)
	}

	// Verify schema_version was populated.
	var version int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected version 1, got %d", version)
	}

	// Verify a table from the initial migration exists.
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='risks'`).Scan(&name)
	if err != nil {
		t.Fatalf("risks table not found: %v", err)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db := openTestDB(t)

	if err := Migrate(db, 0); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := Migrate(db, 0); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	// Should only have recorded each migration once.
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if count != len(migrations) {
		t.Fatalf("expected %d version rows, got %d", len(migrations), count)
	}
}

func TestMigrateToSpecificVersion(t *testing.T) {
	db := openTestDB(t)

	// Migrate to version 1 only.
	if err := Migrate(db, 1); err != nil {
		t.Fatalf("migrate to v1: %v", err)
	}

	var version int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected version 1, got %d", version)
	}
}

func TestGetMigrationStatus(t *testing.T) {
	db := openTestDB(t)

	// Before any migration.
	st, err := GetMigrationStatus(db)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if st.Current != 0 {
		t.Fatalf("expected current=0, got %d", st.Current)
	}
	if st.Available < 1 {
		t.Fatalf("expected available >= 1, got %d", st.Available)
	}
	if len(st.Pending) == 0 {
		t.Fatal("expected pending migrations on empty db")
	}

	// After migration.
	if err := Migrate(db, 0); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err = GetMigrationStatus(db)
	if err != nil {
		t.Fatalf("get status after migrate: %v", err)
	}
	if st.Current != st.Available {
		t.Fatalf("expected current=%d, got %d", st.Available, st.Current)
	}
	if len(st.Pending) != 0 {
		t.Fatalf("expected no pending, got %d", len(st.Pending))
	}
	if len(st.Applied) == 0 {
		t.Fatal("expected applied migrations")
	}
}

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		filename    string
		wantVersion int
		wantName    string
		wantErr     bool
	}{
		{"001_initial.sql", 1, "initial", false},
		{"012_add_column.sql", 12, "add_column", false},
		{"bad.sql", 0, "", true},
		{"abc_name.sql", 0, "", true},
	}
	for _, tt := range tests {
		v, n, err := parseMigrationFilename(tt.filename)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseMigrationFilename(%q) expected error", tt.filename)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseMigrationFilename(%q) unexpected error: %v", tt.filename, err)
			continue
		}
		if v != tt.wantVersion || n != tt.wantName {
			t.Errorf("parseMigrationFilename(%q) = (%d, %q), want (%d, %q)",
				tt.filename, v, n, tt.wantVersion, tt.wantName)
		}
	}
}
