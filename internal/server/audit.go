package server

import (
	"net/http"
	"strconv"
)

func (s *Server) handleAuditByType(w http.ResponseWriter, r *http.Request) {
	entityType := r.PathValue("entityType")
	if verrs := validateAuditEntityType(entityType); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}
	entries, err := s.store.ListAuditByType(entityType, 100)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"Entries":    entries,
		"EntityType": entityType,
	}
	s.tmpl.ExecuteTemplate(w, "audit-log", data)
}

func (s *Server) handleAuditByEntity(w http.ResponseWriter, r *http.Request) {
	entityType := r.PathValue("entityType")
	if verrs := validateAuditEntityType(entityType); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	entries, err := s.store.ListAuditByEntity(entityType, id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"Entries":    entries,
		"EntityType": entityType,
		"EntityID":   id,
	}
	s.tmpl.ExecuteTemplate(w, "audit-log", data)
}
