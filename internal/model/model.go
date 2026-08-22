package model

import (
	"time"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/fair/cam"
)

// Risk is a persisted FAIR loss event scenario with metadata.
type Risk struct {
	ID         int64
	Identifier string
	Scenario   string

	fair.LossEvent

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Requirement struct {
	ID          int64
	Identifier  string
	Name        string
	Description string
	Source      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Control struct {
	ID          int64
	Identifier  string
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Functions holds the FAIR-CAM function assignments for this control.
	// Populated by store queries that join control_functions.
	Functions []ControlFunction `json:"functions,omitempty"`
}

// ControlFunction is a persisted FAIR-CAM function assignment for a control,
// with operational effectiveness (Capability, Coverage, Reliability).
type ControlFunction struct {
	ID            int64
	ControlID     int64
	Function      cam.Function
	Effectiveness cam.Effectiveness
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Gap struct {
	ID          int64
	Identifier  string
	Name        string
	Description string
	Severity    string
	Status      string
	ParentType  *string
	ParentID    *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
