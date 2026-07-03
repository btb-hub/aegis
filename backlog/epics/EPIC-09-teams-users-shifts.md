# EPIC-09 — Teams, users & shifts setup

**Phase:** 8 (post-MVP operational setup)  
**Exit:** An admin can create a team, add members (including users who signed in via SSO), define a
weekly rotation and overrides in the UI, and see real on-call data on the shifts calendar — locally
with dev seeds or after real OIDC sign-ins.

## Problem

Phase 1 shipped **backend** teams, schedules, overrides, and on-call APIs ([EPIC-02](./EPIC-02-shifts.md)),
and Phase 1 UI stories delivered **presentational** calendar components wired to **demo fixtures**
(`apps/web/src/lib/shiftsDemoData.ts`). There is no UI to create teams or schedules, and the shifts
page never calls `/api/v1/teams/*`.

With Phase 7 dev auth, a single admin can reach protected pages, but they still cannot configure
on-call: schedule `participants` must be team-member user UUIDs, and there is no member picker or
team management screen.

SSO user provisioning is **partially** implemented: `AuthService.CompleteLogin` upserts on
`(provider, provider_sub)`, but `apps/api/internal/oidc/client.go` still returns stub userinfo instead
of parsing the OIDC id_token / userinfo response. Profile fields needed for shifts UI and paging
(avatar, Slack user id) are not populated on login. There is no admin **list users** API and no
**dev seed** script for realistic multi-provider users.

**Multi-provider gap:** Today each `(provider, provider_sub)` is a separate `users` row. Someone who
signs in with Google then Slack gets two accounts. We need one canonical user per person, multiple
linked identities, and **fill-if-empty backfill** on link — never overwrite profile fields the user
(or admin) set manually.

**Account page gap:** No `/account` route; locale lives in `localStorage` only; express `/link` has API
but no UI. See [`docs/features/account.md`](../docs/features/account.md).

## Solution

Close the gap between Phase 1 API and the web app in five tracks:

1. **Identity model + OIDC backfill** — `user_identities` table; login links providers by email;
   fill-if-empty only; real id_token parsing; `avatar_url` on users.
2. **Users directory** — admin list/search API for member pickers.
3. **Dev seeds** — idempotent local seed of SSO-like users with identities + profile fields.
4. **Account page** — profile, locale sync, connected providers, eXpress link UI.
5. **Teams + shifts UI** — team CRUD and membership UI; wire shifts calendar to API.

**Out of scope:** Incidents list/detail API wiring (still demo in `App.tsx`), Mattermost/Telegram,
changing RBAC model, production IdP changes, Helm/K8s.

---

### AEG-064 — OIDC identities, backfill, and profile fields

- **Status:** In Review
- **Depends on:** AEG-005
- **PRD:** REQ-AUTH-01, REQ-AUTH-03; [`09-security.md`](../docs/09-security.md) flow step 3–4
- **Acceptance:**
  - [x] Migration `user_identities` (`user_id`, `provider`, `provider_sub`, `linked_at`; unique on
        `(provider, provider_sub)` and on `(user_id, provider)`)
  - [x] Backfill migration: copy existing `users.provider` / `users.provider_sub` into `user_identities`
  - [x] Migration adds nullable `avatar_url TEXT` on `users`
  - [x] Replace stub `UserInfoFromToken` with id_token parsing (`sub`, `email`, `name`, `picture`); Slack
        claim for `slack_user_id` when present
  - [x] **Login resolution order:**
        1. Match `user_identities` on `(provider, provider_sub)` → session for that user
        2. Else match `users` on normalized `email` (case-insensitive) → **link** new identity to existing
           user; apply backfill rules; session for that user
        3. Else create `users` row + identity; default role `member`, locale `en`
  - [x] **Backfill rules (fill-if-empty only — never overwrite non-empty):**
        - `display_name`, `avatar_url`: set only when currently empty/null
        - `slack_user_id`: set only when null (from Slack OIDC claims)
        - `express_user_huid`: unchanged on OIDC login (still via `/link` unless claim added later)
        - `email`, `role`, `locale`: never updated from OIDC merge/link
  - [x] **Same-provider re-login:** same backfill rules (no clobber)
  - [x] **Manual overwrite:** only via `PATCH /auth/me` (display_name) or future admin `PATCH /users/{id}`;
        document that OIDC never overwrites user-edited fields
  - [x] `users.provider` retained as `primary_provider` (first linked provider) for compat; deprecate as
        login key in docs
  - [x] `GET /auth/me` includes `avatar_url`, `slack_user_id`, `express_user_huid`, `identities[]`
  - [x] Unit tests: Google-first then Slack link enriches same user; existing display_name not overwritten;
        recorded id_token fixtures (no live IdP in CI)
  - [x] Audit log entry on new identity link (`REQ-AUDIT-01`)

