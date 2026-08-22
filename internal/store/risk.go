package store

import (
	"fmt"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/internal/model"
)

const riskColumns = `id, identifier, scenario,
	lef_mode,
	lef_min, lef_ml, lef_max, lef_rationale,
	tef_min, tef_ml, tef_max, tef_rationale,
	susc_min, susc_ml, susc_max, susc_rationale,
	pl_prodl_min, pl_prodl_ml, pl_prodl_max, pl_prodl_rationale,
	pl_respc_min, pl_respc_ml, pl_respc_max, pl_respc_rationale,
	pl_replc_min, pl_replc_ml, pl_replc_max, pl_replc_rationale,
	pl_finju_min, pl_finju_ml, pl_finju_max, pl_finju_rationale,
	pl_repud_min, pl_repud_ml, pl_repud_max, pl_repud_rationale,
	pl_cadvl_min, pl_cadvl_ml, pl_cadvl_max, pl_cadvl_rationale,
	sl_prodl_min, sl_prodl_ml, sl_prodl_max, sl_prodl_rationale,
	sl_respc_min, sl_respc_ml, sl_respc_max, sl_respc_rationale,
	sl_replc_min, sl_replc_ml, sl_replc_max, sl_replc_rationale,
	sl_finju_min, sl_finju_ml, sl_finju_max, sl_finju_rationale,
	sl_repud_min, sl_repud_ml, sl_repud_max, sl_repud_rationale,
	sl_cadvl_min, sl_cadvl_ml, sl_cadvl_max, sl_cadvl_rationale,
	created_at, updated_at`

func scanEstimate(ptrs *[4]any, e *fair.Estimate) {
	ptrs[0] = &e.Min
	ptrs[1] = &e.ML
	ptrs[2] = &e.Max
	ptrs[3] = &e.Rationale
}

func scanRisk(scanner interface{ Scan(...any) error }) (model.Risk, error) {
	var r model.Risk
	// 3 meta + 1 lef_mode + 15×4 estimates + 2 timestamps = 66
	var ptrs [66]any
	ptrs[0] = &r.ID
	ptrs[1] = &r.Identifier
	ptrs[2] = &r.Scenario
	ptrs[3] = &r.LEFMode

	var ep [15][4]any
	estimates := []*fair.Estimate{
		&r.DirectLEF, &r.TEF, &r.Susc,
		&r.PL.ProdL, &r.PL.RespC, &r.PL.ReplC, &r.PL.FinJu, &r.PL.RepuD, &r.PL.CAdvL,
		&r.SL.ProdL, &r.SL.RespC, &r.SL.ReplC, &r.SL.FinJu, &r.SL.RepuD, &r.SL.CAdvL,
	}
	for i, e := range estimates {
		scanEstimate(&ep[i], e)
		for j := 0; j < 4; j++ {
			ptrs[4+i*4+j] = ep[i][j]
		}
	}
	ptrs[64] = &r.CreatedAt
	ptrs[65] = &r.UpdatedAt

	err := scanner.Scan(ptrs[:]...)
	return r, err
}

func estimateArgs(e fair.Estimate) []any {
	return []any{e.Min, e.ML, e.Max, e.Rationale}
}

func riskArgs(r *model.Risk) []any {
	var args []any
	args = append(args, r.Identifier, r.Scenario, r.LEFMode.String())
	for _, e := range []fair.Estimate{
		r.DirectLEF, r.TEF, r.Susc,
		r.PL.ProdL, r.PL.RespC, r.PL.ReplC, r.PL.FinJu, r.PL.RepuD, r.PL.CAdvL,
		r.SL.ProdL, r.SL.RespC, r.SL.ReplC, r.SL.FinJu, r.SL.RepuD, r.SL.CAdvL,
	} {
		args = append(args, estimateArgs(e)...)
	}
	return args
}

func (s *Store) ListRisks() ([]model.Risk, error) {
	rows, err := s.DB.Query(`SELECT ` + riskColumns + ` FROM risks ORDER BY identifier`)
	if err != nil {
		return nil, fmt.Errorf("list risks: %w", err)
	}
	defer rows.Close()

	var risks []model.Risk
	for rows.Next() {
		r, err := scanRisk(rows)
		if err != nil {
			return nil, fmt.Errorf("scan risk: %w", err)
		}
		risks = append(risks, r)
	}
	return risks, rows.Err()
}

func (s *Store) GetRisk(id int64) (model.Risk, error) {
	r, err := scanRisk(s.DB.QueryRow(`SELECT `+riskColumns+` FROM risks WHERE id = ?`, id))
	if err != nil {
		return r, fmt.Errorf("get risk %d: %w", id, err)
	}
	return r, nil
}

