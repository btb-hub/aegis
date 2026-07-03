# Feature: Web auth & session

Implements PRD §2 (`REQ-AUTH-*`) for the **browser UI**. API OIDC and sessions shipped in Phase 0
([AEG-005](../../backlog/epics/EPIC-01-foundation.md)); this feature closes the gap for human sign-in
from `apps/web`.

**Backlog:** Phase 3.5 — [AEG-057](../../backlog/epics/EPIC-01-foundation.md) (login page),
[AEG-058](../../backlog/epics/EPIC-01-foundation.md) (session + shell), [AEG-059](../../backlog/epics/EPIC-01-foundation.md) (callback redirect).

## Problem

The API supports OIDC login, but the web app has no login screen. Users hitting API-backed pages
(e.g. Integrations) see auth errors and are told to open `/auth/google/login` manually. Phase 0 exit
criteria covered **API** auth only.

## Solution

```text
/login -> user picks provider -> GET /auth/{provider}/login -> IdP
     -> GET /auth/{provider}/callback -> session cookie -> redirect to app
App load -> GET /auth/me -> shell shows user + Sign out
Protected routes -> redirect to /login if no session
```

## Dev login (local only)

When `DEV_AUTH_ENABLED=true` and `PUBLIC_URL` is localhost, the API exposes dev auth routes and the
login page shows **Dev sign in**. This creates a real `users` row and session cookie — same middleware
as OIDC. **Never enable in production.**

```text
/login -> Dev sign in -> GET /auth/dev/login?role=admin -> session cookie -> app
GET /auth/dev/status -> { "enabled": true }   (no session required)
```

Disable by unsetting `DEV_AUTH_ENABLED` or setting it to `false`. `GET /auth/dev/login` returns `404`
when disabled.

## Login page

- Three provider buttons only (no local password).
- Same-origin links so the session cookie works on `localhost:3000` (nginx proxy in Docker) and Vite dev.
- Unsigned visitors: locale from browser / language switcher ([REQ-I18N-03](../../docs/01-prd.md)).

## Session in the shell

- `GET /auth/me` on load; poll or refetch after login redirect.
- Header: display name, role badge optional for admin, **Sign out**.
- `POST /auth/logout` clears server session and client state.

## Protected routes

Initial set (expand as pages wire to API):

- `/integrations` (admin integrations list + test connection)
- Future: shifts/incidents admin views, alerting workspace, setup wizard

Demo-only pages (shifts/incidents fixtures) stay public until wired to API in later stories.

## Callback redirect (AEG-059)

After successful OIDC callback, the API sets `aegis_session` and responds with `302` to `PUBLIC_URL`
(unless `?format=json` is passed for API clients). Set `PUBLIC_URL` to the web app origin
(e.g. `http://localhost:3000` with Vite or Docker nginx) so the browser lands signed in.

## References

- Security flow: [`09-security.md`](../09-security.md)
- API routes: [`04-api-spec.md`](../04-api-spec.md)
- Roadmap: [`roadmap.md`](../../backlog/roadmap.md) — Phase 3.5
