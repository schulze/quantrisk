# secriskquant - Agent Context

## Purpose

Quantitative security risk and control management proof of concept. The application combines FAIR loss modeling, FAIR-CAM control analytics, SQLite persistence, passkey authentication, and an HTMX web interface.

## Module

- Path: `github.com/schulze/quantrisk`
- Go: 1.25
- Database: SQLite via `modernc.org/sqlite`

## Main components

- `cmd/quantriskd`: web server
- `cmd/quantriskcli`: administration, CSV import, and simulation CLI
- `cmd/seed`: realistic fixture loader
- `fair/`: FAIR model and Monte Carlo simulation
- `fair/cam/`: FAIR-CAM ontology and effectiveness calculations
- `chart/`: SVG loss exceedance curves
- `internal/model/`: persisted domain types
- `internal/store/`: SQLite queries and migrations
- `internal/server/`: authenticated HTTP handlers
- `web/`: embedded templates and static assets

## Domain

The proof of concept manages:

- Loss scenarios
- Controls
- Requirements
- Gaps
- FAIR-CAM function assignments
- Audit history
- Users, passkeys, sessions, and invitations

Relationships connect controls to scenarios and requirements. Gaps can belong to controls or requirements.

## Database migrations

Migrations are forward-only SQL files embedded from `internal/store/migrations/`. The current starting point is `001_core.sql`, which defines the complete proof-of-concept schema. Add future changes as sequential migrations; do not edit an established deployment's schema manually.

## Authentication

Passkey-only authentication uses WebAuthn. The first visitor creates the initial account. Additional users join through authenticated invitation links. Application routes require a session except static and authentication endpoints. See `DESIGN.md` for the onboarding flow and security decisions.

## Tests and formatting

Run before committing:

```sh
gofmt -w <changed Go files>
go test ./...
go vet ./...
```

For a clean integration environment:

```sh
cd test/integration
docker compose up --build
```

## Conventions

- Keep reusable FAIR and chart logic outside `internal/`.
- Validate user input in handlers before persistence.
- Full page requests render `layout.html`; HTMX requests return fragments.
- Use Go method-aware `net/http` routes.
- Keep assets embedded so the server deploys as one binary.
- Record entity changes in the audit log.
