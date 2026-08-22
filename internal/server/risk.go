package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/internal/model"
)

func parseFloat(r *http.Request, key string) float64 {
	v, _ := strconv.ParseFloat(r.FormValue(key), 64)
	return v
}

func parseEstimate(r *http.Request, prefix string) fair.Estimate {
	return fair.Estimate{
		Min:       parseFloat(r, prefix+"_min"),
		ML:        parseFloat(r, prefix+"_ml"),
		Max:       parseFloat(r, prefix+"_max"),
		Rationale: r.FormValue(prefix + "_rationale"),
	}
}

func parseLossForm(r *http.Request, prefix string) fair.LossForm {
	return fair.LossForm{
		ProdL: parseEstimate(r, prefix+"_prodl"),
		RespC: parseEstimate(r, prefix+"_respc"),
		ReplC: parseEstimate(r, prefix+"_replc"),
		FinJu: parseEstimate(r, prefix+"_finju"),
		RepuD: parseEstimate(r, prefix+"_repud"),
		CAdvL: parseEstimate(r, prefix+"_cadvl"),
	}
}

// fieldDiff records a single changed field for audit logging.
type fieldDiff struct {
	field  string
	oldVal string
	newVal string
}

// fmtNum formats a float for audit display: integers stay clean, small
// decimals show up to 4 significant digits.
func fmtNum(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// fmtEstimate formats an Estimate as "min / ml / max" for audit display.
func fmtEstimate(e fair.Estimate) string {
	if e.IsZero() {
		return ""
	}
	return fmtNum(e.Min) + " / " + fmtNum(e.ML) + " / " + fmtNum(e.Max)
}

// diffEstimate compares two estimates and returns diffs for numeric values and rationale.
func diffEstimate(label string, old, new fair.Estimate) []fieldDiff {
	var diffs []fieldDiff
	if old.Min != new.Min || old.ML != new.ML || old.Max != new.Max {
		diffs = append(diffs, fieldDiff{
			field:  label,
			oldVal: fmtEstimate(old),
			newVal: fmtEstimate(new),
		})
	}
	if old.Rationale != new.Rationale {
		diffs = append(diffs, fieldDiff{
			field:  label + " rationale",
			oldVal: old.Rationale,
			newVal: new.Rationale,
		})
	}
	return diffs
}

// diffLossEvent compares two LossEvents and returns per-field diffs.
func diffLossEvent(old, new fair.LossEvent) []fieldDiff {
	var diffs []fieldDiff

	if old.LEFMode != new.LEFMode {
		diffs = append(diffs, fieldDiff{
			field:  "LEF Mode",
			oldVal: old.LEFMode.String(),
			newVal: new.LEFMode.String(),
		})
	}

	diffs = append(diffs, diffEstimate("LEF", old.DirectLEF, new.DirectLEF)...)
	diffs = append(diffs, diffEstimate("TEF", old.TEF, new.TEF)...)
	diffs = append(diffs, diffEstimate("Susceptibility", old.Susc, new.Susc)...)

	// Primary Loss forms
	plNames := []struct {
		label string
		o, n  fair.Estimate
	}{
		{"PL Productivity Loss", old.PL.ProdL, new.PL.ProdL},
		{"PL Response Costs", old.PL.RespC, new.PL.RespC},
		{"PL Replacement Costs", old.PL.ReplC, new.PL.ReplC},
		{"PL Fines & Judgments", old.PL.FinJu, new.PL.FinJu},
		{"PL Reputation Damage", old.PL.RepuD, new.PL.RepuD},
		{"PL Competitive Adv Loss", old.PL.CAdvL, new.PL.CAdvL},
	}
	for _, p := range plNames {
		diffs = append(diffs, diffEstimate(p.label, p.o, p.n)...)
	}

	// Secondary Loss forms
	slNames := []struct {
		label string
		o, n  fair.Estimate
	}{
		{"SL Productivity Loss", old.SL.ProdL, new.SL.ProdL},
		{"SL Response Costs", old.SL.RespC, new.SL.RespC},
		{"SL Replacement Costs", old.SL.ReplC, new.SL.ReplC},
		{"SL Fines & Judgments", old.SL.FinJu, new.SL.FinJu},
		{"SL Reputation Damage", old.SL.RepuD, new.SL.RepuD},
		{"SL Competitive Adv Loss", old.SL.CAdvL, new.SL.CAdvL},
	}
	for _, p := range slNames {
		diffs = append(diffs, diffEstimate(p.label, p.o, p.n)...)
	}

	return diffs
}

func riskFromForm(r *http.Request) *model.Risk {
	mode, err := fair.ParseLEFMode(r.FormValue("lef_mode"))
	if err != nil {
		mode = fair.LEFDecomposed
	}

	risk := &model.Risk{
		Identifier: r.FormValue("identifier"),
		Scenario:   r.FormValue("scenario"),
		LossEvent: fair.LossEvent{
			LEFMode: mode,
			PL:      parseLossForm(r, "pl"),
			SL:      parseLossForm(r, "sl"),
		},
	}

	if mode == fair.LEFDirect {
		risk.DirectLEF = parseEstimate(r, "lef")
		// Clear TEF/Susc — they are not applicable in direct mode
	} else {
		risk.TEF = parseEstimate(r, "tef")
		risk.Susc = parseEstimate(r, "susc")
		// Clear DirectLEF — it is not applicable in decomposed mode
	}

	return risk
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	risks, err := s.store.ListRisks()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	controls, err := s.store.ListControls()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	requirements, err := s.store.ListRequirements()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	gaps, err := s.store.ListGaps()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"Risks":        risks,
		"RiskCount":    len(risks),
		"Controls":     controls,
		"Requirements": requirements,
		"Gaps":         gaps,
	}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "index-content", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title":   "Dashboard",
		"Content": "index",
		"Data":    data,
	})
}

