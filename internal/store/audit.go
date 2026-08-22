package store

import (
	"fmt"
	"time"
)

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID         int64
	EntityType string
	EntityID   int64
	Identifier string
	Action     string
	Field      string
	OldValue   string
	NewValue   string
	CreatedAt  time.Time
}

// RecordAudit inserts an audit log entry.
func (s *Store) RecordAudit(entityType string, entityID int64, identifier, action, field, oldValue, newValue string) error {
	_, err := s.DB.Exec(
		`INSERT INTO audit_log (entity_type, entity_id, identifier, action, field, old_value, new_value) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entityType, entityID, identifier, action, field, oldValue, newValue,
	)
	if err != nil {
		return fmt.Errorf("record audit: %w", err)
	}
	return nil
}

// ListAuditByEntity returns audit entries for a specific entity.
func (s *Store) ListAuditByEntity(entityType string, entityID int64) ([]AuditEntry, error) {
	rows, err := s.DB.Query(
		`SELECT id, entity_type, entity_id, identifier, action, field, old_value, new_value, created_at
		 FROM audit_log WHERE entity_type = ? AND entity_id = ? ORDER BY created_at DESC, id DESC`,
		entityType, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit by entity: %w", err)
	}
	defer rows.Close()
	return scanAuditEntries(rows)
}

// ListAuditByType returns recent audit entries for an entity type.
func (s *Store) ListAuditByType(entityType string, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.DB.Query(
		`SELECT id, entity_type, entity_id, identifier, action, field, old_value, new_value, created_at
		 FROM audit_log WHERE entity_type = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
		entityType, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit by type: %w", err)
	}
	defer rows.Close()
	return scanAuditEntries(rows)
}

func scanAuditEntries(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]AuditEntry, error) {
	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.Identifier, &e.Action, &e.Field, &e.OldValue, &e.NewValue, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
