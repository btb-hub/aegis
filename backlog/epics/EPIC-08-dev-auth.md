# EPIC-08 — Local dev auth

**Phase:** 7 (post-MVP developer experience)  
**Exit:** Engineers can open Alerts, Integrations, Dashboard, and Setup locally without registering an OIDC app.

**Problem:** Protected web routes require a session from Google/Slack/eXpress OIDC. Local `.env` files
often have empty OIDC keys, so `/login` cannot complete and API-backed pages stay unreachable.

**Solution:** Opt-in dev login behind `DEV_AUTH_ENABLED=true`. Creates a real DB user + session (same
cookie and middleware as OIDC). Disabled by default; returns 404 when off. Does **not** add passwords
or a new IdP — it reuses the existing session model.

**Out of scope:** Production auth changes, username/password login, bypassing RBAC, disabling auth in CI
tests (handler tests keep using mocks/`seedAdmin`).

---

### AEG-060 — Dev auth config and guardrails

- **Status:** Done
- **Depends on:** AEG-005
- **PRD:** REQ-AUTH-01 (exception: dev-only; OIDC remains the only production path)
- **Acceptance:**
  - [x] `DEV_AUTH_ENABLED` parsed in `pkg/config` (default off)
  - [x] Optional `DEV_AUTH_DEFAULT_ROLE` (`admin` | `member` | `viewer`; default `admin`)
  - [x] Optional `DEV_AUTH_EMAIL` (default `dev@localhost`)
  - [x] API refuses to enable dev auth when `PUBLIC_URL` host is not localhost / `127.0.0.1` / `[::1]`
  - [x] Startup logs a clear warning when dev auth is on
  - [x] Keys documented in `deploy/.env.example` and `deploy/.env.local.example`
  - [x] Unit tests for config parsing and host guard

---

### AEG-061 — Dev login API

- **Status:** Done
- **Depends on:** AEG-060
- **PRD:** REQ-AUTH-03, REQ-AUTH-04
- **Acceptance:**
  - [x] `GET /auth/dev/status` returns `{ "enabled": true|false }` (always 200; no session required)
  - [x] `GET /auth/dev/login` when enabled: upsert user `(provider=dev, provider_sub=dev-local)`,
        create session, set `aegis_session` cookie, redirect to `PUBLIC_URL` (or `?redirect=` path)
  - [x] `?role=` query overrides default role for the upserted dev user (`admin`, `member`, `viewer`)
  - [x] When disabled: login returns `404`; status `{ "enabled": false }`
  - [x] `AuthService.DevLogin(ctx, role)` covered by unit tests
  - [x] Handler tests: enabled issues cookie + redirect; disabled returns 404; admin role can hit
        `POST /api/v1/setup/test-alert`
  - [x] Route documented in `docs/04-api-spec.md` under a **Development only** section

---

### AEG-062 — Dev sign-in on login page

- **Status:** Done
- **Depends on:** AEG-061, AEG-057
- **PRD:** REQ-AUTH-01
- **Acceptance:**
  - [x] Login page fetches `GET /auth/dev/status` on load
  - [x] When `enabled: true`, show a **Dev sign in** button below OIDC providers (secondary style)
  - [x] Button links to `/auth/dev/login?role=admin` (same-origin)
  - [x] Copy in `en` and `ru` locale files (`auth.dev_sign_in`, helper text that this is local only)
  - [x] Vitest: button hidden when status disabled; visible when enabled
  - [x] No dev button when API returns `enabled: false`

---

### AEG-063 — Local dev runbook

- **Status:** Done
- **Depends on:** AEG-060, AEG-061, AEG-062
- **PRD:** NFR-1 (local testing path)
- **Acceptance:**
  - [x] `docs/07-setup-deployment.md` — section **Local testing without OIDC** with env snippet and
        one-click flow
  - [x] `docs/features/web-auth.md` — dev login flow; explicit “not for production”
  - [x] `README.md` — mention dev auth under native dev quick start
  - [x] Setup wizard OIDC step notes that dev sign-in satisfies the check locally
