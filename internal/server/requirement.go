package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/schulze/quantrisk/internal/model"
)

var requirementFieldLabels = map[string]string{
	"name":        "Name",
	"description": "Description",
	"source":      "Source",
}

func requirementFieldValue(r model.Requirement, field string) string {
	switch field {
	case "name":
		return r.Name
	case "description":
		return r.Description
	case "source":
		return r.Source
	default:
		return ""
	}
}

func (s *Server) handleRequirementList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListRequirements()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	auditLog, _ := s.store.ListAuditByType("requirement", 50)
	data := map[string]any{"Requirements": items, "AuditLog": auditLog}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "requirement-list", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title": "Requirements", "Content": "requirement-index", "Data": data,
	})
}

func (s *Server) handleRequirementForm(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/requirements", http.StatusSeeOther)
}

func (s *Server) handleRequirementCreate(w http.ResponseWriter, r *http.Request) {
	item := &model.Requirement{
		Identifier: s.store.NextIdentifier("requirements", "REQ"),
		Name:       "New Requirement",
	}
	if err := s.store.CreateRequirement(item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("requirement", item.ID, item.Identifier, "create", "", "", item.Name)
	redirect := "/requirements/" + strconv.FormatInt(item.ID, 10)
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusCreated)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleRequirementShow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, err := s.store.GetRequirement(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	controls, _ := s.store.ListRequirementControls(id)
	gaps, _ := s.store.ListGapsByParent("requirement", id)
	allControls, _ := s.store.ListControls()
	auditLog, _ := s.store.ListAuditByEntity("requirement", id)
	data := map[string]any{
		"Requirement": item,
		"Controls":    controls,
		"Gaps":        gaps,
		"AllControls": allControls,
		"AuditLog":    auditLog,
	}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "requirement-show", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title": item.Name, "Content": "requirement-show", "Data": data,
	})
}

func (s *Server) handleRequirementEditForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	http.Redirect(w, r, "/requirements/"+id, http.StatusSeeOther)
}

func (s *Server) handleRequirementPatch(w http.ResponseWriter, r *http.Request) {
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

	if _, ok := requirementFieldLabels[field]; !ok {
		s.renderError(w, "unknown field", http.StatusBadRequest)
		return
	}

	if verrs := validateRequirementFields(field, value); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}

	item, err := s.store.GetRequirement(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	oldValue := requirementFieldValue(item, field)

	switch field {
	case "name":
		item.Name = value
	case "description":
		item.Description = value
	case "source":
		item.Source = value
	}

	if err := s.store.UpdateRequirement(&item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if oldValue != value {
		s.audit("requirement", id, item.Identifier, "update", field, oldValue, value)
	}

	if field == "name" {
		fmt.Fprintf(w, `<h1 id="req-title">%s: %s</h1>`,
			template.HTMLEscapeString(item.Identifier),
			template.HTMLEscapeString(item.Name))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRequirementUpdate(w http.ResponseWriter, r *http.Request) {
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
	source := r.FormValue("source")
	if verrs := validateRequirementForm(identifier, name, description, source); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}
	item := &model.Requirement{
		ID:          id,
		Identifier:  identifier,
		Name:        name,
		Description: description,
		Source:      source,
	}
	if err := s.store.UpdateRequirement(item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", "/requirements/"+strconv.FormatInt(id, 10))
		return
	}
	http.Redirect(w, r, "/requirements/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) handleRequirementLinked(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	controls, _ := s.store.ListRequirementControls(id)
	gaps, _ := s.store.ListGapsByParent("requirement", id)
	s.tmpl.ExecuteTemplate(w, "requirement-linked", map[string]any{
		"Controls": controls,
		"Gaps":     gaps,
	})
}

func (s *Server) handleRequirementDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, _ := s.store.GetRequirement(id)
	if err := s.store.DeleteRequirement(id); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("requirement", id, item.Identifier, "delete", "", item.Name, "")
	if s.isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/requirements", http.StatusSeeOther)
}

func (s *Server) handleRequirementLinkControl(w http.ResponseWriter, r *http.Request) {
	reqID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}
	controlID, err := strconv.ParseInt(r.FormValue("control_id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid control_id", http.StatusBadRequest)
		return
	}
	if err := s.store.LinkControlRequirement(controlID, reqID); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirect := "/requirements/" + strconv.FormatInt(reqID, 10)
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", redirect)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleRequirementUnlinkControl(w http.ResponseWriter, r *http.Request) {
	reqID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	controlID, err := strconv.ParseInt(r.PathValue("controlID"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid controlID", http.StatusBadRequest)
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
	http.Redirect(w, r, "/requirements/"+strconv.FormatInt(reqID, 10), http.StatusSeeOther)
}

func (s *Server) handleRequirementCreateGap(w http.ResponseWriter, r *http.Request) {
	reqID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}
	parentType := "requirement"
	item := &model.Gap{
		Identifier: s.store.NextIdentifier("gaps", "GAP"),
		Name:       "New Gap",
		Severity:   "medium",
		Status:     "open",
		ParentType: &parentType,
		ParentID:   &reqID,
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
