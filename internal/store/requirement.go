package store

import (
	"fmt"

	"github.com/schulze/quantrisk/internal/model"
)

func (s *Store) ListRequirements() ([]model.Requirement, error) {
	rows, err := s.DB.Query(`SELECT id, identifier, name, description, source, created_at, updated_at FROM requirements ORDER BY identifier`)
	if err != nil {
		return nil, fmt.Errorf("list requirements: %w", err)
	}
	defer rows.Close()

	var items []model.Requirement
	for rows.Next() {
		var r model.Requirement
		if err := rows.Scan(&r.ID, &r.Identifier, &r.Name, &r.Description, &r.Source, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan requirement: %w", err)
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (s *Store) GetRequirement(id int64) (model.Requirement, error) {
	var r model.Requirement
	err := s.DB.QueryRow(`SELECT id, identifier, name, description, source, created_at, updated_at FROM requirements WHERE id = ?`, id).
		Scan(&r.ID, &r.Identifier, &r.Name, &r.Description, &r.Source, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return r, fmt.Errorf("get requirement %d: %w", id, err)
	}
	return r, nil
}

func (s *Store) CreateRequirement(r *model.Requirement) error {
	res, err := s.DB.Exec(
		`INSERT INTO requirements (identifier, name, description, source) VALUES (?, ?, ?, ?)`,
		r.Identifier, r.Name, r.Description, r.Source,
	)
	if err != nil {
		return fmt.Errorf("create requirement: %w", err)
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateRequirement(r *model.Requirement) error {
	_, err := s.DB.Exec(
		`UPDATE requirements SET identifier = ?, name = ?, description = ?, source = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		r.Identifier, r.Name, r.Description, r.Source, r.ID,
	)
	if err != nil {
		return fmt.Errorf("update requirement %d: %w", r.ID, err)
	}
	return nil
}

func (s *Store) CountRequirements() (int64, error) {
	var count int64
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM requirements`).Scan(&count)
	return count, err
}

func (s *Store) DeleteRequirement(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM requirements WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete requirement %d: %w", id, err)
	}
	return nil
}

// ListRequirementControls returns controls linked to a requirement (reverse lookup).
func (s *Store) ListRequirementControls(requirementID int64) ([]model.Control, error) {
	rows, err := s.DB.Query(`SELECT c.id, c.identifier, c.name, c.description, c.status, c.created_at, c.updated_at
		FROM controls c JOIN control_requirements cr ON c.id = cr.control_id WHERE cr.requirement_id = ? ORDER BY c.identifier`, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Control
	for rows.Next() {
		var c model.Control
		if err := rows.Scan(&c.ID, &c.Identifier, &c.Name, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

