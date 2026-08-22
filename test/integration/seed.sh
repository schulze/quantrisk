#!/bin/sh
set -eu

DB="${DB_PATH:-/data/quantrisk.db}"
SERVER="${SERVER_URL:-http://quantriskd:8000}"
CLI="quantriskcli -db $DB"

until curl -sf "$SERVER/login" >/dev/null 2>&1; do
    sleep 1
done

$CLI seed -user seed "Seed Automation"

echo "seed complete"
