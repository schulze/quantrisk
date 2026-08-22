package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// User represents a quantrisk user with WebAuthn credentials.
type User struct {
	ID          int64
	Username    string
	DisplayName string
	Credentials []webauthn.Credential
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WebAuthnID returns the user's opaque ID.
func (u *User) WebAuthnID() []byte {
	// Use the integer ID encoded as 8 bytes.
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[i] = byte(u.ID >> (56 - 8*i))
	}
	return b
}

// WebAuthnName returns the username.
func (u *User) WebAuthnName() string { return u.Username }

// WebAuthnDisplayName returns the display name.
func (u *User) WebAuthnDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}

// WebAuthnCredentials returns the user's WebAuthn credentials.
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// GetUserByID returns a user with their WebAuthn credentials loaded.
func (s *Store) GetUserByID(id int64) (*User, error) {
	var u User
	err := s.DB.QueryRow(
		`SELECT id, username, display_name, created_at, updated_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}
	creds, err := s.listCredentials(u.ID)
	if err != nil {
		return nil, err
	}
	u.Credentials = creds
	return &u, nil
}

// GetUserByUsername returns a user with their WebAuthn credentials loaded.
func (s *Store) GetUserByUsername(username string) (*User, error) {
	var u User
	err := s.DB.QueryRow(
		`SELECT id, username, display_name, created_at, updated_at FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user %q: %w", username, err)
	}
	creds, err := s.listCredentials(u.ID)
	if err != nil {
		return nil, err
	}
	u.Credentials = creds
	return &u, nil
}

// CreateUser creates a new user.
func (s *Store) CreateUser(username, displayName string) (*User, error) {
	res, err := s.DB.Exec(
		`INSERT INTO users (username, display_name) VALUES (?, ?)`,
		username, displayName,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username, DisplayName: displayName}, nil
}

// CountUsers returns the total number of users.
func (s *Store) CountUsers() (int, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// AddCredential persists a WebAuthn credential for a user.
func (s *Store) AddCredential(userID int64, cred *webauthn.Credential) error {
	transport, _ := json.Marshal(cred.Transport)
	flags, _ := json.Marshal(cred.Flags)
	auth, _ := json.Marshal(cred.Authenticator)
	attest, _ := json.Marshal(cred.Attestation)

	_, err := s.DB.Exec(
		`INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, transport, flags_json, authenticator_json, attestation_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, cred.ID, cred.PublicKey, cred.AttestationType,
		string(transport), string(flags), string(auth), string(attest),
	)
	if err != nil {
		return fmt.Errorf("add credential: %w", err)
	}
	return nil
}

// UpdateCredential updates the authenticator data (sign count, flags) after login.
func (s *Store) UpdateCredential(cred *webauthn.Credential) error {
	flags, _ := json.Marshal(cred.Flags)
	auth, _ := json.Marshal(cred.Authenticator)
	_, err := s.DB.Exec(
		`UPDATE webauthn_credentials SET flags_json = ?, authenticator_json = ? WHERE credential_id = ?`,
		string(flags), string(auth), cred.ID,
	)
	if err != nil {
		return fmt.Errorf("update credential: %w", err)
	}
	return nil
}

func (s *Store) listCredentials(userID int64) ([]webauthn.Credential, error) {
	rows, err := s.DB.Query(
		`SELECT credential_id, public_key, attestation_type, transport, flags_json, authenticator_json, attestation_json
		 FROM webauthn_credentials WHERE user_id = ?`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}
	defer rows.Close()

	var creds []webauthn.Credential
	for rows.Next() {
		var (
			c             webauthn.Credential
			transportJSON string
			flagsJSON     string
			authJSON      string
			attestJSON    string
		)
		if err := rows.Scan(&c.ID, &c.PublicKey, &c.AttestationType, &transportJSON, &flagsJSON, &authJSON, &attestJSON); err != nil {
			return nil, fmt.Errorf("scan credential: %w", err)
		}
		json.Unmarshal([]byte(transportJSON), &c.Transport)
		json.Unmarshal([]byte(flagsJSON), &c.Flags)
		json.Unmarshal([]byte(authJSON), &c.Authenticator)
		json.Unmarshal([]byte(attestJSON), &c.Attestation)
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// CreateSession creates a new session token for a user.
func (s *Store) CreateSession(userID int64, ttl time.Duration) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	token := hex.EncodeToString(b)
	expiresAt := time.Now().Add(ttl).UTC().Format("2006-01-02 15:04:05")
	_, err := s.DB.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

// GetSession validates a session token and returns the user ID.
func (s *Store) GetSession(token string) (int64, error) {
	var userID int64
	err := s.DB.QueryRow(
		`SELECT user_id FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`, token,
	).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("get session: %w", err)
	}
	return userID, nil
}

// DeleteSession removes a session.
func (s *Store) DeleteSession(token string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// DeleteExpiredSessions cleans up expired sessions.
func (s *Store) DeleteExpiredSessions() error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	return err
}

// LookupUserByCredentialID finds a user by WebAuthn credential ID (for discoverable login).
func (s *Store) LookupUserByCredentialID(credentialID []byte) (*User, error) {
	var userID int64
	err := s.DB.QueryRow(
		`SELECT user_id FROM webauthn_credentials WHERE credential_id = ?`, credentialID,
	).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("lookup user by credential: %w", err)
	}
	return s.GetUserByID(userID)
}

// Invitation represents a pending user invitation.
type Invitation struct {
	ID        int64
	Token     string
	UserID    int64
	InvitedBy int64
	ExpiresAt string
	UsedAt    *string
	CreatedAt string
}

const invitationTTL = 72 * time.Hour

// CreateInvitation creates a user and an invitation token for them.
// Returns the invitation token.
func (s *Store) CreateInvitation(username, displayName string, invitedBy int64) (string, error) {
	user, err := s.CreateUser(username, displayName)
	if err != nil {
		return "", fmt.Errorf("create invited user: %w", err)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate invitation token: %w", err)
	}
	token := hex.EncodeToString(b)
	expiresAt := time.Now().Add(invitationTTL).UTC().Format("2006-01-02 15:04:05")

	_, err = s.DB.Exec(
		`INSERT INTO invitations (token, user_id, invited_by, expires_at) VALUES (?, ?, ?, ?)`,
		token, user.ID, invitedBy, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("create invitation: %w", err)
	}
	return token, nil
}

// GetInvitation returns a valid (unexpired, unused) invitation and its user.
func (s *Store) GetInvitation(token string) (*Invitation, *User, error) {
	var inv Invitation
	err := s.DB.QueryRow(
		`SELECT id, token, user_id, invited_by, expires_at, used_at, created_at
		 FROM invitations WHERE token = ?`, token,
	).Scan(&inv.ID, &inv.Token, &inv.UserID, &inv.InvitedBy, &inv.ExpiresAt, &inv.UsedAt, &inv.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("invitation not found: %w", err)
	}
	if inv.UsedAt != nil {
		return nil, nil, fmt.Errorf("invitation already used")
	}

	user, err := s.GetUserByID(inv.UserID)
	if err != nil {
		return nil, nil, err
	}
	return &inv, user, nil
}

// UseInvitation marks an invitation as used.
func (s *Store) UseInvitation(token string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := s.DB.Exec(
		`UPDATE invitations SET used_at = ? WHERE token = ? AND used_at IS NULL`,
		now, token,
	)
	if err != nil {
		return fmt.Errorf("use invitation: %w", err)
	}
	return nil
}
