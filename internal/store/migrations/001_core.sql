-- 001_core.sql
-- Initial schema for the proof of concept.

CREATE TABLE risks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    identifier  TEXT NOT NULL UNIQUE,
    scenario    TEXT NOT NULL,
    lef_mode    TEXT NOT NULL DEFAULT 'decomposed',
    lef_min     REAL NOT NULL DEFAULT 0,
    lef_ml      REAL NOT NULL DEFAULT 0,
    lef_max     REAL NOT NULL DEFAULT 0,
    lef_rationale TEXT NOT NULL DEFAULT '',
    tef_min     REAL NOT NULL DEFAULT 0,
    tef_ml      REAL NOT NULL DEFAULT 0,
    tef_max     REAL NOT NULL DEFAULT 0,
    tef_rationale TEXT NOT NULL DEFAULT '',
    susc_min    REAL NOT NULL DEFAULT 0,
    susc_ml     REAL NOT NULL DEFAULT 0,
    susc_max    REAL NOT NULL DEFAULT 0,
    susc_rationale TEXT NOT NULL DEFAULT '',
    pl_prodl_min REAL NOT NULL DEFAULT 0,
    pl_prodl_ml  REAL NOT NULL DEFAULT 0,
    pl_prodl_max REAL NOT NULL DEFAULT 0,
    pl_prodl_rationale TEXT NOT NULL DEFAULT '',
    pl_respc_min REAL NOT NULL DEFAULT 0,
    pl_respc_ml  REAL NOT NULL DEFAULT 0,
    pl_respc_max REAL NOT NULL DEFAULT 0,
    pl_respc_rationale TEXT NOT NULL DEFAULT '',
    pl_replc_min REAL NOT NULL DEFAULT 0,
    pl_replc_ml  REAL NOT NULL DEFAULT 0,
    pl_replc_max REAL NOT NULL DEFAULT 0,
    pl_replc_rationale TEXT NOT NULL DEFAULT '',
    pl_finju_min REAL NOT NULL DEFAULT 0,
    pl_finju_ml  REAL NOT NULL DEFAULT 0,
    pl_finju_max REAL NOT NULL DEFAULT 0,
    pl_finju_rationale TEXT NOT NULL DEFAULT '',
    pl_repud_min REAL NOT NULL DEFAULT 0,
    pl_repud_ml  REAL NOT NULL DEFAULT 0,
    pl_repud_max REAL NOT NULL DEFAULT 0,
    pl_repud_rationale TEXT NOT NULL DEFAULT '',
    pl_cadvl_min REAL NOT NULL DEFAULT 0,
    pl_cadvl_ml  REAL NOT NULL DEFAULT 0,
    pl_cadvl_max REAL NOT NULL DEFAULT 0,
    pl_cadvl_rationale TEXT NOT NULL DEFAULT '',
    sl_prodl_min REAL NOT NULL DEFAULT 0,
    sl_prodl_ml  REAL NOT NULL DEFAULT 0,
    sl_prodl_max REAL NOT NULL DEFAULT 0,
    sl_prodl_rationale TEXT NOT NULL DEFAULT '',
    sl_respc_min REAL NOT NULL DEFAULT 0,
    sl_respc_ml  REAL NOT NULL DEFAULT 0,
    sl_respc_max REAL NOT NULL DEFAULT 0,
    sl_respc_rationale TEXT NOT NULL DEFAULT '',
    sl_replc_min REAL NOT NULL DEFAULT 0,
    sl_replc_ml  REAL NOT NULL DEFAULT 0,
    sl_replc_max REAL NOT NULL DEFAULT 0,
    sl_replc_rationale TEXT NOT NULL DEFAULT '',
    sl_finju_min REAL NOT NULL DEFAULT 0,
    sl_finju_ml  REAL NOT NULL DEFAULT 0,
    sl_finju_max REAL NOT NULL DEFAULT 0,
    sl_finju_rationale TEXT NOT NULL DEFAULT '',
    sl_repud_min REAL NOT NULL DEFAULT 0,
    sl_repud_ml  REAL NOT NULL DEFAULT 0,
    sl_repud_max REAL NOT NULL DEFAULT 0,
    sl_repud_rationale TEXT NOT NULL DEFAULT '',
    sl_cadvl_min REAL NOT NULL DEFAULT 0,
    sl_cadvl_ml  REAL NOT NULL DEFAULT 0,
    sl_cadvl_max REAL NOT NULL DEFAULT 0,
    sl_cadvl_rationale TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE requirements (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    identifier  TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE controls (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    identifier  TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'planned',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE gaps (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    identifier  TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    severity    TEXT NOT NULL DEFAULT 'medium',
    status      TEXT NOT NULL DEFAULT 'open',
    parent_type TEXT DEFAULT NULL,
    parent_id   INTEGER DEFAULT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE control_risks (
    control_id INTEGER REFERENCES controls(id) ON DELETE CASCADE,
    risk_id    INTEGER REFERENCES risks(id) ON DELETE CASCADE,
    PRIMARY KEY (control_id, risk_id)
);

CREATE TABLE control_requirements (
    control_id     INTEGER REFERENCES controls(id) ON DELETE CASCADE,
    requirement_id INTEGER REFERENCES requirements(id) ON DELETE CASCADE,
    PRIMARY KEY (control_id, requirement_id)
);

CREATE TABLE control_functions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    control_id   INTEGER NOT NULL REFERENCES controls(id) ON DELETE CASCADE,
    function     TEXT NOT NULL,
    cap_min      REAL NOT NULL DEFAULT 0,
    cap_ml       REAL NOT NULL DEFAULT 0,
    cap_max      REAL NOT NULL DEFAULT 0,
    cap_rationale TEXT NOT NULL DEFAULT '',
    cov_min      REAL NOT NULL DEFAULT 0,
    cov_ml       REAL NOT NULL DEFAULT 0,
    cov_max      REAL NOT NULL DEFAULT 0,
    cov_rationale TEXT NOT NULL DEFAULT '',
    rel_min      REAL NOT NULL DEFAULT 0,
    rel_ml       REAL NOT NULL DEFAULT 0,
    rel_max      REAL NOT NULL DEFAULT 0,
    rel_rationale TEXT NOT NULL DEFAULT '',
    notes        TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(control_id, function)
);

CREATE TABLE audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id   INTEGER NOT NULL,
    identifier  TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    field       TEXT NOT NULL DEFAULT '',
    old_value   TEXT NOT NULL DEFAULT '',
    new_value   TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_type ON audit_log(entity_type, created_at);

CREATE TABLE users (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    username     TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE webauthn_credentials (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id            INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id      BLOB NOT NULL UNIQUE,
    public_key         BLOB NOT NULL,
    attestation_type   TEXT NOT NULL DEFAULT '',
    transport          TEXT NOT NULL DEFAULT '[]',
    flags_json         TEXT NOT NULL DEFAULT '{}',
    authenticator_json TEXT NOT NULL DEFAULT '{}',
    attestation_json   TEXT NOT NULL DEFAULT '{}',
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE TABLE invitations (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT NOT NULL UNIQUE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invited_by INTEGER NOT NULL REFERENCES users(id),
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invitations_token ON invitations(token);
