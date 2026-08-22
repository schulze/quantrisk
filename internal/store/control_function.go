package store

import (
	"fmt"

	"github.com/schulze/quantrisk/fair/cam"
	"github.com/schulze/quantrisk/internal/model"
)

const cfColumns = `id, control_id, function,
	cap_min, cap_ml, cap_max, cap_rationale,
	cov_min, cov_ml, cov_max, cov_rationale,
	rel_min, rel_ml, rel_max, rel_rationale,
	notes, created_at, updated_at`

func scanControlFunction(scanner interface{ Scan(...any) error }) (model.ControlFunction, error) {
	var cf model.ControlFunction
	err := scanner.Scan(
		&cf.ID, &cf.ControlID, &cf.Function,
		&cf.Effectiveness.Capability.Min, &cf.Effectiveness.Capability.ML,
		&cf.Effectiveness.Capability.Max, &cf.Effectiveness.Capability.Rationale,
		&cf.Effectiveness.Coverage.Min, &cf.Effectiveness.Coverage.ML,
		&cf.Effectiveness.Coverage.Max, &cf.Effectiveness.Coverage.Rationale,
		&cf.Effectiveness.Reliability.Min, &cf.Effectiveness.Reliability.ML,
		&cf.Effectiveness.Reliability.Max, &cf.Effectiveness.Reliability.Rationale,
		&cf.Notes, &cf.CreatedAt, &cf.UpdatedAt,
	)
	return cf, err
}

func cfArgs(cf *model.ControlFunction) []any {
	e := cf.Effectiveness
	return []any{
		cf.ControlID, string(cf.Function),
		e.Capability.Min, e.Capability.ML, e.Capability.Max, e.Capability.Rationale,
		e.Coverage.Min, e.Coverage.ML, e.Coverage.Max, e.Coverage.Rationale,
		e.Reliability.Min, e.Reliability.ML, e.Reliability.Max, e.Reliability.Rationale,
		cf.Notes,
	}
}

// ListControlFunctions returns all FAIR-CAM function assignments for a control.
func (s *Store) ListControlFunctions(controlID int64) ([]model.ControlFunction, error) {
	rows, err := s.DB.Query(
		`SELECT `+cfColumns+` FROM control_functions WHERE control_id = ? ORDER BY function`,
		controlID,
	)
	if err != nil {
		return nil, fmt.Errorf("list control functions: %w", err)
	}
	defer rows.Close()

	var items []model.ControlFunction
	for rows.Next() {
		cf, err := scanControlFunction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan control function: %w", err)
		}
		items = append(items, cf)
	}
	return items, rows.Err()
}

// GetControlFunction returns a single function assignment by ID.
func (s *Store) GetControlFunction(id int64) (model.ControlFunction, error) {
	cf, err := scanControlFunction(s.DB.QueryRow(
		`SELECT `+cfColumns+` FROM control_functions WHERE id = ?`, id,
	))
	if err != nil {
		return cf, fmt.Errorf("get control function %d: %w", id, err)
	}
	return cf, nil
}

// CreateControlFunction inserts a new FAIR-CAM function assignment.
func (s *Store) CreateControlFunction(cf *model.ControlFunction) error {
	// Validate the function exists in the CAM catalog.
	if _, ok := cam.Lookup(cf.Function); !ok {
		return fmt.Errorf("unknown FAIR-CAM function: %q", cf.Function)
	}
	args := cfArgs(cf)
	res, err := s.DB.Exec(
		`INSERT INTO control_functions (control_id, function,
			cap_min, cap_ml, cap_max, cap_rationale,
			cov_min, cov_ml, cov_max, cov_rationale,
			rel_min, rel_ml, rel_max, rel_rationale,
			notes
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		args...,
	)
	if err != nil {
		return fmt.Errorf("create control function: %w", err)
	}
	cf.ID, _ = res.LastInsertId()
	return nil
}

// UpdateControlFunction updates an existing function assignment.
func (s *Store) UpdateControlFunction(cf *model.ControlFunction) error {
	if _, ok := cam.Lookup(cf.Function); !ok {
		return fmt.Errorf("unknown FAIR-CAM function: %q", cf.Function)
	}
	e := cf.Effectiveness
	_, err := s.DB.Exec(
		`UPDATE control_functions SET function=?,
			cap_min=?, cap_ml=?, cap_max=?, cap_rationale=?,
			cov_min=?, cov_ml=?, cov_max=?, cov_rationale=?,
			rel_min=?, rel_ml=?, rel_max=?, rel_rationale=?,
			notes=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		string(cf.Function),
		e.Capability.Min, e.Capability.ML, e.Capability.Max, e.Capability.Rationale,
		e.Coverage.Min, e.Coverage.ML, e.Coverage.Max, e.Coverage.Rationale,
		e.Reliability.Min, e.Reliability.ML, e.Reliability.Max, e.Reliability.Rationale,
		cf.Notes, cf.ID,
	)
	if err != nil {
		return fmt.Errorf("update control function %d: %w", cf.ID, err)
	}
	return nil
}

// DeleteControlFunction removes a function assignment.
func (s *Store) DeleteControlFunction(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM control_functions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete control function %d: %w", id, err)
	}
	return nil
}

// ListFunctionsByCAMDomain returns all control function assignments in a given
// FAIR-CAM domain, across all controls. Useful for domain-wide views.
func (s *Store) ListFunctionsByCAMDomain(domain cam.Domain) ([]model.ControlFunction, error) {
	prefix := string(domain) + ".%"
	rows, err := s.DB.Query(
		`SELECT `+cfColumns+` FROM control_functions WHERE function LIKE ? ORDER BY function`,
		prefix,
	)
	if err != nil {
		return nil, fmt.Errorf("list functions by domain %q: %w", domain, err)
	}
	defer rows.Close()

	var items []model.ControlFunction
	for rows.Next() {
		cf, err := scanControlFunction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan control function: %w", err)
		}
		items = append(items, cf)
	}
	return items, rows.Err()
}

