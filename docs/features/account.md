# Feature: Account page

**Backlog:** Phase 8 — [AEG-071](../../backlog/epics/EPIC-09-teams-users-shifts.md)  
**Related:** [web-auth.md](./web-auth.md), [EPIC-09](../../backlog/epics/EPIC-09-teams-users-shifts.md) (AEG-064 identity backfill)

## Problem

There is no `/account` route. The shell shows display name + **Sign out** only. Locale is toggled from
the header via `localStorage` and is **not** synced to `PATCH /auth/me` / `users.locale`. Express paging
identity (`express_user_huid`) is bindable via API but has no UI. Users cannot see which SSO providers
are linked or fix profile fields without admin intervention.

## Goals

1. One place for **profile**, **language**, and **notification identities** (Slack / eXpress).
2. Surface **connected SSO providers** after [AEG-064](../../backlog/epics/EPIC-09-teams-users-shifts.md)
   multi-provider linking ships.
3. **Manual edits only** for overwriting profile data — automatic OIDC login never clobber existing fields.

## Non-goals (Phase 8)

- Change global RBAC role (admin-only, future admin user screen).
- Unlink a provider (manual admin / support flow later).
- Password or local credentials.
- Notification channel preferences beyond “is Slack/eXpress id present”.

---

## Information architecture

```text
/account
├── Profile          — avatar, display name, email (read-only)
├── Language         — en | ru → PATCH /auth/me
├── Connected sign-in — Google / Slack / eXpress status + “Add …” links
└── Paging identity  — Slack user id (read-only), eXpress link flow
```

Route: `/account`, protected (session required). All roles.

**Entry points:**

- Header: display name becomes a link to `/account` (or add explicit **Account** next to Sign out).
- Setup wizard auth step: link “Review your account” after sign-in.

---

## Layout (design system)

Follow [`12-design-system.md`](../12-design-system.md): max-width content column inside shell main area,
H1 **Account**, stacked **Card** sections with H2 headings.

### Section 1 — Profile

| Field | Source | Editable |
|-------|--------|----------|
| Avatar | `avatar_url` or initials fallback | No (future: “Refresh from Google” optional) |
| Display name | `display_name` | Yes → `PATCH /auth/me` |
| Email | `email` | Read-only |
| Role | `role` | Read-only badge (admin / member / viewer) |

**Save** primary button when display name dirty. Toast: “Profile updated”.

Empty avatar: circle with initials from `display_name` (same pattern as on-call calendar).

### Section 2 — Language

Radio or segmented control: **English** | **Русский**.

On change:

1. `PATCH /auth/me` `{ "locale": "en" | "ru" }`
2. Update i18n + `localStorage` (same as today’s switcher)
3. Toast: “Language updated”

Move language switcher from global header to this section **or** keep header switcher but sync both
ways when session exists.

### Section 3 — Connected sign-in

List one row per provider (Google, Slack, eXpress):

| State | UI |
|-------|-----|
| Linked | Provider icon + “Connected · signed in {relative time}” or “Connected” |
| Not linked | Muted “Not connected” + secondary **Connect {Provider}** → `/auth/{provider}/login?redirect=/account` |

Data from expanded `GET /auth/me`:

```json
{
  "identities": [
    { "provider": "google", "linked_at": "2026-06-01T10:00:00Z" },
    { "provider": "slack", "linked_at": "2026-07-03T14:00:00Z" }
  ]
}
```

Copy explains: connecting another provider adds paging/sign-in options; profile fields you set here are
kept (OIDC only fills empty fields).

### Section 4 — Paging identity

Used by incident pages (Slack DM, eXpress bubble). Distinct from “sign-in” — you may sign in with Google
but page via Slack once `slack_user_id` is set.

| Channel | Field | UI |
|---------|-------|-----|
| Slack | `slack_user_id` | Read-only; “Set when you sign in with Slack” or show id |
| eXpress | `express_user_huid` | If missing: **Generate link code** → calls `POST /users/me/express-link-code`, shows `/link <code>` instruction + copy button. If set: read-only UUID |

Reuse wording from [`integrations/express.md`](../integrations/express.md).

---

## API additions (AEG-071)

| Method | Path | Body | Notes |
|--------|------|------|-------|
| GET | `/auth/me` | — | Extend with `avatar_url`, `slack_user_id`, `express_user_huid`, `identities[]` |
| PATCH | `/auth/me` | `{ "locale"?, "display_name"? }` | Validate display_name non-empty, max length |

Admin overwrite (out of scope for account page, AEG-064):

- `PATCH /api/v1/users/{id}` — admin force-set profile / role (future).

---

## OIDC backfill interaction (AEG-064)

When user clicks **Connect Slack** on account page:

1. OAuth callback resolves identity.
2. If `(provider, sub)` new but **email matches** existing user → attach identity to **same** user row.
3. Backfill **only empty** fields: `slack_user_id`, `avatar_url`, `display_name`.
4. Session continues as that user; redirect to `/account` with toast “Slack connected”.

Never auto-overwrite user-edited `display_name` or chosen `avatar_url`.

---

## Empty and error states

- **Not signed in:** redirect to `/login?redirect=/account`.
- **Express code expired:** inline error under link code; button to regenerate.
- **PATCH failed:** inline error under field; no silent fallback.

---

## i18n

New keys under `account.*` in `en` and `ru` locale files. Button labels match verbs used elsewhere
(`Save`, `Connect Google`, `Generate link code`).

---

## Tests & Storybook

- Vitest: profile save, locale PATCH, express link code display.
- Storybook: `AccountPage` with linked / unlinked provider variants (en/ru).

---

## References

- Security: [`09-security.md`](../09-security.md)
- API: [`04-api-spec.md`](../04-api-spec.md)
- eXpress link: [`integrations/express.md`](../integrations/express.md)
