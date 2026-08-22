# secriskquant

> **Experiment:** This project primarily explores a tool-based approach to FAIR and FAIR-CAM security risk management. It also serves as a place for experiments with agentic coding, prototyping, and exploration.

secriskquant is a proof of concept for quantitative security risk analysis (FAIR) and control management (FAIR-CAM). It implements loss event scenarios, probability distributions, Monte Carlo simulation, and loss exceedance curves.

## What it does

- Models security loss scenarios with FAIR estimates.
- Runs reproducible Monte Carlo simulations and renders loss exceedance curves.
- Maps controls to risks and requirements.
- Classifies controls with FAIR-CAM functions and effectiveness estimates.
- Tracks control gaps and change history.
- Provides a small authenticated HTMX web interface backed by SQLite.

## What is missing

- Connection of control gaps and FAIR-CAM function control effectiveness with scenario estimates for simulations or in the data model.

## Components

```text
quantriskd    Web application and HTTP server
quantriskcli  Database administration, imports, and simulations
seed          Realistic proof-of-concept fixture data
```

## Quick start

```sh
# Build
go build ./cmd/quantriskd
go build ./cmd/quantriskcli
go build ./cmd/seed

# Start with a fresh database
./quantriskd -db quantrisk.db -addr :8000

# Optionally load sample data
./seed -db quantrisk.db

# Import scenarios from CSV
./quantriskcli -db quantrisk.db import -risks test/fixtures/test.csv

# Run a simulation from CSV
./quantriskcli simulate -file test/fixtures/test.csv
```

Open `http://localhost:8000`. The first visitor creates the initial passkey account.

## Service deployment

The tracked `srv.service.example` contains publishable placeholders for the
service account, database path, and WebAuthn domain. Create a local service file
and adjust it for the deployment:

```sh
make service-config
$EDITOR srv.service
make deploy
```

`srv.service` is ignored by git, so environment-specific domains and paths stay
local while the example remains safe to publish.

## Integration environment

```sh
cd test/integration
docker compose up --build
```

See `test/integration/README.md` for details.

## Background

The project began as a Go reimplementation of Netflix's `riskquant` library. It grew into a FAIR and FAIR-CAM web application while learning about the FAIR model and experimenting with implementation of FAIR-CAM ideas.

## License

Apache 2.0 — see `LICENSE.txt`.

Original riskquant: Copyright 2019–2020 Netflix, Inc.  
Extensions: Copyright 2022-2026 Frithjof Schulze.