**Plan:** `AuthService.CompleteLogin` → `Store.ResolveOIDCLogin`; store methods in `pkg/db/identity.go`;
update [`03-data-model.md`](../docs/03-data-model.md). Branch: `feat/identity-AEG-064-oidc-backfill`.

---

### AEG-065 — Users list API (admin)

- **Status:** Ready
- **Depends on:** AEG-064
- **PRD:** REQ-SHIFT-01 (admins assign members)
- **Acceptance:**
  - [ ] `GET /api/v1/users` — session + admin; paginated list with `q` search on email/display_name
  - [ ] Response items: `id`, `email`, `display_name`, `role`, `avatar_url`, `identities[]` (or
        `providers[]`), `slack_user_id`, `express_user_huid`
  - [ ] Default sort: `display_name` asc; `page` / `page_size` (max 100)
  - [ ] Documented in `docs/04-api-spec.md`
  - [ ] Handler + service tests including authz (member gets 403)

**Plan:** Store query + `UserService.ListUsers`; `UserHandler` registered in API main; sqlc optional
(raw query in store is fine if faster).

---

### AEG-066 — Dev user seeds

- **Status:** In Progress
- **Depends on:** AEG-064
- **PRD:** NFR-1 (local setup); supports REQ-SHIFT-01 local testing
- **Acceptance:**
  - [x] `make seed-dev` (or `go run ./cmd/seed-dev`) upserts a fixed set of users idempotently
  - [x] At least: 1 Google, 1 Slack (with `slack_user_id` + avatar), 1 eXpress (with `express_user_huid` +
        avatar), 1 admin-capable dev-style row optional
  - [x] Each seed user has: `email`, `display_name`, `avatar_url`, plus `user_identities` rows for
        google/slack/express as appropriate; Slack/eXpress include provider-specific ids
  - [x] Avatars use stable public placeholder URLs (no secrets, no binary blobs in repo)
  - [x] Documented in `README.md` and `docs/07-setup-deployment.md` under local dev
  - [x] Seed does not run in production (guard: `PUBLIC_URL` localhost or explicit `SEED_DEV=1`)

**Plan:** `pkg/db/seed_dev.go` with idempotent `UpsertSeedDevUser`; `apps/api/cmd/seed-dev` CLI;
`make seed-dev` via Makefile. Guard with localhost `PUBLIC_URL` or `SEED_DEV=1`. Branch:
`feat/users-AEG-066-dev-users-seed`.

---

### AEG-067 — Teams management UI

- **Status:** Ready
- **Depends on:** AEG-065, AEG-058, AEG-056
- **PRD:** REQ-SHIFT-01
- **Acceptance:**
  - [ ] `/teams` page (admin mutations; all roles can list): create team (name, description), edit, delete
  - [ ] Team detail: list members; add member via user search picker (`GET /users?q=`); remove member;
        optional `team_role` (`member` | `lead`)
  - [ ] Empty state when no teams: CTA to create first team
  - [ ] Nav entry or shifts breadcrumb links to team list
  - [ ] Copy in `en` + `ru`; Vitest for create flow and member add; Storybook for `TeamMemberPicker` (or
        equivalent shared component)

**Plan:** TanStack Query hooks for teams/members/users APIs; reuse design-system `Button`, `Input`, `Modal`;
admin-only action buttons gated on `user.role === 'admin'`.

---

### AEG-068 — Shifts page wired to API

