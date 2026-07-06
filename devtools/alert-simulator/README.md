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
make dev-simulator-bootstrap   # create Platform team + routing rule via API
make simulate-alert            # send one random alert
make dev-simulator             # continuous loop (default 30s)
```

Or directly:

```bash
go run ./devtools/alert-simulator/cmd/alert-simulator -list
go run ./devtools/alert-simulator/cmd/alert-simulator -bootstrap
go run ./devtools/alert-simulator/cmd/alert-simulator -once
```

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
| `ALERT_SIM_TEAM` | Routing label (default `platform`) |
| `ALERT_SIM_INTERVAL` | Loop interval (default `30s`) |
