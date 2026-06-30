# EPIC-01 — Foundation

**Phase:** 0  
**Exit:** App runs locally, alert stored, OIDC auth works, CI green, Storybook documents base UI components.

---

### AEG-001 — Monorepo layout and Makefile

- **Status:** Done
- **Depends on:** —
- **PRD:** NFR-3
- **Acceptance:**
  - [x] `apps/api`, `apps/worker`, `apps/web`, `deploy/`, `db/` directories exist
  - [x] Root `Makefile` with `lint`, `type`, `test` targets (may noop until wired)
  - [x] Go modules for api and worker; `package.json` for web
  - [x] `sqlc.yaml` and golang-migrate path documented in README

**Plan (implemented):** Monorepo with `go.work`, `pkg/`, `apps/{api,worker,web}`, `db/migrations`, root `Makefile` (`lint`, `type`, `test` with ≥90% business-logic coverage gate).

---

### AEG-002 — Docker Compose dev stack

- **Status:** Done
- **Depends on:** AEG-001
- **PRD:** NFR-1
- **Acceptance:**
  - [x] `deploy/docker-compose.yml` with Postgres 16, api, worker, web
  - [x] No Redis service
  - [x] `docker compose up` starts all services
  - [x] Health endpoints reachable when api story complete

---

### AEG-003 — Config package and env example

- **Status:** Done
- **Depends on:** AEG-001
- **PRD:** NFR-4
- **Acceptance:**
  - [x] `deploy/.env.example` lists all required keys (DB, session, OIDC x3, webhook)
  - [x] Typed config struct loaded once at startup in api and worker
  - [x] Missing required env fails fast with clear error

---

### AEG-004 — Migrations, sqlc, and jobs table

- **Status:** Done
- **Depends on:** AEG-002, AEG-003
- **PRD:** REQ-INC-02
- **Acceptance:**
  - [x] Initial migration: `users` (incl. `locale`), `sessions`, `jobs`, `alerts` baseline
  - [x] `make migrate-up` / `migrate-down` work
  - [x] sqlc generates Go from `db/query/` (`pkg/db` store; run `sqlc generate` when CLI available)
  - [x] Worker can claim a job row with `SKIP LOCKED`

---

### AEG-005 — OIDC auth and RBAC skeleton

- **Status:** Done
- **Depends on:** AEG-004
- **PRD:** REQ-AUTH-01, REQ-AUTH-02, REQ-AUTH-03, REQ-AUTH-04, REQ-I18N-02
- **Acceptance:**
  - [x] Login/callback/logout for Google, Slack, eXpress
  - [x] Server-side session cookie; `GET /auth/me` returns user + role + locale
  - [x] `PATCH /auth/me` updates `locale` (`en` \| `ru`)
  - [x] Role middleware skeleton (`admin`, `member`, `viewer`)
  - [x] No local password flow

---

### AEG-006 — Health and metrics endpoints

- **Status:** Done
- **Depends on:** AEG-004
- **PRD:** —
- **Acceptance:**
  - [x] `GET /healthz` returns 200
  - [x] `GET /readyz` checks Postgres
  - [x] `GET /metrics` exposes Prometheus handler

---

### AEG-007 — Generic alert webhook

- **Status:** Done
- **Depends on:** AEG-004, AEG-006
- **PRD:** REQ-INC-01, REQ-INC-02
- **Acceptance:**
  - [x] `POST /api/v1/alerts/webhook` validates secret and payload
  - [x] Stores `alerts` row + enqueues `process_alert` job
  - [x] Returns `202` with alert id
  - [x] Worker stub logs job (full processing in EPIC-04)

---

### AEG-008 — CI gate and PR template

- **Status:** Done
- **Depends on:** AEG-001
- **PRD:** —
- **Acceptance:**
  - [x] GitHub Actions: golangci-lint, `go test`, eslint, `tsc`
  - [x] `.github/pull_request_template.md` present
  - [x] CI runs on PR to main

---

### AEG-054 — Web i18n scaffold (en + ru)

- **Status:** Done
- **Depends on:** AEG-001
- **PRD:** REQ-I18N-01, REQ-I18N-03, REQ-I18N-05, REQ-I18N-06
- **Acceptance:**
  - [x] `react-i18next` wired in `apps/web` with `en` and `ru` locale JSON namespaces
  - [x] Language switcher (English / Русский) in app shell; persists to `localStorage`
  - [x] `pkg/i18n` Go package with `en`/`ru` message files for worker chat templates
  - [x] CI test: every `en` locale key has a matching `ru` key (and vice versa)
  - [x] Sample page demonstrates translated strings and `Intl` date formatting

