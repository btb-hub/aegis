# Design: First-admin bootstrap via `ADMIN_EMAILS` + day-2 role management

**Date:** 2026-07-20  
**Status:** Draft for review  
**Context:** Deployed OIDC users are created as `member`. There is no production path to grant `admin`, so the first operator cannot configure teams, integrations, or setup. Dev auth must stay off in production.

## Goals

1. **Day 0:** An operator can become admin by listing their email in `.env` and signing in once.
2. **Day 2:** An existing admin can change other users’ roles in-product (`PATCH` + Users UI).
3. Preserve auditability (REQ-AUDIT-01) and avoid silent privilege that cannot be reasoned about.

## Non-goals

- Bootstrap CLI / one-shot token API.
- Creating stub users before first OIDC login.
- Self-service role changes on `/account`.
- Changing `team_role` (team membership) via this feature.
- Auto-demotion when an email is removed from `ADMIN_EMAILS` (removal only stops re-promotion).

## Decisions (from brainstorming)

| Topic | Choice |
|-------|--------|
| Day-0 mechanism | Env allowlist `ADMIN_EMAILS` |
| Allowlist semantics | Always-on: matching emails are forced to `admin` on OIDC login |
| Day-2 mechanism | Admin API + Users UI |
| Env vs UI demote | Env wins for listed emails; API rejects demote of listed emails |
| Apply timing | On OIDC login only (not every request middleware) |

## Architecture

```text
.env ADMIN_EMAILS ──► config.AdminEmails (normalized set)
                            │
OIDC CompleteLogin ─────────┼──► if email ∈ set && role ≠ admin
                            │         UpdateUserRole(admin) + audit
                            ▼
                     session cookie (user already admin)

Admin UI / PATCH /users/{id} ──► UpdateUserRole (guards below)
```

Session middleware already loads `users.role` from the DB on each request, so non-env role changes take effect on the next API call without requiring re-login. Env re-promotion still runs only on the next OIDC login.

---

## §1 — Config: `ADMIN_EMAILS`

**Variable:** `ADMIN_EMAILS`  
**Format:** comma-separated emails, e.g. `alice@company.com,bob@company.com`  
**Empty / unset:** no auto-promotion.

**Parsing (startup):**
- Split on `,`
- Trim whitespace
- Lower-case
- Drop empty tokens
- Invalid tokens (no `@`) → config load error (fail fast)

Document in `deploy/.env.example`, `deploy/.env.local.example`, and `docs/07-setup-deployment.md`.

---

## §2 — Login promotion

**Where:** `AuthService.CompleteLogin`, after `ResolveOIDCLogin` returns the user and before the session is created.

**Algorithm:**
1. Normalize user email the same way as config parsing.
2. If email ∈ `AdminEmails` and `user.Role != admin`:
   - Persist `role = admin`
   - Append `audit_log` action `user.role_changed` with details: `old_role`, `new_role`, `reason: admin_emails_env`
3. If already admin → no-op (no audit noise).
4. Create session for the **updated** user so the first cookie has admin.

**Dev auth:** unchanged (`?role=` / `DEV_AUTH_DEFAULT_ROLE`). Do **not** apply `ADMIN_EMAILS` to the fixed `dev` provider user unless we later decide otherwise — keeps local RBAC testing predictable.

**Not applied on:** `GET /auth/me`, request middleware, worker jobs.

---

## §3 — Day-2 API

Builds on existing admin `GET /api/v1/users` (AEG-065).

### `PATCH /api/v1/users/{id}`

- **Authz:** session + `RequireAdmin`
- **Body:** `{ "role": "admin" | "member" | "viewer" }`
- Validate with `pkg/rbac.Parse`
- **Last-admin rule:** if changing a user away from `admin` would leave zero admins → `409` with a stable error code (e.g. `last_admin`)
- **Env pin rule:** if target user’s email ∈ `ADMIN_EMAILS` and requested role ≠ `admin` → `409` with code e.g. `admin_emails_pinned` and a message that the env list owns that account
- Idempotent: same role → `200` with current user, no audit
- Audit on real change: `user.role_changed` with actor user id, `old_role`, `new_role`, `reason: admin_api`
- Document in `docs/04-api-spec.md`

### DB

- New store method `UpdateUserRole(ctx, id, role)` (and `CountUsersByRole` or equivalent for last-admin check)
- No migration required (`users.role` already exists)

---

## §4 — Day-2 UI

**Route:** `/users` (admin-only mutations; page itself admin-gated in nav and route guard)

**Contents:**
- Table backed by `GET /api/v1/users` (existing `q`, paging)
- Columns: display name, email, role, optional “Pinned by config” when email is known to be env-managed (see note below)
- Role control: select → `PATCH`; toast “Role updated” / clear error from API body
- Copy in `en` + `ru`
- Vitest for promote/demote happy path and pinned-demote error
- Storybook for shared role control if extracted

**Pinned badge:** Include `role_pinned: true` on user JSON when email ∈ server `ADMIN_EMAILS` (do not ship the raw env list to the browser). Show a short “Pinned by config” hint and disable demote in the role control.

Nav: admin-only top-level **Users** entry, matching design system.

---

## §5 — Docs & interim ops

**Deploy runbook (“First admin”):**
1. Set `ADMIN_EMAILS=you@company.com` in `.env`
2. Restart API
3. Sign in with OIDC using that email
4. Confirm `GET /auth/me` → `role: admin`

**Interim / recovery** (if env was wrong before first login, or API down):

```sql
UPDATE users SET role = 'admin' WHERE email = 'you@company.com' RETURNING id, email, role;
```

Then sign out/in if an old session somehow cached role (normally unnecessary because session middleware reloads from DB).

**Security docs:** note that `ADMIN_EMAILS` is a standing privilege source; treat like other secrets-adjacent ops config (not a credential, but access-defining).

---

## §6 — Testing

| Layer | Cases |
|-------|--------|
| Config | Parse list; trim/lower; reject bad tokens; empty OK |
| Auth service | Listed member → admin + audit; listed admin → no-op; unlisted unchanged |
| User service | PATCH promote/demote; last admin 409; pinned demote 409; non-admin caller N/A at handler |
| Handler | 403 member; 200 admin; 409 bodies |
| Web | Role change success toast; pinned error surfaced |

---

## §7 — Rollout / stories

Ship as **two** small stories (separate PRs preferred):

1. **Bootstrap:** `ADMIN_EMAILS` config + login promotion + docs (+ interim SQL note). Unblocks production immediately.
2. **Role management:** `PATCH /users/{id}` + Users UI + last-admin + env-pin guards + `role_pinned` on list/get.

Backlog: append stories under an auth/users epic (EPIC-09 or a small follow-on); do not rewrite existing Done acceptance criteria.

---

## Error handling (summary)

| Situation | Response |
|-----------|----------|
| Non-admin PATCH | `403` |
| Invalid role | `400` validation |
| Last admin demoted | `409` `last_admin` |
| Env-listed demoted | `409` `admin_emails_pinned` |
| Unknown user id | `404` |

UI errors use the API `message` — what happened and how to fix (e.g. remove email from `ADMIN_EMAILS` and restart, then demote).

## Resolved details

- Env var name: `ADMIN_EMAILS` (unprefixed, consistent with `SESSION_SECRET` / `WEBHOOK_SECRET`).
- No `GET /users/{id}` required for MVP — list + patch is enough.
- `role_pinned` on list responses is required for story 2 UX.
