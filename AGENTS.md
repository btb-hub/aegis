# AGENTS.md

For the full development loop, conventions, and Definition of Done, read [`CLAUDE.md`](./CLAUDE.md).
Standard commands live in the [`Makefile`](./Makefile); app-specific docs are under [`docs/`](./docs).

## Cursor Cloud specific instructions

These notes cover non-obvious startup/run caveats in the Cursor Cloud VM. The update script already
refreshes dependencies (`make deps`: Go module downloads + `npm install`). System tooling (Go 1.25,
PostgreSQL 16, the `migrate` CLI) is baked into the VM image, not the update script.

### Toolchain

- **Docker is NOT installed.** Makefile targets that use Compose (`make up`, `make dev-db`,
  `make migrate-docker`, `make up-dev`) will not work here. Use the native Postgres + native process
  path described below instead.
- **Go 1.25** lives at `/usr/local/go` and is symlinked into `/usr/local/bin`, so it wins over the
  distro's `/usr/bin/go` (1.22). The modules require 1.25 (`go.work`), so don't reorder PATH such that
  the 1.22 binary takes precedence.
- The `migrate` CLI (golang-migrate v4.17.1) is on PATH at `/usr/local/bin/migrate`.

### Postgres (start it yourself — it does not auto-start)

The DB runs as a native cluster, not in Docker, and is not started on VM boot. Start it before
running the API/worker, tests that touch the DB, or migrations:

```bash
sudo pg_ctlcluster 16 main start   # check state with: pg_lsclusters
```

Connection: `postgres://aegis:aegis@localhost:5432/aegis?sslmode=disable` (role `aegis`, db `aegis`).

### Local dev run (native, no Docker)

1. Create `.env` once (native template, with dev login enabled):
   `cp deploy/.env.local.example .env` then set `DEV_AUTH_ENABLED=true` and non-empty
   `SESSION_SECRET` / `WEBHOOK_SECRET`.
2. Apply migrations (Postgres must be running):
   `export DATABASE_URL="postgres://aegis:aegis@localhost:5432/aegis?sslmode=disable"` then
   `make migrate-up`.
3. Run each service in its own shell: `make dev-api` (`:8080`), `make dev-worker` (no port),
   `make dev-web` (`:3000`; Vite proxies `/api` and `/auth` to the API). Go targets load `.env` via
   `scripts/load-env.sh`.
4. Log in: open `http://localhost:3000` and click **Dev sign in** (only present when
   `DEV_AUTH_ENABLED=true`).

### Exercising the alert → incident pipeline

`make dev-simulator-bootstrap` (creates NOC/Helpdesk/Ops/Platform teams + routing) then
`make simulate-alert` posts an alert to the webhook; the worker turns it into an incident visible on
the Incidents page. See [`devtools/alert-simulator/README.md`](./devtools/alert-simulator/README.md).

### Verification gate

`make lint type test` is the gate. `make test` does not need Postgres (Go tests use mocks/fixtures).
`make lint` also builds Storybook, so it needs the web dependencies installed.

### Gotchas

- `npm install` (used by `make deps`) may rewrite `apps/web/package-lock.json` with a benign
  normalization; it is safe to `git checkout -- apps/web/package-lock.json`. `npm ci` installs
  without touching the lockfile if you want a clean tree.
