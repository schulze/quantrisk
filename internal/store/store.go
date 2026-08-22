package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := Migrate(db, 0); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	s := &Store{DB: db}
	return s, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

// NextIdentifier returns the next sequential identifier for a given prefix.
// It scans existing identifiers matching "PREFIX-NNN" and returns PREFIX-(max+1).
func (s *Store) NextIdentifier(table, prefix string) string {
	// Extract max numeric suffix from identifiers matching the prefix pattern.
	var maxNum int64
	query := fmt.Sprintf(
		`SELECT COALESCE(MAX(CAST(SUBSTR(identifier, %d) AS INTEGER)), 0) FROM %s WHERE identifier LIKE ?`,
		len(prefix)+2, // skip "PREFIX-"
		table,
	)
	s.DB.QueryRow(query, prefix+"-%").Scan(&maxNum)
	return fmt.Sprintf("%s-%03d", prefix, maxNum+1)
}