---

### AEG-055 — Design system tokens and base components

- **Status:** Done
- **Depends on:** AEG-001
- **PRD:** REQ-DS-01, REQ-DS-02, REQ-DS-03
- **Acceptance:**
  - [x] Tailwind theme maps tokens from [`docs/12-design-system.md`](../../docs/12-design-system.md) / [`design_system.html`](../../docs/design_system.html)
  - [x] IBM Plex Sans + IBM Plex Mono loaded; typography scale applied
  - [x] Base components: Button (primary/secondary/ghost), Input, Severity tag, Toast shell
  - [x] App shell layout: 240px sidebar, 56px header per design system
  - [x] Sample route demonstrates component states (P1 severity tag, primary button, `Intl` date)

---

### AEG-056 — Storybook for UI components

- **Status:** Done
- **Depends on:** AEG-055, AEG-054
- **PRD:** REQ-DS-04
- **Acceptance:**
  - [x] Storybook 8+ in `apps/web` (`npm run storybook` on port 6006)
  - [x] Stories for Button (primary/secondary/ghost, hover/disabled), Input, SeverityTag (P1–P4), Toast
  - [x] Stories for AppShell and LanguageSwitcher with `en` and `ru` decorators
  - [x] Tailwind + design tokens apply in Storybook preview (same theme as app)
  - [x] `make lint` / CI includes Storybook build (`npm run build-storybook`)
  - [x] New UI components added in feature work include a Storybook story in the same PR

**Plan (implemented):** Storybook 8 with Vite in `apps/web`; stories for base components; locale toolbar + `en`/`ru` layout stories; `build-storybook` in `make lint` / CI.

---

### AEG-057 — Web login page

- **Status:** In Review
- **Depends on:** AEG-005, AEG-055, AEG-054
- **PRD:** REQ-AUTH-01, REQ-I18N-01, REQ-DS-01
- **Acceptance:**
  - [ ] Dedicated login route (e.g. `/login`) reachable when unsigned
  - [ ] Buttons: Sign in with Google, Slack, eXpress — each links to `GET /auth/{provider}/login` on the same origin
  - [ ] Copy in `en` and `ru`; uses design-system `Button` components
  - [ ] Storybook story for login page (default + `ru` decorator)
  - [ ] Unsigned access to API-backed app routes redirects to login (integrations, future admin pages)

**Plan:** `LoginPage` in `apps/web`; provider list from config or static three providers; `react-router` or conditional render in `App` for `/login`.

---

### AEG-058 — Web session and app shell auth

- **Status:** In Review
- **Depends on:** AEG-057
- **PRD:** REQ-AUTH-03, REQ-AUTH-04, REQ-I18N-02
- **Acceptance:**
  - [ ] On app load, `GET /auth/me` with credentials; expose user + role in React context (or lightweight store)
  - [ ] App shell header shows signed-in display name (or email) and **Sign out** when session exists
  - [ ] Sign out calls `POST /auth/logout` and returns user to login
  - [ ] Integrations page loads data when signed in (remove manual `/auth/.../login` workaround)
  - [ ] Vitest tests for session provider and signed-out redirect

**Plan:** `AuthProvider` + `useAuth`; wire `IntegrationsPage` to session; header auth block in `AppShell`.

---

### AEG-059 — OIDC callback redirect to web

- **Status:** In Review
- **Depends on:** AEG-005, AEG-057
- **PRD:** REQ-AUTH-01
- **Acceptance:**
  - [ ] After successful OIDC callback, API responds with `302` redirect to `PUBLIC_URL` (or `/`) instead of JSON body
  - [ ] Session cookie still set on redirect; user lands in app signed in
  - [ ] Documented in [`docs/04-api-spec.md`](../../docs/04-api-spec.md) and [`docs/features/web-auth.md`](../../docs/features/web-auth.md)
  - [ ] API test: callback sets cookie and redirect location

**Plan:** Change `AuthHandler.callback` to redirect; use `PUBLIC_URL` from config; keep JSON path only if needed for API clients (optional query `?format=json`).