func (s *Store) GetRiskByIdentifier(identifier string) (model.Risk, error) {
	r, err := scanRisk(s.DB.QueryRow(`SELECT `+riskColumns+` FROM risks WHERE identifier = ?`, identifier))
	if err != nil {
		return r, fmt.Errorf("get risk by identifier %q: %w", identifier, err)
	}
	return r, nil
}

func (s *Store) CreateRisk(r *model.Risk) error {
	args := riskArgs(r)
	// 3 + 15*4 = 63 placeholders
	res, err := s.DB.Exec(
		`INSERT INTO risks (identifier, scenario, lef_mode,
			lef_min, lef_ml, lef_max, lef_rationale,
			tef_min, tef_ml, tef_max, tef_rationale,
			susc_min, susc_ml, susc_max, susc_rationale,
			pl_prodl_min, pl_prodl_ml, pl_prodl_max, pl_prodl_rationale,
			pl_respc_min, pl_respc_ml, pl_respc_max, pl_respc_rationale,
			pl_replc_min, pl_replc_ml, pl_replc_max, pl_replc_rationale,
			pl_finju_min, pl_finju_ml, pl_finju_max, pl_finju_rationale,
			pl_repud_min, pl_repud_ml, pl_repud_max, pl_repud_rationale,
			pl_cadvl_min, pl_cadvl_ml, pl_cadvl_max, pl_cadvl_rationale,
			sl_prodl_min, sl_prodl_ml, sl_prodl_max, sl_prodl_rationale,
			sl_respc_min, sl_respc_ml, sl_respc_max, sl_respc_rationale,
			sl_replc_min, sl_replc_ml, sl_replc_max, sl_replc_rationale,
			sl_finju_min, sl_finju_ml, sl_finju_max, sl_finju_rationale,
			sl_repud_min, sl_repud_ml, sl_repud_max, sl_repud_rationale,
			sl_cadvl_min, sl_cadvl_ml, sl_cadvl_max, sl_cadvl_rationale
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		args...)
	if err != nil {
		return fmt.Errorf("create risk: %w", err)
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateRisk(r *model.Risk) error {
	args := riskArgs(r)
	args = append(args, r.ID)
	_, err := s.DB.Exec(
		`UPDATE risks SET identifier=?, scenario=?, lef_mode=?,
			lef_min=?, lef_ml=?, lef_max=?, lef_rationale=?,
			tef_min=?, tef_ml=?, tef_max=?, tef_rationale=?,
			susc_min=?, susc_ml=?, susc_max=?, susc_rationale=?,
			pl_prodl_min=?, pl_prodl_ml=?, pl_prodl_max=?, pl_prodl_rationale=?,
			pl_respc_min=?, pl_respc_ml=?, pl_respc_max=?, pl_respc_rationale=?,
			pl_replc_min=?, pl_replc_ml=?, pl_replc_max=?, pl_replc_rationale=?,
			pl_finju_min=?, pl_finju_ml=?, pl_finju_max=?, pl_finju_rationale=?,
			pl_repud_min=?, pl_repud_ml=?, pl_repud_max=?, pl_repud_rationale=?,
			pl_cadvl_min=?, pl_cadvl_ml=?, pl_cadvl_max=?, pl_cadvl_rationale=?,
			sl_prodl_min=?, sl_prodl_ml=?, sl_prodl_max=?, sl_prodl_rationale=?,
			sl_respc_min=?, sl_respc_ml=?, sl_respc_max=?, sl_respc_rationale=?,
			sl_replc_min=?, sl_replc_ml=?, sl_replc_max=?, sl_replc_rationale=?,
			sl_finju_min=?, sl_finju_ml=?, sl_finju_max=?, sl_finju_rationale=?,
			sl_repud_min=?, sl_repud_ml=?, sl_repud_max=?, sl_repud_rationale=?,
			sl_cadvl_min=?, sl_cadvl_ml=?, sl_cadvl_max=?, sl_cadvl_rationale=?,
			updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		args...)
	if err != nil {
		return fmt.Errorf("update risk %d: %w", r.ID, err)
	}
	return nil
}

func (s *Store) CountRisks() (int64, error) {
	var count int64
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM risks`).Scan(&count)
	return count, err
}

func (s *Store) DeleteRisk(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM risks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete risk %d: %w", id, err)
	}
	return nil
}

// ListRiskControls returns controls linked to a risk (reverse lookup).
func (s *Store) ListRiskControls(riskID int64) ([]model.Control, error) {
	rows, err := s.DB.Query(`SELECT c.id, c.identifier, c.name, c.description, c.status, c.created_at, c.updated_at
		FROM controls c JOIN control_risks cr ON c.id = cr.control_id WHERE cr.risk_id = ? ORDER BY c.identifier`, riskID)
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
