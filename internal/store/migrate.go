package store

import (
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	Version int
	Name    string
	SQL     string
}

// MigrationStatus holds information about the current migration state.
type MigrationStatus struct {
	Current   int
	Available int
	Pending   []string
	Applied   []string
}

// Migrate applies all pending migrations up to targetVersion.
// If targetVersion is 0, all available migrations are applied.
func Migrate(db *sql.DB, targetVersion int) error {
	if err := ensureSchemaVersionTable(db); err != nil {
		return fmt.Errorf("ensure schema_version table: %w", err)
	}

	current, err := currentVersion(db)
	if err != nil {
		return fmt.Errorf("read current version: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	for _, m := range migrations {
		if m.Version <= current {
			continue
		}
		if targetVersion > 0 && m.Version > targetVersion {
			break
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec(m.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d (%s): %w", m.Version, m.Name, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_version (version, name) VALUES (?, ?)`,
			m.Version, m.Name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.Version, err)
		}
	}

	return nil
}

// GetMigrationStatus returns the current migration state of the database.
func GetMigrationStatus(db *sql.DB) (MigrationStatus, error) {
	if err := ensureSchemaVersionTable(db); err != nil {
		return MigrationStatus{}, err
	}

	current, err := currentVersion(db)
	if err != nil {
		return MigrationStatus{}, err
	}

	applied, err := appliedMigrations(db)
	if err != nil {
		return MigrationStatus{}, err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return MigrationStatus{}, err
	}

	var pending []string
	for _, m := range migrations {
		if m.Version > current {
			pending = append(pending, fmt.Sprintf("%03d_%s", m.Version, m.Name))
		}
	}

	available := 0
	if len(migrations) > 0 {
		available = migrations[len(migrations)-1].Version
	}

	return MigrationStatus{
		Current:   current,
		Available: available,
		Pending:   pending,
		Applied:   applied,
	}, nil
}

func ensureSchemaVersionTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func currentVersion(db *sql.DB) (int, error) {
	var v int
	err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&v)
	return v, err
}

func appliedMigrations(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT version, name, applied_at FROM schema_version ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applied []string
	for rows.Next() {
		var version int
		var name, appliedAt string
		if err := rows.Scan(&version, &name, &appliedAt); err != nil {
			return nil, err
		}
		applied = append(applied, fmt.Sprintf("%03d_%s (applied %s)", version, name, appliedAt))
	}
	return applied, rows.Err()
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var migrations []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		version, name, err := parseMigrationFilename(e.Name())
		if err != nil {
			return nil, fmt.Errorf("parse migration filename %q: %w", e.Name(), err)
		}

		data, err := migrationsFS.ReadFile(path.Join("migrations", e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", e.Name(), err)
		}

		migrations = append(migrations, migration{
			Version: version,
			Name:    name,
			SQL:     string(data),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFilename parses "001_initial.sql" into (1, "initial", nil).
func parseMigrationFilename(filename string) (int, string, error) {
	base := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("expected format NNN_name.sql, got %q", filename)
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid version number %q: %w", parts[0], err)
	}
	return version, parts[1], nil
}
