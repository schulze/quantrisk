package store

import (
	"fmt"

	"github.com/schulze/quantrisk/internal/model"
)

func (s *Store) ListControls() ([]model.Control, error) {
	rows, err := s.DB.Query(`SELECT id, identifier, name, description, status, created_at, updated_at FROM controls ORDER BY identifier`)
	if err != nil {
		return nil, fmt.Errorf("list controls: %w", err)
	}
	defer rows.Close()

	var items []model.Control
	for rows.Next() {
		var c model.Control
		if err := rows.Scan(&c.ID, &c.Identifier, &c.Name, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan control: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (s *Store) GetControl(id int64) (model.Control, error) {
	var c model.Control
	err := s.DB.QueryRow(`SELECT id, identifier, name, description, status, created_at, updated_at FROM controls WHERE id = ?`, id).
		Scan(&c.ID, &c.Identifier, &c.Name, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("get control %d: %w", id, err)
	}
	return c, nil
}

func (s *Store) CreateControl(c *model.Control) error {
	res, err := s.DB.Exec(
		`INSERT INTO controls (identifier, name, description, status) VALUES (?, ?, ?, ?)`,
		c.Identifier, c.Name, c.Description, c.Status,
	)
	if err != nil {
		return fmt.Errorf("create control: %w", err)
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateControl(c *model.Control) error {
	_, err := s.DB.Exec(
		`UPDATE controls SET identifier = ?, name = ?, description = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		c.Identifier, c.Name, c.Description, c.Status, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update control %d: %w", c.ID, err)
	}
	return nil
}

func (s *Store) CountControls() (int64, error) {
	var count int64
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM controls`).Scan(&count)
	return count, err
}

func (s *Store) DeleteControl(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM controls WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete control %d: %w", id, err)
	}
	return nil
}

// Link/unlink methods for controls

func (s *Store) LinkControlRisk(controlID, riskID int64) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO control_risks (control_id, risk_id) VALUES (?, ?)`, controlID, riskID)
	return err
}

func (s *Store) UnlinkControlRisk(controlID, riskID int64) error {
	_, err := s.DB.Exec(`DELETE FROM control_risks WHERE control_id = ? AND risk_id = ?`, controlID, riskID)
	return err
}

func (s *Store) ListControlRisks(controlID int64) ([]model.Risk, error) {
	rows, err := s.DB.Query(`SELECT `+riskColumns+`
		FROM risks WHERE id IN (SELECT risk_id FROM control_risks WHERE control_id = ?) ORDER BY identifier`, controlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Risk
	for rows.Next() {
		r, err := scanRisk(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (s *Store) LinkControlRequirement(controlID, requirementID int64) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO control_requirements (control_id, requirement_id) VALUES (?, ?)`, controlID, requirementID)
	return err
}

func (s *Store) UnlinkControlRequirement(controlID, requirementID int64) error {
	_, err := s.DB.Exec(`DELETE FROM control_requirements WHERE control_id = ? AND requirement_id = ?`, controlID, requirementID)
	return err
}

func (s *Store) ListControlRequirements(controlID int64) ([]model.Requirement, error) {
	rows, err := s.DB.Query(`SELECT r.id, r.identifier, r.name, r.description, r.source, r.created_at, r.updated_at
		FROM requirements r JOIN control_requirements cr ON r.id = cr.requirement_id WHERE cr.control_id = ? ORDER BY r.identifier`, controlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Requirement
	for rows.Next() {
		var r model.Requirement
		if err := rows.Scan(&r.ID, &r.Identifier, &r.Name, &r.Description, &r.Source, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}
