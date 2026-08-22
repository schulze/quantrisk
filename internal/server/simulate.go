package server

import (
	"net/http"
	"strconv"

	"github.com/schulze/quantrisk/chart"
	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/internal/model"
)

// riskToScenario converts a persisted Risk into a fair.Scenario.
func riskToScenario(r model.Risk) fair.Scenario {
	return fair.Scenario{
		Identifier: r.Identifier,
		Name:       r.Scenario,
		LossEvent:  r.LossEvent,
	}
}

func (s *Server) loadScenarios() ([]fair.Scenario, error) {
	risks, err := s.store.ListRisks()
	if err != nil {
		return nil, err
	}
	scenarios := make([]fair.Scenario, len(risks))
	for i, r := range risks {
		scenarios[i] = riskToScenario(r)
	}
	return scenarios, nil
}

func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	years := s.years
	if v := r.FormValue("years"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			years = n
		}
	}

	scenarios, err := s.loadScenarios()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(scenarios) == 0 {
		s.renderError(w, "no risks defined", http.StatusBadRequest)
		return
	}

	priorities, err := fair.PrioritizedLosses(scenarios)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Priorities": priorities,
		"Years":      years,
	}

	if s.isHTMX(r) {
		s.tmpl.ExecuteTemplate(w, "simulate-results", data)
		return
	}
	s.tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title":   "Simulation Results",
		"Content": "simulate-results",
		"Data":    data,
	})
}

func (s *Server) handleLECChart(w http.ResponseWriter, r *http.Request) {
	years := s.years
	if v := r.URL.Query().Get("years"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			years = n
		}
	}

	scenarios, err := s.loadScenarios()
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(scenarios) == 0 {
		s.renderError(w, "no risks defined", http.StatusBadRequest)
		return
	}

	_ = years // simulation uses 10000 internally for chart resolution
	perScenario, aggregate, err := fair.SimulateMulti(scenarios, 10000)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var curves []chart.NamedCurve
	for i, sc := range scenarios {
		points := chart.ExceedancePointsFrom(perScenario[i].YearlyLosses, 99)
		curves = append(curves, chart.NamedCurve{
			Label:  sc.Label(),
			Points: points,
		})
	}
	if len(scenarios) > 1 {
		points := chart.ExceedancePointsFrom(aggregate.YearlyLosses, 99)
		curves = append(curves, chart.NamedCurve{
			Label:  "Aggregate",
			Points: points,
		})
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	chart.RenderLEC(curves, "Loss Exceedance Curves", w)
}

// handleRiskSimulate runs simulation for a single risk scenario.
func (s *Server) handleRiskSimulate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	rsk, err := s.store.GetRisk(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	al, err := fair.AnnualizedLoss(rsk.LossEvent)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Risk":       rsk,
		"AnnualLoss": al,
	}
	s.tmpl.ExecuteTemplate(w, "risk-simulate-results", data)
}

// handleRiskLECChart serves an SVG LEC for a single risk scenario.
func (s *Server) handleRiskLECChart(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, "invalid id", http.StatusBadRequest)
		return
	}
	rsk, err := s.store.GetRisk(id)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	sc := riskToScenario(rsk)
	result, err := fair.SimulateLossEvent(sc.LossEvent, 10000)
	if err != nil {
		s.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	points := chart.ExceedancePointsFrom(result.YearlyLosses, 99)
	title := sc.Label()
	if len(title) > 60 {
		title = title[:57] + "\u2026"
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	chart.RenderLEC([]chart.NamedCurve{{Label: sc.Label(), Points: points}}, title, w)
}