- **Status:** Ready
- **Depends on:** AEG-067, AEG-014, AEG-015
- **PRD:** REQ-SHIFT-06, REQ-SHIFT-07
- **Acceptance:**
  - [ ] Remove `shiftsDemoData` from production route; `TeamShiftsPage` loads team by id (route
        `/teams/:teamId/shifts` or team selector on `/shifts`)
  - [ ] Fetches `GET /teams/{id}/on-call/current` and `GET /teams/{id}/on-call/calendar?from&to`
  - [ ] `OnCallBanner` and `ShiftsCalendar` render API data; loading and error states
  - [ ] Redirect or empty state when team has no schedule yet
  - [ ] Vitest with mocked fetch; update existing App tests

**Plan:** API client helpers in `apps/web/src/lib/shiftsApi.ts`; map API slot shape to existing
`CalendarSlot` types where possible.

---

### AEG-069 — Schedule and override admin UI

- **Status:** Ready
- **Depends on:** AEG-068, AEG-010, AEG-011
- **PRD:** REQ-SHIFT-02, REQ-SHIFT-03
- **Acceptance:**
  - [ ] Admin: create/edit weekly schedule (name, IANA timezone, handoff weekday/time, ordered participants
        from team members)
  - [ ] Admin: create/delete override (user, start/end in team timezone); validation errors surfaced in UI
  - [ ] Successful mutations refresh calendar; toast on success
  - [ ] Copy in `en` + `ru`; Vitest for schedule form validation; Storybook for override dialog

**Plan:** Forms call existing schedule/override handlers; participant multiselect from team members list.

---

### AEG-071 — Account page

- **Status:** Ready
- **Depends on:** AEG-064, AEG-058, AEG-056
- **PRD:** REQ-AUTH-03, REQ-I18N-02, REQ-INT-04 (eXpress link surfacing)
- **Acceptance:**
  - [ ] Protected route `/account` per [`docs/features/account.md`](../docs/features/account.md)
  - [ ] **Profile:** avatar (or initials), editable display name, read-only email + role badge;
        `PATCH /auth/me` accepts `display_name` + existing `locale`
  - [ ] **Language:** en/ru control persists via `PATCH /auth/me` and syncs i18n/localStorage
  - [ ] **Connected sign-in:** list Google/Slack/eXpress linked state from `identities[]`; **Connect** buttons
        → `/auth/{provider}/login?redirect=/account`
  - [ ] **Paging identity:** read-only Slack id; eXpress link-code generator UI (`POST /users/me/express-link-code`)
  - [ ] Header: display name links to `/account` (or explicit Account entry)
  - [ ] Copy in `en` + `ru`; Vitest for profile/locale save; Storybook for AccountPage variants

**Plan:** `AccountPage` + section components; extend `AuthContext` user type; wire locale switcher to API
when session present.

---

### AEG-070 — Setup wizard team step and docs

- **Status:** Ready
- **Depends on:** AEG-067, AEG-069, AEG-066, AEG-071
- **PRD:** NFR-1, REQ-SHIFT-01
- **Acceptance:**
  - [ ] Setup wizard new step (or extend existing): create first team + add self or seeded users + minimal
        primary schedule (can link to full shifts page)
  - [ ] `docs/features/shifts-calendar.md` updated: UI no longer demo-only
  - [ ] `docs/03-data-model.md` — `avatar_url` on `users`
  - [ ] `README.md` — flow: seed dev users → create team → open shifts
  - [ ] `backlog/epics/EPIC-02-shifts.md` — note post-MVP UI wiring closed by AEG-068–069

**Plan:** Insert wizard step after auth; reuse team/schedule components from AEG-067/069.

---

## Dependency graph

```text
AEG-064 (identities + OIDC backfill)
  ├── AEG-065 (users list)
  ├── AEG-066 (dev seeds)
  └── AEG-071 (account page)
AEG-065 + AEG-058 → AEG-067 (teams UI)
AEG-067 + AEG-014/015 → AEG-068 (shifts read API)
AEG-068 + AEG-010/011 → AEG-069 (schedule/override write UI)
AEG-067 + AEG-069 + AEG-066 + AEG-071 → AEG-070 (wizard + docs)
```

## Suggested pick order

1. AEG-064 → AEG-071 (parallel) → AEG-066 → AEG-065  
2. AEG-067 → AEG-068 → AEG-069 → AEG-070
