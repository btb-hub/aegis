# EPIC-15 — Production ops (BotX CTS, on-call announce, OIDC)

**Phase:** 14 (production follow-up)  
**Exit:** BotX CTS can reach `/status` and `/command` so `/link` works. Admins store a team
channel and Aegis publishes current on-call (replacing an external CronJob). Login and Account
only offer configured OIDC providers; SSO is separate from eXpress paging bind.

DevOps reports after production use. These stories interrupt EPIC-14 (AEG-100+); they do not wait
on alert-row work. IDs continue from **AEG-104**. Pick one story at a time unless a human
parallelizes.

## Problem

1. **BotX URL in admin is a base.** CTS appends `/status` and `/command`. Aegis only listens on
   `POST /api/v1/callbacks/express/bot`, so `/link` 404s.
2. **On-call announce lives in an external CronJob** (Aegis API → BotX). Teams have no channel IDs.
   Worker already materialises on-call; it does not post to a group with an @mention by huid.
3. **Connect Slack / eXpress goes to OIDC.** Prod has Google only. API returns 400
   `unknown or unconfigured provider`. Account uses a React Router `Link` into `/auth/...`, which
   the SPA catch-all sends to `/shifts` with no error. SSO and “bind eXpress for paging” are mixed
   in copy.

## Solution (three tracks)

1. **BotX CTS paths** — `GET …/status` and `POST …/command` beside `/bot`.
2. **On-call channel publish** — team chat IDs, `publish_oncall` job, optional Publish now.
3. **Configured OIDC** — `GET /auth/providers`; hide unconfigured Connect; SSO vs paging copy.

**Out of scope:** SmartApp; `POST /notification/callback`; second Gin listener; unlinking OIDC
identities; changing incident DM paging; rewriting AEG-097–103 except the BotX URL string on
AEG-103 (base, not `/bot`).

---

### AEG-104 — BotX `/status` and `/command`

- **Status:** In Review
- **Depends on:** AEG-019 (Done)
- **PRD:** REQ-INT-04, REQ-INT-07 (proposed: CTS calls `{base}/status` and `{base}/command`)
- **Acceptance:**
  - [x] Given BotX CTS, when it `GET {PUBLIC_URL}/api/v1/callbacks/express/status` with a valid BotX
        JWT, then **200**
        `{ "status": "ok", "result": { "enabled": true, "status_message": "…", "commands": [ { "name", "body": "/link", "description" } ] } }`
  - [x] Given no express integration/secret, when status is called, then **503**
        `{ "reason": "bot_disabled", "error_data": { "status_message": "…" }, "errors": [] }`
  - [x] Given an invalid JWT on status or command, when called, then **401**
  - [x] Given `POST …/command` with JWT and `/link <code>`, when processed, then **202**
        `{ "result": "accepted" }` within 5s and the huid is bound (sync process, then 202)
  - [x] Given a known ack command, when processed, then **202** `{ "result": "accepted" }` and the
        incident is acknowledged
  - [x] Given `system:*` or any unknown command body, when posted to `/command`, then **202**
        (do not 400)
  - [x] Given `POST …/bot`, when called, then it is a backward-compat alias of `/command` (same 202)
  - [x] Docs: webhook **base** `{PUBLIC_URL}/api/v1/callbacks/express` (CTS appends `/status` and
        `/command`). [`docs/04-api-spec.md`](../../docs/04-api-spec.md),
        [`docs/integrations/express.md`](../../docs/integrations/express.md),
        [`docs/09-security.md`](../../docs/09-security.md). AEG-103 copyable URL is this **base**
  - [x] No `/notification/callback`. No second HTTP listener. Tests use fixtures; no live BotX

**Plan:** Shared `handleCommand` in `express_callback.go`. Branch:
`feat/integrations-AEG-104-botx-status-command`.

---

### AEG-106 — Configured OIDC only; SSO vs paging

