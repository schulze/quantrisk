package server

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/schulze/quantrisk/internal/store"
)

const (
	sessionCookie = "qr_session"
	sessionTTL    = 30 * 24 * time.Hour // 30 days
)

// webauthnSessions stores in-flight registration/login challenges.
// Keyed by a short-lived challenge string; cleaned up after use.
var webauthnSessions = struct {
	sync.Mutex
	m map[string]*webauthn.SessionData
}{m: make(map[string]*webauthn.SessionData)}

func storeWebAuthnSession(sd *webauthn.SessionData) {
	webauthnSessions.Lock()
	webauthnSessions.m[sd.Challenge] = sd
	webauthnSessions.Unlock()
}

func loadWebAuthnSession(challenge string) (*webauthn.SessionData, bool) {
	webauthnSessions.Lock()
	sd, ok := webauthnSessions.m[challenge]
	if ok {
		delete(webauthnSessions.m, challenge)
	}
	webauthnSessions.Unlock()
	return sd, ok
}

// requireAuth is middleware that checks for a valid session cookie.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || cookie.Value == "" {
			s.redirectToLogin(w, r)
			return
		}
		_, err = s.store.GetSession(cookie.Value)
		if err != nil {
			s.redirectToLogin(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authenticatedUserID returns the user ID from the session cookie, or 0.
func (s *Server) authenticatedUserID(r *http.Request) int64 {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return 0
	}
	uid, err := s.store.GetSession(cookie.Value)
	if err != nil {
		return 0
	}
	return uid
}

func (s *Server) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if s.isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard.
	if s.authenticatedUserID(r) != 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// No users? Show setup (first-user registration) form.
	count, _ := s.store.CountUsers()
	if count == 0 {
		s.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
			"Title":     "Setup",
			"SetupMode": true,
		})
		return
	}

	s.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
		"Title": "Login",
	})
}

// handleRegisterBegin handles the initial admin registration when no users exist,
// AND the invitation-based registration for new users.
func (s *Server) handleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"displayName"`
		InviteToken string `json:"inviteToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	var user *store.User

	count, _ := s.store.CountUsers()
	if count == 0 {
		// Setup mode: create the first user.
		if req.Username == "" {
			jsonError(w, "username is required", http.StatusBadRequest)
			return
		}
		if req.DisplayName == "" {
			req.DisplayName = req.Username
		}
		var err error
		user, err = s.store.CreateUser(req.Username, req.DisplayName)
		if err != nil {
			jsonError(w, "failed to create user", http.StatusInternalServerError)
			return
		}
	} else if req.InviteToken != "" {
		// Invitation mode: look up the invited user.
		inv, u, err := s.store.GetInvitation(req.InviteToken)
		if err != nil {
			jsonError(w, "invalid or expired invitation", http.StatusForbidden)
			return
		}
		// Check expiry (GetInvitation already checks used_at).
		expires := parseTime(inv.ExpiresAt)
		if time.Now().UTC().After(expires) {
			jsonError(w, "invitation has expired", http.StatusForbidden)
			return
		}
		// Check user doesn't already have credentials.
		if len(u.WebAuthnCredentials()) > 0 {
			jsonError(w, "user already has a passkey registered", http.StatusConflict)
			return
		}
		user = u
	} else {
		jsonError(w, "registration requires an invitation", http.StatusForbidden)
		return
	}

	creation, session, err := s.webauthn.BeginRegistration(user,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
	)
	if err != nil {
		log.Printf("webauthn begin registration: %v", err)
		jsonError(w, "failed to begin registration", http.StatusInternalServerError)
		return
	}

	storeWebAuthnSession(session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creation)
}

