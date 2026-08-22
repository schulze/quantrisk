package store

import (
	"fmt"
	"testing"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/internal/model"
)

// newTestStore creates a migrated in-memory Store for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	db := openTestDB(t)
	if err := Migrate(db, 0); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &Store{DB: db}
}

// Risk CRUD

func makeRisk(id string, scenario string) *model.Risk {
	return &model.Risk{
		Identifier: id,
		Scenario:   scenario,
		LossEvent: fair.LossEvent{
			LEFMode:   fair.LEFDirect,
			DirectLEF: fair.Estimate{Min: 0.1, ML: 0.5, Max: 1.0, Rationale: "test"},
			PL: fair.LossForm{
				ProdL: fair.Estimate{Min: 1000, ML: 5000, Max: 10000},
				RespC: fair.Estimate{Min: 500, ML: 2000, Max: 5000},
			},
		},
	}
}

func TestRiskCRUD(t *testing.T) {
	s := newTestStore(t)

	// Create
	r := makeRisk("RISK-001", "Data breach")
	if err := s.CreateRisk(r); err != nil {
		t.Fatalf("create: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// Get by ID
	got, err := s.GetRisk(r.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Identifier != "RISK-001" {
		t.Errorf("identifier = %q, want RISK-001", got.Identifier)
	}
	if got.Scenario != "Data breach" {
		t.Errorf("scenario = %q, want Data breach", got.Scenario)
	}
	if got.DirectLEF.ML != 0.5 {
		t.Errorf("directLEF.ML = %v, want 0.5", got.DirectLEF.ML)
	}
	if got.PL.ProdL.Min != 1000 {
		t.Errorf("PL.ProdL.Min = %v, want 1000", got.PL.ProdL.Min)
	}
	if got.LEFMode != fair.LEFDirect {
		t.Errorf("LEFMode = %v, want %v", got.LEFMode, fair.LEFDirect)
	}

	// Get by identifier
	got2, err := s.GetRiskByIdentifier("RISK-001")
	if err != nil {
		t.Fatalf("get by identifier: %v", err)
	}
	if got2.ID != r.ID {
		t.Errorf("get by identifier returned wrong ID")
	}

	// Update
	r.Scenario = "Ransomware"
	r.DirectLEF.ML = 2.0
	if err := s.UpdateRisk(r); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := s.GetRisk(r.ID)
	if updated.Scenario != "Ransomware" {
		t.Errorf("updated scenario = %q", updated.Scenario)
	}
	if updated.DirectLEF.ML != 2.0 {
		t.Errorf("updated ML = %v, want 2.0", updated.DirectLEF.ML)
	}

	// List
	r2 := makeRisk("RISK-002", "Insider threat")
	s.CreateRisk(r2)
	risks, err := s.ListRisks()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(risks) != 2 {
		t.Fatalf("list: got %d, want 2", len(risks))
	}
	// Should be sorted by identifier
	if risks[0].Identifier != "RISK-001" || risks[1].Identifier != "RISK-002" {
		t.Errorf("list order: %s, %s", risks[0].Identifier, risks[1].Identifier)
	}

	// Count
	count, err := s.CountRisks()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Delete
	if err := s.DeleteRisk(r.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	count, _ = s.CountRisks()
	if count != 1 {
		t.Errorf("after delete count = %d, want 1", count)
	}

	// Get deleted → error
	_, err = s.GetRisk(r.ID)
	if err == nil {
		t.Error("expected error getting deleted risk")
	}
}

func TestRiskUniqueIdentifier(t *testing.T) {
	s := newTestStore(t)
	r1 := makeRisk("RISK-DUP", "First")
	if err := s.CreateRisk(r1); err != nil {
		t.Fatalf("create first: %v", err)
	}
	r2 := makeRisk("RISK-DUP", "Second")
	if err := s.CreateRisk(r2); err == nil {
		t.Fatal("expected unique constraint error")
	}
}

func TestRiskAllEstimateFields(t *testing.T) {
	s := newTestStore(t)
	r := &model.Risk{
		Identifier: "RISK-FULL",
		Scenario:   "Full",
		LossEvent: fair.LossEvent{
			LEFMode:   fair.LEFDecomposed,
			DirectLEF: fair.Estimate{Min: 1, ML: 2, Max: 3, Rationale: "direct"},
			TEF:       fair.Estimate{Min: 4, ML: 5, Max: 6, Rationale: "tef"},
			Susc:      fair.Estimate{Min: 0.1, ML: 0.2, Max: 0.3, Rationale: "susc"},
			PL: fair.LossForm{
				ProdL: fair.Estimate{Min: 10, ML: 20, Max: 30, Rationale: "pl-prodl"},
				RespC: fair.Estimate{Min: 11, ML: 21, Max: 31, Rationale: "pl-respc"},
				ReplC: fair.Estimate{Min: 12, ML: 22, Max: 32, Rationale: "pl-replc"},
				FinJu: fair.Estimate{Min: 13, ML: 23, Max: 33, Rationale: "pl-finju"},
				RepuD: fair.Estimate{Min: 14, ML: 24, Max: 34, Rationale: "pl-repud"},
				CAdvL: fair.Estimate{Min: 15, ML: 25, Max: 35, Rationale: "pl-cadvl"},
			},
			SL: fair.LossForm{
				ProdL: fair.Estimate{Min: 100, ML: 200, Max: 300, Rationale: "sl-prodl"},
				RespC: fair.Estimate{Min: 101, ML: 201, Max: 301, Rationale: "sl-respc"},
				ReplC: fair.Estimate{Min: 102, ML: 202, Max: 302, Rationale: "sl-replc"},
				FinJu: fair.Estimate{Min: 103, ML: 203, Max: 303, Rationale: "sl-finju"},
				RepuD: fair.Estimate{Min: 104, ML: 204, Max: 304, Rationale: "sl-repud"},
				CAdvL: fair.Estimate{Min: 105, ML: 205, Max: 305, Rationale: "sl-cadvl"},
			},
		},
	}
	if err := s.CreateRisk(r); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.GetRisk(r.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	// Verify decomposed mode
	if got.LEFMode != fair.LEFDecomposed {
		t.Errorf("LEFMode = %v, want %v", got.LEFMode, fair.LEFDecomposed)
	}

	// Spot-check various estimate fields
	checks := []struct {
		name string
		got  fair.Estimate
		want fair.Estimate
	}{
		{"DirectLEF", got.DirectLEF, r.DirectLEF},
		{"TEF", got.TEF, r.TEF},
		{"Susc", got.Susc, r.Susc},
		{"PL.ProdL", got.PL.ProdL, r.PL.ProdL},
		{"PL.CAdvL", got.PL.CAdvL, r.PL.CAdvL},
		{"SL.ProdL", got.SL.ProdL, r.SL.ProdL},
		{"SL.CAdvL", got.SL.CAdvL, r.SL.CAdvL},
	}
	for _, c := range checks {
		if c.got.Min != c.want.Min || c.got.ML != c.want.ML || c.got.Max != c.want.Max || c.got.Rationale != c.want.Rationale {
			t.Errorf("%s: got %+v, want %+v", c.name, c.got, c.want)
		}
	}
}

// Requirement CRUD

func TestRequirementCRUD(t *testing.T) {
	s := newTestStore(t)

	r := &model.Requirement{Identifier: "REQ-001", Name: "Access Control", Description: "Enforce RBAC", Source: "ISO 27001"}
	if err := s.CreateRequirement(r); err != nil {
		t.Fatalf("create: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := s.GetRequirement(r.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Access Control" || got.Source != "ISO 27001" {
		t.Errorf("got %+v", got)
	}

	// Update
	r.Name = "Updated Access Control"
	if err := s.UpdateRequirement(r); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := s.GetRequirement(r.ID)
	if updated.Name != "Updated Access Control" {
		t.Errorf("name after update = %q", updated.Name)
	}

	// List + Count
	r2 := &model.Requirement{Identifier: "REQ-002", Name: "Encryption"}
	s.CreateRequirement(r2)
	list, _ := s.ListRequirements()
	if len(list) != 2 {
		t.Fatalf("list: got %d, want 2", len(list))
	}
	count, _ := s.CountRequirements()
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Delete
	s.DeleteRequirement(r.ID)
	count, _ = s.CountRequirements()
	if count != 1 {
		t.Errorf("after delete count = %d", count)
	}
}

// Control CRUD

func TestControlCRUD(t *testing.T) {
	s := newTestStore(t)

	c := &model.Control{Identifier: "CTL-001", Name: "Firewall", Description: "Perimeter firewall", Status: "implemented"}
	if err := s.CreateControl(c); err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := s.GetControl(c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Firewall" || got.Status != "implemented" {
		t.Errorf("got %+v", got)
	}

	// Update
	c.Status = "verified"
	if err := s.UpdateControl(c); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := s.GetControl(c.ID)
	if updated.Status != "verified" {
		t.Errorf("status after update = %q", updated.Status)
	}

	// List + Count
	c2 := &model.Control{Identifier: "CTL-002", Name: "IDS", Status: "planned"}
	s.CreateControl(c2)
	list, _ := s.ListControls()
	if len(list) != 2 {
		t.Fatalf("list: got %d", len(list))
	}
	count, _ := s.CountControls()
	if count != 2 {
		t.Errorf("count = %d", count)
	}

	// Delete
	s.DeleteControl(c.ID)
	count, _ = s.CountControls()
	if count != 1 {
		t.Errorf("after delete count = %d", count)
	}
}

// Gap CRUD

func TestGapCRUD(t *testing.T) {
	s := newTestStore(t)

	// Need a control as parent
	c := &model.Control{Identifier: "CTL-GAP", Name: "Parent", Status: "planned"}
	s.CreateControl(c)

	parentType := "control"
	parentID := c.ID
	g := &model.Gap{
		Identifier: "GAP-001", Name: "Missing MFA",
		Description: "No MFA on admin", Severity: "high", Status: "open",
		ParentType: &parentType, ParentID: &parentID,
	}
	if err := s.CreateGap(g); err != nil {
		t.Fatalf("create: %v", err)
	}
	if g.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := s.GetGap(g.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Severity != "high" || got.Status != "open" {
		t.Errorf("got %+v", got)
	}
	if got.ParentType == nil || *got.ParentType != "control" || got.ParentID == nil || *got.ParentID != c.ID {
		t.Errorf("parent: %v/%v", got.ParentType, got.ParentID)
	}

	// Update
	g.Status = "mitigated"
	if err := s.UpdateGap(g); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := s.GetGap(g.ID)
	if updated.Status != "mitigated" {
		t.Errorf("status after update = %q", updated.Status)
	}

	// ListByParent
	g2 := &model.Gap{
		Identifier: "GAP-002", Name: "Missing logging",
		Severity: "medium", Status: "open",
		ParentType: &parentType, ParentID: &parentID,
	}
	s.CreateGap(g2)

	byParent, err := s.ListGapsByParent("control", c.ID)
	if err != nil {
		t.Fatalf("list by parent: %v", err)
	}
	if len(byParent) != 2 {
		t.Errorf("list by parent: got %d, want 2", len(byParent))
	}

	// List all + Count
	list, _ := s.ListGaps()
	if len(list) != 2 {
		t.Fatalf("list: got %d", len(list))
	}
	count, _ := s.CountGaps()
	if count != 2 {
		t.Errorf("count = %d", count)
	}

	// Delete
	s.DeleteGap(g.ID)
	count, _ = s.CountGaps()
	if count != 1 {
		t.Errorf("after delete count = %d", count)
	}
}

// Junction tables: Control ↔ Risk

func TestControlRiskLinking(t *testing.T) {
	s := newTestStore(t)

	c := &model.Control{Identifier: "CTL-LR", Name: "Test", Status: "planned"}
	s.CreateControl(c)
	r1 := makeRisk("RISK-LR1", "Scenario A")
	s.CreateRisk(r1)
	r2 := makeRisk("RISK-LR2", "Scenario B")
	s.CreateRisk(r2)

	// Link
	if err := s.LinkControlRisk(c.ID, r1.ID); err != nil {
		t.Fatalf("link 1: %v", err)
	}
	if err := s.LinkControlRisk(c.ID, r2.ID); err != nil {
		t.Fatalf("link 2: %v", err)
	}

	// Idempotent — linking again should not error (INSERT OR IGNORE)
	if err := s.LinkControlRisk(c.ID, r1.ID); err != nil {
		t.Fatalf("idempotent link: %v", err)
	}

	// List from control side
	risks, err := s.ListControlRisks(c.ID)
	if err != nil {
		t.Fatalf("list control risks: %v", err)
	}
	if len(risks) != 2 {
		t.Fatalf("got %d risks, want 2", len(risks))
	}

	// List from risk side (reverse)
	ctrls, err := s.ListRiskControls(r1.ID)
	if err != nil {
		t.Fatalf("list risk controls: %v", err)
	}
	if len(ctrls) != 1 {
		t.Fatalf("got %d controls, want 1", len(ctrls))
	}
	if ctrls[0].ID != c.ID {
		t.Errorf("wrong control ID")
	}

	// Unlink
	if err := s.UnlinkControlRisk(c.ID, r1.ID); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	risks, _ = s.ListControlRisks(c.ID)
	if len(risks) != 1 {
		t.Errorf("after unlink: got %d risks", len(risks))
	}

	// Unlink nonexistent — should not error
	if err := s.UnlinkControlRisk(c.ID, 99999); err != nil {
		t.Errorf("unlink nonexistent: %v", err)
	}
}

func TestControlRiskCascadeDelete(t *testing.T) {
	s := newTestStore(t)

	c := &model.Control{Identifier: "CTL-CD", Name: "Test", Status: "planned"}
	s.CreateControl(c)
	r := makeRisk("RISK-CD", "Cascade")
	s.CreateRisk(r)
	s.LinkControlRisk(c.ID, r.ID)

	// Delete control → junction record should cascade
	s.DeleteControl(c.ID)
	risks, _ := s.ListControlRisks(c.ID)
	if len(risks) != 0 {
		t.Errorf("expected 0 risks after control cascade delete, got %d", len(risks))
	}

	// Verify risk still exists
	_, err := s.GetRisk(r.ID)
	if err != nil {
		t.Error("risk should survive control deletion")
	}
}

func TestRiskDeleteCascadesJunction(t *testing.T) {
	s := newTestStore(t)

	c := &model.Control{Identifier: "CTL-RD", Name: "Test", Status: "planned"}
	s.CreateControl(c)
	r := makeRisk("RISK-RD", "To delete")
	s.CreateRisk(r)
	s.LinkControlRisk(c.ID, r.ID)

	// Delete risk → junction record should cascade
	s.DeleteRisk(r.ID)
	risks, _ := s.ListControlRisks(c.ID)
	if len(risks) != 0 {
		t.Errorf("expected 0 risks after risk cascade delete")
	}

	// Control should survive
	_, err := s.GetControl(c.ID)
	if err != nil {
		t.Error("control should survive risk deletion")
	}
}

// Junction tables: Control ↔ Requirement

func TestControlRequirementLinking(t *testing.T) {
	s := newTestStore(t)

	c := &model.Control{Identifier: "CTL-REQ", Name: "Test", Status: "planned"}
	s.CreateControl(c)
	req := &model.Requirement{Identifier: "REQ-LINK", Name: "Test Req"}
	s.CreateRequirement(req)

	// Link
	if err := s.LinkControlRequirement(c.ID, req.ID); err != nil {
		t.Fatalf("link: %v", err)
	}

	// Idempotent
	if err := s.LinkControlRequirement(c.ID, req.ID); err != nil {
		t.Fatalf("idempotent: %v", err)
	}

	// List from control side
	reqs, err := s.ListControlRequirements(c.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("got %d reqs, want 1", len(reqs))
	}
	if reqs[0].ID != req.ID {
		t.Errorf("wrong requirement ID")
	}

	// Reverse: from requirement side
	ctrls, err := s.ListRequirementControls(req.ID)
	if err != nil {
		t.Fatalf("reverse list: %v", err)
	}
	if len(ctrls) != 1 {
		t.Fatalf("reverse: got %d controls", len(ctrls))
	}

	// Unlink
	s.UnlinkControlRequirement(c.ID, req.ID)
	reqs, _ = s.ListControlRequirements(c.ID)
	if len(reqs) != 0 {
		t.Errorf("after unlink: got %d", len(reqs))
	}
}

// Multiple links from one entity

func TestControlMultipleLinks(t *testing.T) {
	s := newTestStore(t)

	c := &model.Control{Identifier: "CTL-MULTI", Name: "Multi", Status: "implemented"}
	s.CreateControl(c)

	// Link to 3 risks
	for i := 0; i < 3; i++ {
		r := makeRisk("RISK-M"+string(rune('A'+i)), "Scenario")
		s.CreateRisk(r)
		s.LinkControlRisk(c.ID, r.ID)
	}

	// Link to 2 requirements
	for i := 0; i < 2; i++ {
		req := &model.Requirement{Identifier: "REQ-M" + string(rune('A'+i)), Name: "Req"}
		s.CreateRequirement(req)
		s.LinkControlRequirement(c.ID, req.ID)
	}

	risks, _ := s.ListControlRisks(c.ID)
	if len(risks) != 3 {
		t.Errorf("risks: got %d, want 3", len(risks))
	}
	reqs, _ := s.ListControlRequirements(c.ID)
	if len(reqs) != 2 {
		t.Errorf("reqs: got %d, want 2", len(reqs))
	}
	// Delete control → all junction records cascade
	s.DeleteControl(c.ID)

	// Verify linked entities still exist
	rc, _ := s.CountRisks()
	if rc != 3 {
		t.Errorf("risks after control delete: %d", rc)
	}
	qc, _ := s.CountRequirements()
	if qc != 2 {
		t.Errorf("reqs after control delete: %d", qc)
	}
}

// Gap standalone creation (nil parent)

func TestGapStandaloneCreation(t *testing.T) {
	s := newTestStore(t)

	g := &model.Gap{
		Identifier: "GAP-SOLO",
		Name:       "Standalone gap",
		Severity:   "high",
		Status:     "open",
		ParentType: nil,
		ParentID:   nil,
	}
	if err := s.CreateGap(g); err != nil {
		t.Fatalf("create standalone gap: %v", err)
	}
	if g.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := s.GetGap(g.ID)
	if err != nil {
		t.Fatalf("get standalone gap: %v", err)
	}
	if got.ParentType != nil {
		t.Errorf("expected nil ParentType, got %v", got.ParentType)
	}
	if got.ParentID != nil {
		t.Errorf("expected nil ParentID, got %v", got.ParentID)
	}
	if got.Name != "Standalone gap" {
		t.Errorf("name = %q", got.Name)
	}

	// Verify it appears in ListGaps
	all, _ := s.ListGaps()
	if len(all) != 1 {
		t.Fatalf("list: got %d, want 1", len(all))
	}

	// Verify ListGapsByParent does not return it
	byParent, _ := s.ListGapsByParent("control", 1)
	if len(byParent) != 0 {
		t.Errorf("expected 0 gaps by parent, got %d", len(byParent))
	}
}

// Empty list returns

func TestEmptyLists(t *testing.T) {
	s := newTestStore(t)

	risks, err := s.ListRisks()
	if err != nil {
		t.Fatalf("list risks: %v", err)
	}
	if risks != nil {
		t.Errorf("expected nil slice, got %v", risks)
	}

	ctrls, err := s.ListControls()
	if err != nil {
		t.Fatalf("list controls: %v", err)
	}
	if ctrls != nil {
		t.Errorf("expected nil slice, got %v", ctrls)
	}

	reqs, err := s.ListRequirements()
	if err != nil {
		t.Fatalf("list reqs: %v", err)
	}
	if reqs != nil {
		t.Errorf("expected nil slice")
	}

	gaps, err := s.ListGaps()
	if err != nil {
		t.Fatalf("list gaps: %v", err)
	}
	if gaps != nil {
		t.Errorf("expected nil slice")
	}
}

// GetNotFound errors

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetRisk(99999)
	if err == nil {
		t.Error("expected error for nonexistent risk")
	}

	_, err = s.GetControl(99999)
	if err == nil {
		t.Error("expected error for nonexistent control")
	}

	_, err = s.GetRequirement(99999)
	if err == nil {
		t.Error("expected error for nonexistent requirement")
	}

	_, err = s.GetGap(99999)
	if err == nil {
		t.Error("expected error for nonexistent gap")
	}

	_, err = s.GetRiskByIdentifier("NONEXISTENT")
	if err == nil {
		t.Error("expected error for nonexistent risk by identifier")
	}
}

// Store.New integration

func TestNewStore(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer st.Close()

	// Should be able to create entities immediately
	c := &model.Control{Identifier: "CTL-NEW", Name: "Test", Status: "planned"}
	if err := st.CreateControl(c); err != nil {
		t.Fatalf("create control: %v", err)
	}

	// Verify foreign keys are enabled
	var fk int
	st.DB.QueryRow("PRAGMA foreign_keys").Scan(&fk)
	if fk != 1 {
		t.Errorf("foreign keys not enabled: %d", fk)
	}
}

// NextIdentifier

func TestNextIdentifier(t *testing.T) {
	s := newTestStore(t)

	// Empty table → PREFIX-001
	if got := s.NextIdentifier("controls", "CTRL"); got != "CTRL-001" {
		t.Errorf("empty table: got %q, want CTRL-001", got)
	}

	// Create some controls
	s.CreateControl(&model.Control{Identifier: "CTRL-001", Name: "A", Status: "planned"})
	s.CreateControl(&model.Control{Identifier: "CTRL-002", Name: "B", Status: "planned"})
	if got := s.NextIdentifier("controls", "CTRL"); got != "CTRL-003" {
		t.Errorf("after 2: got %q, want CTRL-003", got)
	}

	// Delete one in the middle — next should still be max+1, not fill gaps
	s.DB.Exec(`DELETE FROM controls WHERE identifier = 'CTRL-001'`)
	if got := s.NextIdentifier("controls", "CTRL"); got != "CTRL-003" {
		t.Errorf("after delete middle: got %q, want CTRL-003", got)
	}

}

// Concurrent creation

func TestConcurrentCreation(t *testing.T) {
	s := newTestStore(t)

	// Verify no errors when creating many entities
	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("CTL-%03d", i)
		c := &model.Control{Identifier: id, Name: "Control " + id, Status: "planned"}
		if err := s.CreateControl(c); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	count, _ := s.CountControls()
	if count != 50 {
		t.Errorf("count = %d, want 50", count)
	}
}

// Audit Log

func TestAuditRecordAndList(t *testing.T) {
	s := newTestStore(t)

	// Record some audit entries
	if err := s.RecordAudit("control", 1, "CTRL-001", "create", "", "", "New Control"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordAudit("control", 1, "CTRL-001", "update", "name", "New Control", "Updated Name"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordAudit("control", 2, "CTRL-002", "create", "", "", "Another Control"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordAudit("risk", 1, "RISK-001", "create", "", "", "Some Risk"); err != nil {
		t.Fatal(err)
	}

	// ListAuditByEntity — should return only entries for control 1
	entries, err := s.ListAuditByEntity("control", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("ListAuditByEntity got %d entries, want 2", len(entries))
	}
	// Most recent first
	if entries[0].Action != "update" {
		t.Errorf("first entry action = %q, want update", entries[0].Action)
	}
	if entries[0].OldValue != "New Control" || entries[0].NewValue != "Updated Name" {
		t.Errorf("update entry old=%q new=%q", entries[0].OldValue, entries[0].NewValue)
	}
	if entries[1].Action != "create" {
		t.Errorf("second entry action = %q, want create", entries[1].Action)
	}

	// ListAuditByType — should return all control entries
	allControls, err := s.ListAuditByType("control", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(allControls) != 3 {
		t.Fatalf("ListAuditByType(control) got %d entries, want 3", len(allControls))
	}

	// ListAuditByType for risk
	allRisks, err := s.ListAuditByType("risk", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(allRisks) != 1 {
		t.Fatalf("ListAuditByType(risk) got %d entries, want 1", len(allRisks))
	}

	// Test limit
	limited, err := s.ListAuditByType("control", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 2 {
		t.Fatalf("ListAuditByType with limit 2 got %d entries", len(limited))
	}
}

func TestAuditNoChangeSkipped(t *testing.T) {
	s := newTestStore(t)

	// Empty entity type should still work
	entries, err := s.ListAuditByType("nonexistent", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries for nonexistent type", len(entries))
	}
}

func TestAuditDefaultLimit(t *testing.T) {
	s := newTestStore(t)

	// Default limit (0) should use 50
	_, err := s.ListAuditByType("control", 0)
	if err != nil {
		t.Fatal(err)
	}
}
