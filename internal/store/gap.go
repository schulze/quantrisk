package store

import (
	"fmt"

	"github.com/schulze/quantrisk/internal/model"
)

func (s *Store) ListGaps() ([]model.Gap, error) {
	rows, err := s.DB.Query(`SELECT id, identifier, name, description, severity, status, parent_type, parent_id, created_at, updated_at FROM gaps ORDER BY identifier`)
	if err != nil {
		return nil, fmt.Errorf("list gaps: %w", err)
	}
	defer rows.Close()

	var items []model.Gap
	for rows.Next() {
		var g model.Gap
		if err := rows.Scan(&g.ID, &g.Identifier, &g.Name, &g.Description, &g.Severity, &g.Status, &g.ParentType, &g.ParentID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan gap: %w", err)
		}
		items = append(items, g)
	}
	return items, rows.Err()
}

func (s *Store) GetGap(id int64) (model.Gap, error) {
	var g model.Gap
	err := s.DB.QueryRow(`SELECT id, identifier, name, description, severity, status, parent_type, parent_id, created_at, updated_at FROM gaps WHERE id = ?`, id).
		Scan(&g.ID, &g.Identifier, &g.Name, &g.Description, &g.Severity, &g.Status, &g.ParentType, &g.ParentID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return g, fmt.Errorf("get gap %d: %w", id, err)
	}
	return g, nil
}

func (s *Store) CreateGap(g *model.Gap) error {
	res, err := s.DB.Exec(
		`INSERT INTO gaps (identifier, name, description, severity, status, parent_type, parent_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		g.Identifier, g.Name, g.Description, g.Severity, g.Status, g.ParentType, g.ParentID,
	)
	if err != nil {
		return fmt.Errorf("create gap: %w", err)
	}
	g.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateGap(g *model.Gap) error {
	_, err := s.DB.Exec(
		`UPDATE gaps SET identifier = ?, name = ?, description = ?, severity = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		g.Identifier, g.Name, g.Description, g.Severity, g.Status, g.ID,
	)
	if err != nil {
		return fmt.Errorf("update gap %d: %w", g.ID, err)
	}
	return nil
}

func (s *Store) CountGaps() (int64, error) {
	var count int64
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM gaps`).Scan(&count)
	return count, err
}

func (s *Store) DeleteGap(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM gaps WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete gap %d: %w", id, err)
	}
	return nil
}

func (s *Store) ListGapsByParent(parentType string, parentID int64) ([]model.Gap, error) {
	rows, err := s.DB.Query(`SELECT id, identifier, name, description, severity, status, parent_type, parent_id, created_at, updated_at FROM gaps WHERE parent_type = ? AND parent_id = ? ORDER BY identifier`, parentType, parentID)
	if err != nil {
		return nil, fmt.Errorf("list gaps by parent: %w", err)
	}
	defer rows.Close()

	var items []model.Gap
	for rows.Next() {
		var g model.Gap
		if err := rows.Scan(&g.ID, &g.Identifier, &g.Name, &g.Description, &g.Severity, &g.Status, &g.ParentType, &g.ParentID, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan gap: %w", err)
		}
		items = append(items, g)
	}
	return items, rows.Err()
}
