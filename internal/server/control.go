package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/fair/cam"
	"github.com/schulze/quantrisk/internal/model"
)

func (s *Server) handleControlList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListControls()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Populate FAIR-CAM functions for each control.
	for i := range items {
		items[i].Functions, _ = s.store.ListControlFunctions(items[i].ID)
	}
	auditLog, _ := s.store.ListAuditByType("control", 50)
	data := map[string]any{"Controls": items, "AuditLog": auditLog}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "control-list", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title": "Controls", "Content": "control-index", "Data": data,
	})
}

func (s *Server) handleControlForm(w http.ResponseWriter, r *http.Request) {
	// Legacy handler kept for backward compat; redirects to controls list.
	http.Redirect(w, r, "/controls", http.StatusSeeOther)
}

func (s *Server) handleControlCreate(w http.ResponseWriter, r *http.Request) {
	// Generate a default identifier based on existing count.
	item := &model.Control{
		Identifier: s.store.NextIdentifier("controls", "CTRL"),
		Name:       "New Control",
		Status:     "planned",
	}
	if err := s.store.CreateControl(item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("control", item.ID, item.Identifier, "create", "", "", item.Name)
	redirect := "/controls/" + strconv.FormatInt(item.ID, 10)
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusCreated)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleControlShow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, err := s.store.GetControl(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	funcs, _ := s.store.ListControlFunctions(id)
	item.Functions = funcs
	risks, _ := s.store.ListControlRisks(id)
	reqs, _ := s.store.ListControlRequirements(id)
	gaps, _ := s.store.ListGapsByParent("control", id)
	allRisks, _ := s.store.ListRisks()
	allReqs, _ := s.store.ListRequirements()
	auditLog, _ := s.store.ListAuditByEntity("control", id)
	data := map[string]any{
		"Control":         item,
		"Risks":           risks,
		"Requirements":    reqs,
		"Gaps":            gaps,
		"AllRisks":        allRisks,
		"AllRequirements": allReqs,
		"CAMCatalog":      cam.Catalog,
		"ZeroEstimate":    fair.Estimate{},
		"AuditLog":        auditLog,
	}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "control-show", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title": item.Name, "Content": "control-show", "Data": data,
	})
}

func (s *Server) handleControlEditForm(w http.ResponseWriter, r *http.Request) {
	// Legacy handler kept for backward compat; redirects to show page.
	id := r.PathValue("id")
	http.Redirect(w, r, "/controls/"+id, http.StatusSeeOther)
}

// controlFieldLabel maps field names to display labels.
var controlFieldLabel = map[string]string{
	"identifier":  "Identifier",
	"name":        "Name",
	"description": "Description",
	"status":      "Status",
}

func controlFieldValue(c model.Control, field string) string {
	switch field {
	case "identifier":
		return c.Identifier
	case "name":
		return c.Name
	case "description":
		return c.Description
	case "status":
		return c.Status
	default:
		return ""
	}
}

func (s *Server) handleControlPatch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}
	field := r.FormValue("field")
	value := r.FormValue("value")

	if _, ok := controlFieldLabel[field]; !ok {
		s.renderError(w, "unknown field", http.StatusBadRequest)
		return
	}

	if verrs := validateControlFields(field, value); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}

	item, err := s.store.GetControl(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	oldValue := controlFieldValue(item, field)

	switch field {
	case "identifier":
		item.Identifier = value
	case "name":
		item.Name = value
	case "description":
		item.Description = value
	case "status":
		item.Status = value
	}

	if err := s.store.UpdateControl(&item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if oldValue != value {
		s.audit("control", id, item.Identifier, "update", field, oldValue, value)
	}

	// For name changes, return the updated h1 title.
	if field == "name" {
		fmt.Fprintf(w, `<h1 id="control-title">%s: %s</h1>`,
			template.HTMLEscapeString(item.Identifier),
			template.HTMLEscapeString(item.Name))
		return
	}
	// For other fields, nothing to swap.
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleControlUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}
	identifier := r.FormValue("identifier")
	name := r.FormValue("name")
	description := r.FormValue("description")
	status := r.FormValue("status")
	if verrs := validateControlForm(identifier, name, description, status); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}
	item := &model.Control{
		ID:          id,
		Identifier:  identifier,
		Name:        name,
		Description: description,
		Status:      status,
	}
	if err := s.store.UpdateControl(item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", "/controls/"+strconv.FormatInt(id, 10))
		return
	}
	http.Redirect(w, r, "/controls/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) handleControlLinked(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	risks, _ := s.store.ListControlRisks(id)
	reqs, _ := s.store.ListControlRequirements(id)
	gaps, _ := s.store.ListGapsByParent("control", id)
	s.tmpl.ExecuteTemplate(w, "control-linked", map[string]any{
		"Risks":        risks,
		"Requirements": reqs,
		"Gaps":         gaps,
	})
}

