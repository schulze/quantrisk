package store

import (
	"testing"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/fair/cam"
	"github.com/schulze/quantrisk/internal/model"
)

func TestControlFunctionCRUD(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db, 0); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s := &Store{DB: db}

	// Create a control to attach functions to.
	ctrl := &model.Control{Identifier: "CTL-001", Name: "Firewall", Status: "implemented"}
	if err := s.CreateControl(ctrl); err != nil {
		t.Fatalf("create control: %v", err)
	}

	// Create a function assignment.
	cf := &model.ControlFunction{
		ControlID: ctrl.ID,
		Function:  cam.LECAvoidance,
		Effectiveness: cam.Effectiveness{
			Capability:  fair.Estimate{Min: 0.5, ML: 0.7, Max: 0.9, Rationale: "blocks most traffic"},
			Coverage:    fair.Estimate{Min: 0.8, ML: 0.9, Max: 1.0, Rationale: "all internet-facing"},
			Reliability: fair.Estimate{Min: 0.9, ML: 0.95, Max: 1.0, Rationale: "HA pair"},
		},
		Notes: "Network perimeter firewall",
	}
	if err := s.CreateControlFunction(cf); err != nil {
		t.Fatalf("create control function: %v", err)
	}
	if cf.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// Get it back.
	got, err := s.GetControlFunction(cf.ID)
	if err != nil {
		t.Fatalf("get control function: %v", err)
	}
	if got.Function != cam.LECAvoidance {
		t.Errorf("function = %q, want %q", got.Function, cam.LECAvoidance)
	}
	if got.Effectiveness.Capability.ML != 0.7 {
		t.Errorf("capability ML = %v, want 0.7", got.Effectiveness.Capability.ML)
	}
	if got.Notes != "Network perimeter firewall" {
		t.Errorf("notes = %q, want %q", got.Notes, "Network perimeter firewall")
	}

	// Add a second function (same control, different function).
	cf2 := &model.ControlFunction{
		ControlID: ctrl.ID,
		Function:  cam.LECDeterrence,
		Effectiveness: cam.Effectiveness{
			Capability:  fair.Estimate{Min: 0.2, ML: 0.4, Max: 0.6},
			Coverage:    fair.Estimate{Min: 0.5, ML: 0.7, Max: 0.9},
			Reliability: fair.Estimate{Min: 0.8, ML: 0.9, Max: 1.0},
		},
	}
	if err := s.CreateControlFunction(cf2); err != nil {
		t.Fatalf("create second function: %v", err)
	}

	// List functions.
	all, err := s.ListControlFunctions(ctrl.ID)
	if err != nil {
		t.Fatalf("list control functions: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d functions, want 2", len(all))
	}

	// Update the first function.
	cf.Effectiveness.Capability.ML = 0.8
	cf.Notes = "Updated notes"
	if err := s.UpdateControlFunction(cf); err != nil {
		t.Fatalf("update control function: %v", err)
	}
	updated, _ := s.GetControlFunction(cf.ID)
	if updated.Effectiveness.Capability.ML != 0.8 {
		t.Errorf("updated capability ML = %v, want 0.8", updated.Effectiveness.Capability.ML)
	}
	if updated.Notes != "Updated notes" {
		t.Errorf("updated notes = %q", updated.Notes)
	}

	// Delete one.
	if err := s.DeleteControlFunction(cf2.ID); err != nil {
		t.Fatalf("delete control function: %v", err)
	}
	after, _ := s.ListControlFunctions(ctrl.ID)
	if len(after) != 1 {
		t.Fatalf("after delete: got %d functions, want 1", len(after))
	}

	// List by domain.
	byDomain, err := s.ListFunctionsByCAMDomain(cam.DomainLEC)
	if err != nil {
		t.Fatalf("list by domain: %v", err)
	}
	if len(byDomain) != 1 {
		t.Fatalf("LEC domain: got %d, want 1", len(byDomain))
	}
}

func TestControlFunctionRejectsUnknown(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db, 0); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s := &Store{DB: db}

	ctrl := &model.Control{Identifier: "CTL-X", Name: "Test", Status: "planned"}
	if err := s.CreateControl(ctrl); err != nil {
		t.Fatalf("create control: %v", err)
	}

	cf := &model.ControlFunction{
		ControlID: ctrl.ID,
		Function:  cam.Function("INVALID.Function"),
	}
	if err := s.CreateControlFunction(cf); err == nil {
		t.Fatal("expected error for unknown function")
	}
}

func TestControlFunctionUniqueConstraint(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db, 0); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s := &Store{DB: db}

	ctrl := &model.Control{Identifier: "CTL-U", Name: "Test", Status: "planned"}
	if err := s.CreateControl(ctrl); err != nil {
		t.Fatalf("create control: %v", err)
	}

	cf1 := &model.ControlFunction{ControlID: ctrl.ID, Function: cam.LECResistance}
	if err := s.CreateControlFunction(cf1); err != nil {
		t.Fatalf("create first: %v", err)
	}

	// Same control + same function should fail (unique constraint).
	cf2 := &model.ControlFunction{ControlID: ctrl.ID, Function: cam.LECResistance}
	if err := s.CreateControlFunction(cf2); err == nil {
		t.Fatal("expected unique constraint error")
	}
}

func TestControlFunctionCascadeDelete(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db, 0); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s := &Store{DB: db}

	ctrl := &model.Control{Identifier: "CTL-DEL", Name: "Test", Status: "planned"}
	if err := s.CreateControl(ctrl); err != nil {
		t.Fatalf("create control: %v", err)
	}

	cf := &model.ControlFunction{ControlID: ctrl.ID, Function: cam.VMCControlMonitoring}
	if err := s.CreateControlFunction(cf); err != nil {
		t.Fatalf("create function: %v", err)
	}

	// Delete the control — functions should cascade.
	if err := s.DeleteControl(ctrl.ID); err != nil {
		t.Fatalf("delete control: %v", err)
	}

	after, err := s.ListControlFunctions(ctrl.ID)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected 0 functions after cascade delete, got %d", len(after))
	}
}
