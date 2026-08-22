package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/schulze/quantrisk/internal/model"
)

var gapFieldLabels = map[string]string{
	"name":        "Name",
	"description": "Description",
	"severity":    "Severity",
	"status":      "Status",
}

func gapFieldValue(g model.Gap, field string) string {
	switch field {
	case "name":
		return g.Name
	case "description":
		return g.Description
	case "severity":
		return g.Severity
	case "status":
		return g.Status
	default:
		return ""
	}
}

func (s *Server) handleGapList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListGaps()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	auditLog, _ := s.store.ListAuditByType("gap", 50)
	data := map[string]any{"Gaps": items, "AuditLog": auditLog}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "gap-list", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title": "Gaps", "Content": "gap-index", "Data": data,
	})
}

func (s *Server) handleGapForm(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/gaps", http.StatusSeeOther)
}

func (s *Server) handleGapCreate(w http.ResponseWriter, r *http.Request) {
	item := &model.Gap{
		Identifier: s.store.NextIdentifier("gaps", "GAP"),
		Name:       "New Gap",
		Severity:   "medium",
		Status:     "open",
		ParentType: nil,
		ParentID:   nil,
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

func (s *Server) handleGapShow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, err := s.store.GetGap(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	// Resolve parent name for display.
	var parentLabel string
	if item.ParentType != nil && item.ParentID != nil {
		switch *item.ParentType {
		case "control":
			if p, err := s.store.GetControl(*item.ParentID); err == nil {
				parentLabel = p.Identifier + ": " + p.Name
			}
		case "requirement":
			if p, err := s.store.GetRequirement(*item.ParentID); err == nil {
				parentLabel = p.Identifier + ": " + p.Name
			}
		}
	}

	auditLog, _ := s.store.ListAuditByEntity("gap", id)
	data := map[string]any{"Gap": item, "ParentLabel": parentLabel, "AuditLog": auditLog}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "gap-show", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title": item.Name, "Content": "gap-show", "Data": data,
	})
}

func (s *Server) handleGapEditForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	http.Redirect(w, r, "/gaps/"+id, http.StatusSeeOther)
}

func (s *Server) handleGapPatch(w http.ResponseWriter, r *http.Request) {
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

	if _, ok := gapFieldLabels[field]; !ok {
		s.renderError(w, "unknown field", http.StatusBadRequest)
		return
	}

	if verrs := validateGapFields(field, value); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}

	item, err := s.store.GetGap(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	oldValue := gapFieldValue(item, field)

	switch field {
	case "name":
		item.Name = value
	case "description":
		item.Description = value
	case "severity":
		item.Severity = value
	case "status":
		item.Status = value
	}

	if err := s.store.UpdateGap(&item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if oldValue != value {
		s.audit("gap", id, item.Identifier, "update", field, oldValue, value)
	}

	if field == "name" {
		fmt.Fprintf(w, `<h1 id="gap-title">%s: %s</h1>`,
			template.HTMLEscapeString(item.Identifier),
			template.HTMLEscapeString(item.Name))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGapUpdate(w http.ResponseWriter, r *http.Request) {
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
	severity := r.FormValue("severity")
	status := r.FormValue("status")
	if verrs := validateGapForm(identifier, name, description, severity, status); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}
	item := &model.Gap{
		ID:          id,
		Identifier:  identifier,
		Name:        name,
		Description: description,
		Severity:    severity,
		Status:      status,
	}
	if err := s.store.UpdateGap(item); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", "/gaps/"+strconv.FormatInt(id, 10))
		return
	}
	http.Redirect(w, r, "/gaps/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) handleGapLinked(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, err := s.store.GetGap(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	var parentLabel string
	if item.ParentType != nil && item.ParentID != nil {
		switch *item.ParentType {
		case "control":
			if p, err := s.store.GetControl(*item.ParentID); err == nil {
				parentLabel = p.Identifier + ": " + p.Name
			}
		case "requirement":
			if p, err := s.store.GetRequirement(*item.ParentID); err == nil {
				parentLabel = p.Identifier + ": " + p.Name
			}
		}
	}
	s.tmpl.ExecuteTemplate(w, "gap-linked", map[string]any{
		"Gap":         item,
		"ParentLabel": parentLabel,
	})
}

func (s *Server) handleGapDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, _ := s.store.GetGap(id)
	if err := s.store.DeleteGap(id); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("gap", id, item.Identifier, "delete", "", item.Name, "")
	if s.isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/gaps", http.StatusSeeOther)
}