func (s *Server) handleControlDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, _ := s.store.GetControl(id)
	if err := s.store.DeleteControl(id); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("control", id, item.Identifier, "delete", "", item.Name, "")
	if s.isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/controls", http.StatusSeeOther)
}

// Link handlers

func (s *Server) handleControlLinkRisk(w http.ResponseWriter, r *http.Request) {
	controlID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}
	riskID, err := strconv.ParseInt(r.FormValue("risk_id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid risk_id", http.StatusBadRequest)
		return
	}
	if err := s.store.LinkControlRisk(controlID, riskID); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", "/controls/"+strconv.FormatInt(controlID, 10))
		return
	}
	http.Redirect(w, r, "/controls/"+strconv.FormatInt(controlID, 10), http.StatusSeeOther)
}

func (s *Server) handleControlUnlinkRisk(w http.ResponseWriter, r *http.Request) {
	controlID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	riskID, err := strconv.ParseInt(r.PathValue("riskID"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid riskID", http.StatusBadRequest)
		return
	}
	if err := s.store.UnlinkControlRisk(controlID, riskID); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/controls/"+strconv.FormatInt(controlID, 10), http.StatusSeeOther)
}

func (s *Server) handleControlLinkRequirement(w http.ResponseWriter, r *http.Request) {
	controlID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}
	reqID, err := strconv.ParseInt(r.FormValue("requirement_id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid requirement_id", http.StatusBadRequest)
		return
	}
	if err := s.store.LinkControlRequirement(controlID, reqID); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirect := "/controls/" + strconv.FormatInt(controlID, 10)
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", redirect)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleControlUnlinkRequirement(w http.ResponseWriter, r *http.Request) {
	controlID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	reqID, err := strconv.ParseInt(r.PathValue("requirementID"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid requirementID", http.StatusBadRequest)
		return
	}
	if err := s.store.UnlinkControlRequirement(controlID, reqID); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/controls/"+strconv.FormatInt(controlID, 10), http.StatusSeeOther)
}

func (s *Server) handleControlCreateGap(w http.ResponseWriter, r *http.Request) {
	controlID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}
	parentType := "control"
	item := &model.Gap{
		Identifier: s.store.NextIdentifier("gaps", "GAP"),
		Name:       "New Gap",
		Severity:   "medium",
		Status:     "open",
		ParentType: &parentType,
		ParentID:   &controlID,
	}
	if err := s.store.CreateGap(item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("gap", item.ID, item.Identifier, "create", "", "", item.Name)
	redirect := "/gaps/" + strconv.FormatInt(item.ID, 10)
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusCreated)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleControlCreateFunction(w http.ResponseWriter, r *http.Request) {
	controlID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}

	fn := cam.Function(r.FormValue("function"))
	notes := r.FormValue("notes")
	eff := cam.Effectiveness{
		Capability:  parseEstimate(r, "cap"),
		Coverage:    parseEstimate(r, "cov"),
		Reliability: parseEstimate(r, "rel"),
	}
	if verrs := validateControlFunction(fn, notes, eff); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}

	cf := &model.ControlFunction{
		ControlID:     controlID,
		Function:      fn,
		Notes:         notes,
		Effectiveness: eff,
	}

	if err := s.store.CreateControlFunction(cf); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirect := "/controls/" + strconv.FormatInt(controlID, 10)
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusCreated)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleControlDeleteFunction(w http.ResponseWriter, r *http.Request) {
	controlID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	cfID, err := strconv.ParseInt(r.PathValue("cfID"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid function id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteControlFunction(cfID); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/controls/"+strconv.FormatInt(controlID, 10), http.StatusSeeOther)
}
