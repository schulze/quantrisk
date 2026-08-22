# Design

## Architecture

Secriskquant is a single Go web application backed by SQLite. Templates and static
assets are embedded in the server binary. The main packages are:

- `fair/` and `fair/cam/`: quantitative risk and control calculations.
- `internal/store/`: SQLite persistence and forward-only migrations.
- `internal/server/`: authenticated HTTP and HTMX handlers.
- `web/`: embedded templates and static assets.

The core domain consists of loss scenarios, controls, gaps, and requirements.
Controls link to scenarios and requirements; gaps belong to either a control or
a requirement.

## Authentication and onboarding

Authentication is passkey-only and uses WebAuthn discoverable credentials.
There is no password or open self-registration flow.

### Initial account

When the database has no users, `/login` enters setup mode. The first visitor
chooses a username and display name and registers a passkey. Once a user exists,
new accounts require an invitation.

### Invitations

An authenticated user creates an invitation from the dashboard by supplying a
username and optional display name. The server:

1. Creates the user record.
2. Generates a cryptographically random 32-byte token encoded as hexadecimal.
3. Stores the token, invited user, inviter, and expiration in `invitations`.
4. Returns a shareable `/invite/{token}` URL.

Invitations expire after 72 hours and are single-use. The invitation page shows
the invited user's display name and starts the WebAuthn registration ceremony.
The invitation is marked used only after the passkey has been saved successfully.
The new user is then signed in automatically.

Invitation URLs are public, but creating invitations and all application pages
require an authenticated session. Invitations only onboard new users; they are
not a credential-recovery or add-passkey mechanism.

### Sessions

Successful registration or login creates a random 32-byte session token with a
30-day lifetime. The token is stored in SQLite and sent in the `qr_session`
HTTP-only, SameSite=Lax cookie. Logout deletes the server-side session.

### Relevant routes

- `GET /login`: setup or passkey login page.
- `POST /auth/register/begin` and `/auth/register/finish`: WebAuthn registration.
- `POST /auth/login/begin` and `/auth/login/finish`: discoverable passkey login.
- `POST /invitations`: authenticated invitation creation.
- `GET /invite/{token}`: public invitation validation and registration page.
- `POST /logout`: session termination.

### Persistence

The initial schema defines `users`, `webauthn_credentials`, `sessions`, and
`invitations`. Credentials and sessions reference users. Invitations reference
both the invited user and inviter and record expiration and use timestamps.