- **Status:** Ready
- **Depends on:** AEG-057, AEG-071 (Done)
- **PRD:** REQ-AUTH-01 (proposed: UI lists only configured providers)
- **Acceptance:**
  - [ ] Given `GET /auth/providers` (no session), when called, then `{ "providers": ["google", …] }`
        contains only names that pass `cfg.Provider` (non-empty credentials; eXpress also needs issuer)
  - [ ] Given `/login`, when only Google is configured, then Slack and eXpress sign-in buttons are
        not rendered. Buttons that are shown remain full-page `<a href="/auth/{p}/login">`
  - [ ] Given `/account` **Connected sign-in (SSO)**, when a provider is not configured, then Connect
        is not shown. Connect for a configured, unlinked provider is `<a href>` (never React Router
        `Link`) so the SPA catch-all cannot send the user to `/shifts`
  - [ ] Given Account **Paging identity**, when the user needs eXpress for paging, then they still
        use **Generate link code** (`/link`). Copy does not say “sign-in and paging”. Headings
        separate SSO from bind-eXpress-for-paging
  - [ ] en + ru; Vitest for login/account provider lists; no silent redirect

**Plan:** `Config.ConfiguredProviders()`; `AuthHandler.providers`. Branch:
`feat/auth-AEG-106-configured-oidc-providers`.

---

### AEG-105 — Publish current on-call to team channels

- **Status:** In Review
- **Depends on:** AEG-014, AEG-019 (Done)
- **PRD:** proposed REQ-SHIFT-08
- **Acceptance:**
  - [x] Given a team, when an admin PATCHes `express_chat_id` and/or `slack_channel_id`, then the
        values persist (nullable). Teams UI can edit them
  - [x] Given job `publish_oncall` with `{ "team_id" }` or `{}`, when run, then current on-call is
        posted to each configured channel. eXpress: group notification + `@mention` by huid
        (skip users with no huid). Slack: channel `chat.postMessage` with `<@U…>` when `slack_user_id`
        is set. Skip teams with neither channel
  - [x] Given a **daily** worker tick (same style as `materialise_oncall`), when it fires, then
        `publish_oncall` is enqueued for all configured teams
  - [x] Given the current on-call user-id set differs from last published (stored on the team), when
        a minute ticker notices, then enqueue `publish_oncall` (rotation handoff)
  - [x] Given an admin clicks **Publish now** (or `POST /teams/{id}/on-call/publish`), when accepted,
        then the job is enqueued and the toast is **Published**
  - [x] Soft-fail per provider (do not fail the other). Do not reuse `notify_handoff` (L2→L3)
  - [x] Recorded fixtures; no live BotX/Slack. en + ru for UI and chat templates. API spec + data
        model updated. Migration up **and** down named after this story

**Plan:** Migration on `teams`; `AnnounceOnCall` beside DM `SendPage`; worker job + ticker + button.
Branch: `feat/shifts-AEG-105-publish-oncall-channel`. Split if the PR approaches ~400 lines.

---

## Dependency graph

```
AEG-019 (Done)
  └── AEG-104 (BotX status/command)     // P0 /link

AEG-057 + AEG-071 (Done)
  └── AEG-106 (OIDC providers UX)

AEG-014 + AEG-019 (Done)
  └── AEG-105 (publish on-call)
```

**Suggested order:** AEG-104 → AEG-106 → AEG-105.

---

## Out of scope

- SmartApp; `/notification/callback`
- Second HTTP listener in `cmd/api`
- Unlink OIDC identities
- Incident DM paging changes
- AEG-097–103 except documenting the BotX **base** URL on AEG-103

## Definition of done (epic)

- [ ] CTS `GET /status` and `POST /command` work; `/link` binds huid
- [x] On-call announce is an in-Aegis job (daily + handoff + publish now)
- [ ] Unconfigured OIDC is not offered; no silent `/shifts` redirect
- [ ] `make lint type test` green on implementing PRs