func (s *Server) handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Query().Get("challenge")
	if challenge == "" {
		challenge = r.Header.Get("X-Challenge")
	}
	if challenge == "" {
		jsonError(w, "missing challenge", http.StatusBadRequest)
		return
	}

	sd, ok := loadWebAuthnSession(challenge)
	if !ok {
		jsonError(w, "invalid or expired challenge", http.StatusBadRequest)
		return
	}

	userID := int64(binary.BigEndian.Uint64(sd.UserID))
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusBadRequest)
		return
	}

	cred, err := s.webauthn.FinishRegistration(user, *sd, r)
	if err != nil {
		var pErr *protocol.Error
		if errors.As(err, &pErr) {
			log.Printf("webauthn finish registration: %s (debug: %s)", pErr.Details, pErr.DevInfo)
		} else {
			log.Printf("webauthn finish registration: %v", err)
		}
		jsonError(w, "registration failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.AddCredential(user.ID, cred); err != nil {
		jsonError(w, "failed to save credential", http.StatusInternalServerError)
		return
	}

	// If this was an invitation, mark it as used.
	inviteToken := r.URL.Query().Get("invite")
	if inviteToken != "" {
		if err := s.store.UseInvitation(inviteToken); err != nil {
			log.Printf("mark invitation used: %v", err)
		}
	}

	// Auto-login after registration.
	token, err := s.store.CreateSession(user.ID, sessionTTL)
	if err != nil {
		jsonError(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleInviteCreate (POST /invitations) creates a user + invitation.
// Auth-required. Returns a partial with the invitation link.
func (s *Server) handleInviteCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	displayName := r.FormValue("display_name")
	if username == "" {
		http.Error(w, "username is required", http.StatusUnprocessableEntity)
		return
	}
	if displayName == "" {
		displayName = username
	}

	inviterID := s.authenticatedUserID(r)
	token, err := s.store.CreateInvitation(username, displayName, inviterID)
	if err != nil {
		log.Printf("create invitation: %v", err)
		http.Error(w, "failed to create invitation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build the full invitation URL.
	scheme := "https"
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	if fp := r.Header.Get("X-Forwarded-Proto"); fp != "" {
		scheme = fp
	}
	inviteURL := fmt.Sprintf("%s://%s/invite/%s", scheme, host, token)

	s.tmpl.ExecuteTemplate(w, "invite-result", map[string]any{
		"Username":  username,
		"InviteURL": inviteURL,
	})
}

// handleInvitePage (GET /invite/{token}) shows the passkey registration
// form for an invited user. Public endpoint.
func (s *Server) handleInvitePage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	inv, user, err := s.store.GetInvitation(token)
	if err != nil {
		s.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
			"Title": "Invalid Invitation",
			"Error": "This invitation is invalid or has already been used.",
		})
		return
	}

	expires := parseTime(inv.ExpiresAt)
	if time.Now().UTC().After(expires) {
		s.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
			"Title": "Expired Invitation",
			"Error": "This invitation has expired. Please ask an administrator for a new one.",
		})
		return
	}

	if len(user.WebAuthnCredentials()) > 0 {
		s.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
			"Title": "Already Registered",
			"Error": "This account already has a passkey. Please sign in instead.",
		})
		return
	}

	s.tmpl.ExecuteTemplate(w, "login.html", map[string]any{
		"Title":       "Create Your Passkey",
		"InviteMode":  true,
		"InviteToken": token,
		"Username":    user.Username,
		"DisplayName": user.WebAuthnDisplayName(),
	})
}

func (s *Server) handleLoginBegin(w http.ResponseWriter, r *http.Request) {
	assertion, session, err := s.webauthn.BeginDiscoverableLogin()
	if err != nil {
		log.Printf("webauthn begin login: %v", err)
		jsonError(w, "failed to begin login", http.StatusInternalServerError)
		return
	}

	storeWebAuthnSession(session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assertion)
}

func (s *Server) handleLoginFinish(w http.ResponseWriter, r *http.Request) {
	challenge := r.URL.Query().Get("challenge")
	if challenge == "" {
		challenge = r.Header.Get("X-Challenge")
	}
	if challenge == "" {
		jsonError(w, "missing challenge", http.StatusBadRequest)
		return
	}

	sd, ok := loadWebAuthnSession(challenge)
	if !ok {
		jsonError(w, "invalid or expired challenge", http.StatusBadRequest)
		return
	}

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		userID := int64(binary.BigEndian.Uint64(userHandle))
		return s.store.GetUserByID(userID)
	}

	user, cred, err := s.webauthn.FinishPasskeyLogin(handler, *sd, r)
	if err != nil {
		log.Printf("webauthn finish login: %v", err)
		jsonError(w, "login failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Update credential sign count.
	if err := s.store.UpdateCredential(cred); err != nil {
		log.Printf("update credential: %v", err)
	}

	storeUser := user.(*store.User)
	token, err := s.store.CreateSession(storeUser.ID, sessionTTL)
	if err != nil {
		jsonError(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		s.store.DeleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// parseTime tries common SQLite datetime formats.
func parseTime(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