func (s *Server) handleRiskList(w http.ResponseWriter, r *http.Request) {
	risks, err := s.store.ListRisks()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	auditLog, _ := s.store.ListAuditByType("risk", 50)
	data := map[string]any{"Risks": risks, "AuditLog": auditLog}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "risk-list", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title":   "Risk Scenarios",
		"Content": "risk-index",
		"Data":    data,
	})
}

func (s *Server) handleRiskForm(w http.ResponseWriter, r *http.Request) {
	// For GET /risks/new without an ID, redirect to the list.
	// The "New Scenario" button now POSTs to create a stub.
	http.Redirect(w, r, "/risks", http.StatusSeeOther)
}

func (s *Server) handleRiskCreate(w http.ResponseWriter, r *http.Request) {
	risk := &model.Risk{
		Identifier: s.store.NextIdentifier("risks", "RISK"),
		Scenario:   "New Scenario",
		LossEvent:  fair.LossEvent{LEFMode: fair.LEFDecomposed},
	}
	if err := s.store.CreateRisk(risk); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("risk", risk.ID, risk.Identifier, "create", "", "", risk.Scenario)
	redirect := "/risks/" + strconv.FormatInt(risk.ID, 10)
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusCreated)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *Server) handleRiskShow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	risk, err := s.store.GetRisk(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	controls, _ := s.store.ListRiskControls(id)
	auditLog, _ := s.store.ListAuditByEntity("risk", id)
	// Open the FAIR form automatically for freshly-created scenarios.
	freshRisk := risk.Scenario == "New Scenario"
	data := map[string]any{"Risk": risk, "Controls": controls, "AuditLog": auditLog, "FreshRisk": freshRisk}
	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "risk-show", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title":   risk.Scenario,
		"Content": "risk-show",
		"Data":    data,
	})
}

func (s *Server) handleRiskEditForm(w http.ResponseWriter, r *http.Request) {
	// The FAIR form is now embedded on the show page. Redirect there.
	id := r.PathValue("id")
	http.Redirect(w, r, "/risks/"+id, http.StatusSeeOther)
}

func (s *Server) handleRiskPatch(w http.ResponseWriter, r *http.Request) {
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

	if field != "scenario" {
		s.renderError(w, "unknown field", http.StatusBadRequest)
		return
	}

	if verrs := validateRiskPatchField(field, value); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}

	risk, err := s.store.GetRisk(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	oldValue := risk.Scenario
	risk.Scenario = value
	if err := s.store.UpdateRisk(&risk); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if oldValue != value {
		s.audit("risk", id, risk.Identifier, "update", field, oldValue, value)
	}

	fmt.Fprintf(w, `<h1 id="risk-title">%s: %s</h1>`,
		template.HTMLEscapeString(risk.Identifier),
		template.HTMLEscapeString(risk.Scenario))
}

func (s *Server) handleRiskUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderError(w, "invalid form data", http.StatusBadRequest)
		return
	}

	old, err := s.store.GetRisk(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	risk := riskFromForm(r)
	risk.ID = id
	if verrs := validateRiskForm(risk); verrs.HasErrors() {
		validationFailed(w, verrs)
		return
	}
	if err := s.store.UpdateRisk(risk); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit individual FAIR field changes.
	for _, d := range diffLossEvent(old.LossEvent, risk.LossEvent) {
		s.audit("risk", id, risk.Identifier, "update", d.field, d.oldVal, d.newVal)
	}

	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", "/risks/"+strconv.FormatInt(id, 10))
		return
	}
	http.Redirect(w, r, "/risks/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) handleRiskLinked(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	controls, _ := s.store.ListRiskControls(id)
	s.tmpl.ExecuteTemplate(w, "risk-linked", map[string]any{
		"Controls": controls,
	})
}

func (s *Server) handleRiskDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	// Capture info before deletion for audit log.
	risk, _ := s.store.GetRisk(id)
	if err := s.store.DeleteRisk(id); err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.audit("risk", id, risk.Identifier, "delete", "", risk.Scenario, "")
	if s.isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/risks", http.StatusSeeOther)
}
