# Alert simulator (dev only)

Local development service that sends realistic monitoring alerts to a running Aegis instance
through the **public alert webhook** and configures demo routing through the **HTTP API**.

This code is **not** part of the production API or worker binaries. It lives under `devtools/`
and is excluded from production Docker images.

## Prerequisites

- Aegis API and worker running
- `WEBHOOK_SECRET` in `.env`
- `DEV_AUTH_ENABLED=true` (for `-bootstrap`)

## Usage

From the repo root:

```bash
make dev-simulator-bootstrap   # create NOC/L1/L2/L3 teams, routing rules, escalation paths
make simulate-alert            # send one random alert (routes to a random tier)
make dev-simulator             # continuous loop (default 30s)
```

Or directly:

```bash
go run ./devtools/alert-simulator/cmd/alert-simulator -list
go run ./devtools/alert-simulator/cmd/alert-simulator -bootstrap
go run ./devtools/alert-simulator/cmd/alert-simulator -once
go run ./devtools/alert-simulator/cmd/alert-simulator -all   # one alert per tier scenario
```

Each built-in scenario sets a `team` label (`noc`, `l1`, `ops`, or `platform`) so incidents land on
the matching support tier. Use `-team platform` or `ALERT_SIM_TEAM=platform` to force a single tier.

## Docker (optional dev profile)

```bash
make up-dev              # full stack + alert simulator (foreground)
make up-dev-detached     # same, detached
```

Or directly:

```bash
docker compose -f deploy/docker-compose.yml --profile dev up --build
```

## Environment

| Variable | Purpose |
|----------|---------|
| `WEBHOOK_SECRET` | Alert webhook auth |
| `AEGIS_API_URL` | API base for bootstrap (default: `PUBLIC_URL` or `http://localhost:8080`) |
| `AEGIS_WEBHOOK_URL` | Full webhook URL (default: `{API}/api/v1/alerts/webhook`) |
| `ALERT_SIM_TEAM` | Override routing label for all scenarios (default: per-scenario tier) |
| `ALERT_SIM_INTERVAL` | Loop interval (default `30s`) |
