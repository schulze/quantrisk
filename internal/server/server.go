package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/schulze/quantrisk/fair/cam"
	"github.com/schulze/quantrisk/internal/store"
	"github.com/schulze/quantrisk/web"
)

type Server struct {
	store    *store.Store
	tmpl     *template.Template
	mux      *http.ServeMux
	years    int
	webauthn *webauthn.WebAuthn
}

func New(s *store.Store, years int, wa *webauthn.WebAuthn) *Server {
	srv := &Server{
		store:    s,
		mux:      http.NewServeMux(),
		years:    years,
		webauthn: wa,
	}
	srv.parseTemplates()
	srv.routes()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) parseTemplates() {
	funcMap := template.FuncMap{
		"printf": func(format string, a ...any) string {
			return fmt.Sprintf(format, a...)
		},
		"dict": func(pairs ...any) map[string]any {
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i+1 < len(pairs); i += 2 {
				m[pairs[i].(string)] = pairs[i+1]
			}
			return m
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"camLookup": func(f cam.Function) cam.FunctionInfo {
			fi, _ := cam.Lookup(f)
			return fi
		},
		"camFunctions": func() []cam.FunctionInfo {
			return cam.Catalog
		},
		"camDomainFunctions": func(d cam.Domain) []cam.FunctionInfo {
			return cam.Functions(d)
		},
		"derefStr": func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		},
		"derefInt": func(p *int64) int64 {
			if p == nil {
				return 0
			}
			return *p
		},
	}

	t, err := template.New("").Funcs(funcMap).ParseFS(web.TemplateFS, "templates/*.html", "templates/**/*.html")
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}
	s.tmpl = t
}

func (s *Server) routes() {
	staticFS, _ := fs.Sub(web.StaticFS, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Public auth routes.
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /logout", s.handleLogout)
	s.mux.HandleFunc("POST /auth/register/begin", s.handleRegisterBegin)
	s.mux.HandleFunc("POST /auth/register/finish", s.handleRegisterFinish)
	s.mux.HandleFunc("POST /auth/login/begin", s.handleLoginBegin)
	s.mux.HandleFunc("POST /auth/login/finish", s.handleLoginFinish)
	s.mux.HandleFunc("GET /invite/{token}", s.handleInvitePage)

	// All routes below require authentication.
	auth := s.requireAuth

	s.mux.Handle("GET /{$}", auth(http.HandlerFunc(s.handleIndex)))

	// af wraps a handler with authentication.
	af := func(pattern string, h http.HandlerFunc) {
		s.mux.Handle(pattern, auth(h))
	}

	// Invitations
	af("POST /invitations", s.handleInviteCreate)

	// Risks
	af("GET /risks", s.handleRiskList)
	af("GET /risks/new", s.handleRiskForm)
	af("POST /risks", s.handleRiskCreate)
	af("GET /risks/{id}/linked", s.handleRiskLinked)
	af("GET /risks/{id}", s.handleRiskShow)
	af("GET /risks/{id}/edit", s.handleRiskEditForm)
	af("PUT /risks/{id}", s.handleRiskUpdate)
	af("PATCH /risks/{id}", s.handleRiskPatch)
	af("DELETE /risks/{id}", s.handleRiskDelete)

	// Requirements
	af("GET /requirements", s.handleRequirementList)
	af("GET /requirements/new", s.handleRequirementForm)
	af("POST /requirements", s.handleRequirementCreate)
	af("GET /requirements/{id}/linked", s.handleRequirementLinked)
	af("GET /requirements/{id}", s.handleRequirementShow)
	af("GET /requirements/{id}/edit", s.handleRequirementEditForm)
	af("PUT /requirements/{id}", s.handleRequirementUpdate)
	af("PATCH /requirements/{id}", s.handleRequirementPatch)
	af("DELETE /requirements/{id}", s.handleRequirementDelete)
	af("POST /requirements/{id}/controls", s.handleRequirementLinkControl)
	af("DELETE /requirements/{id}/controls/{controlID}", s.handleRequirementUnlinkControl)
	af("POST /requirements/{id}/gaps", s.handleRequirementCreateGap)

	// Controls
	af("GET /controls", s.handleControlList)
	af("GET /controls/new", s.handleControlForm)
	af("POST /controls", s.handleControlCreate)
	af("GET /controls/{id}/linked", s.handleControlLinked)
	af("GET /controls/{id}", s.handleControlShow)
	af("GET /controls/{id}/edit", s.handleControlEditForm)
	af("PUT /controls/{id}", s.handleControlUpdate)
	af("PATCH /controls/{id}", s.handleControlPatch)
	af("DELETE /controls/{id}", s.handleControlDelete)
	af("POST /controls/{id}/risks", s.handleControlLinkRisk)
	af("DELETE /controls/{id}/risks/{riskID}", s.handleControlUnlinkRisk)
	af("POST /controls/{id}/requirements", s.handleControlLinkRequirement)
	af("DELETE /controls/{id}/requirements/{requirementID}", s.handleControlUnlinkRequirement)
	af("POST /controls/{id}/gaps", s.handleControlCreateGap)
	af("POST /controls/{id}/functions", s.handleControlCreateFunction)
	af("DELETE /controls/{id}/functions/{cfID}", s.handleControlDeleteFunction)

	// Gaps
	af("GET /gaps", s.handleGapList)
	af("GET /gaps/new", s.handleGapForm)
	af("POST /gaps", s.handleGapCreate)
	af("GET /gaps/{id}/linked", s.handleGapLinked)
	af("GET /gaps/{id}", s.handleGapShow)
	af("GET /gaps/{id}/edit", s.handleGapEditForm)
	af("PUT /gaps/{id}", s.handleGapUpdate)
	af("PATCH /gaps/{id}", s.handleGapPatch)
	af("DELETE /gaps/{id}", s.handleGapDelete)

	// Simulation
	af("POST /simulate", s.handleSimulate)
	af("GET /simulate/lec", s.handleLECChart)
	af("POST /risks/{id}/simulate", s.handleRiskSimulate)
	af("GET /risks/{id}/lec", s.handleRiskLECChart)

	// Audit log
	af("GET /audit/{entityType}", s.handleAuditByType)
	af("GET /audit/{entityType}/{id}", s.handleAuditByEntity)

}

func (s *Server) isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (s *Server) renderError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
}

// audit is a convenience wrapper for recording audit log entries.
func (s *Server) audit(entityType string, entityID int64, identifier, action, field, oldValue, newValue string) {
	if err := s.store.RecordAudit(entityType, entityID, identifier, action, field, oldValue, newValue); err != nil {
		log.Printf("audit log error: %v", err)
	}
}
