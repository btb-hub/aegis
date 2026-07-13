# EPIC-13 — Integration admin configuration

**Phase:** 12 (post-MVP UX)  
**Exit:** An admin can create, edit, enable/disable, delete, and **test** Jira, Slack, and eXpress
from **`/integrations` alone** — no setup wizard — with required credentials. Incomplete config fails
with a clear message instead of “integration provider is not configured.”

## Decision — independent admin, not wizard-gated

**Admins configure each area on its own page.** Connecting Jira (or Slack, or eXpress) must not
require walking the multi-step `/setup` wizard (health → OIDC → integrations → test alert → …).

| Preferred | Deprecated as the config path |
|-----------|--------------------------------|
| `/integrations` for connector credentials + Test connection | Wizard step that forces prior steps to reach integrations |
| Other dedicated admin pages (Teams, Workspaces, …) as they mature | Wizard as the only place a setting can be saved |

The setup wizard may remain as an **optional first-run checklist** (links or short shortcuts into those
pages). It is **not** the source of truth for configuration and must not gate day-2 or day-1 connector
setup. Same direction for future workspace/escalation admin: configure on the dedicated surface, not by
replaying the wizard.

This supersedes the EPIC-03 implication that “admin UI button (minimal)” plus wizard fields were enough
for REQ-INT-05.

## Problem

EPIC-03 shipped the **connector spine**: `TicketProvider` / `ChatProvider`, Jira ticket create, Slack and
eXpress paging + ack, and `POST /integrations/{id}/test` ([EPIC-03](./EPIC-03-integrations.md)). That
work is real and marked `Done`.

What is **not** finished is **standalone admin configuration** on the Integrations page:

1. **Add integration saves empty config.** `/integrations` collects kind, name, and optional workspace
   `project_key` only. For a global Jira row it posts `config: {}`. Providers require credentials
   (`base_url` / `email` / `api_token` / `project_key` for Jira; `bot_token` / `signing_secret` for
   Slack; `bot_id` / `host` / `secret_key` for eXpress — see [`docs/integrations/`](../docs/integrations/)).
2. **Test connection fails opaquely.** `loader.RegisterFromRows` silently skips rows when
   `NewFromJSON` fails. `IntegrationService.Test` then returns
   `"integration provider is not configured"` — the same toaster users see for Jira today, and the
   same path Slack / eXpress hit with empty config.
3. **No edit / configure affordance.** There is no “Configure” action, no form for secrets after create,
   and no `PATCH /integrations/{id}`. Upsert is `POST /integrations` keyed by `(kind)` globally (or
   `(workspace_id, kind)`), which is enough for save — but the UI never sends credentials.
4. **Useful fields live only in the wizard today.** [`SetupWizardPage`](../../apps/web/src/pages/SetupWizardPage.tsx)
   collects provider config, but reaching that step means running earlier wizard steps. Admins should
   use [`IntegrationsPage`](../../apps/web/src/pages/IntegrationsPage.tsx) instead — and that page cannot
   finish the job yet.
5. **Secrets hygiene gap.** `IntegrationJSON` returns full `config` (including tokens) on list. Edit
   flows that round-trip the payload will leak secrets into the browser and logs unless redacted.

Workspace-scoped Jira `project_key` overrides remain in [AEG-085](./EPIC-11-support-levels-workspaces.md)
(EPIC-11). This epic does **not** re-open connector protocols; it makes independent admin config
complete and safe.

**User report:** Created a global Jira integration named “test”, clicked **Check connection**, got
`integration provider is not configured`. Same shape expected for Slack and eXpress from this page.
Product preference: fix `/integrations`, do not push users through the wizard.

## Solution (four tracks)

1. **API validation + actionable test errors** — reject incomplete config on save; surface parse/config
   errors on test instead of a generic “not configured.”
2. **Secrets-safe read/update** — redact secrets on list/get; `PATCH` by id that keeps existing secrets
   when omitted.
3. **Integrations page is the full configure surface** — create + edit forms with kind-specific fields
   (fields documented in `docs/integrations/*`); test works after a valid save. Wizard is not required
   and is not the UX target.
4. **Lifecycle controls** — enable/disable and delete from the same page; empty / incomplete states
   that tell the admin what to fill in.

**Out of scope:** New providers (Mattermost, Telegram, …), changing BotX/Slack/Jira protocols, OIDC
sign-in credentials (those stay env/OIDC), per-workspace credential sets (workspace rows still inherit
global credentials — AEG-085), rewriting or removing the wizard in this epic (optional follow-up:
wizard becomes thin links into admin pages).

**Sequencing note:** EPIC-12 (workspace admin, branch `feat/epic-12-workspace-admin`, stories
AEG-088–092) is separate and not yet on `main`. This epic uses **AEG-093+** and can run in parallel;
prefer merging credential UI before leaning on workspace Integrations tabs.

---

### AEG-093 — Validate integration config and clarify test failures

- **Status:** Ready
- **Depends on:** AEG-016, AEG-020 (Done)
- **PRD:** REQ-INT-05, REQ-INT-01
- **Acceptance:**
  - [ ] `IntegrationService.Upsert` validates required fields for **global** `jira`, `slack`, and
        `express` (same rules as `NewFromJSON` / docs)
  - [ ] Workspace-scoped Jira still requires `project_key` only (credentials may be inherited) —
        keep AEG-085 behaviour
  - [ ] `Test` returns a validation/error message that names what is missing or failed (e.g. incomplete
        config, HTTP failure), not bare `"integration provider is not configured"` when the row exists
        but cannot be registered
  - [ ] Prefer surfacing `NewFromJSON` / provider errors instead of silent skip + generic message
  - [ ] Unit tests for upsert validation and test-path messaging; recorded fixtures remain for live
        provider tests (no live calls in CI)
  - [ ] Locale-ready API `message` strings remain English source; UI may map codes later if needed

**Plan:** Tighten `apps/api/internal/service/integration.go` + loader error pass-through; tests in
`integration_test.go`. Branch: `feat/integrations-AEG-093-config-validation`.

---

### AEG-094 — Secret redaction and PATCH by id

- **Status:** Ready
- **Depends on:** AEG-093
- **PRD:** REQ-INT-05; security notes in [`docs/09-security.md`](../docs/09-security.md)
- **Acceptance:**
  - [ ] `GET /integrations` (and any get-by-id) redact secret keys in `config`: at least
        `api_token`, `bot_token`, `signing_secret`, `secret_key` (omit or replace with a sentinel such
        as `null` / `"***"` — pick one and document)
  - [ ] Non-secret config fields (`base_url`, `email`, `project_key`, `issue_type`, `bot_id`, `host`,
        etc.) remain visible to admins
  - [ ] `PATCH /integrations/{id}` updates name, enabled, and config; omitted secret fields **keep**
        stored values (do not wipe with empty string unless explicitly sent)
  - [ ] Admin-only; documented in [`docs/04-api-spec.md`](../docs/04-api-spec.md)
  - [ ] Handler + service unit tests for redact and patch-merge
  - [ ] Confirm list responses no longer dump live tokens into the SPA

**Plan:** `IntegrationJSON` redact helper; `UpdateIntegration` store method; handler register PATCH.
Branch: `feat/integrations-AEG-094-patch-redact`.

---

### AEG-095 — Integrations page: create and edit credentials (UI)

- **Status:** Ready
- **Depends on:** AEG-093, AEG-094
- **PRD:** REQ-INT-02, REQ-INT-03, REQ-INT-04, REQ-INT-05
- **Acceptance:**
  - [ ] **Add** modal (or drawer) on `/integrations` collects kind-specific required fields for Jira,
        Slack, and eXpress — field set matches [`docs/integrations/`](../docs/integrations/)
  - [ ] **Configure / Edit** on each row opens the same form prefilled with non-secret values; secrets
        show as blank with helper text “leave blank to keep current”
  - [ ] Global create no longer posts `config: {}`
  - [ ] After save, **Test connection** can succeed when credentials are valid (or returns the
        provider’s real failure message)
  - [ ] Completing Jira/Slack/eXpress setup does **not** require opening `/setup` or any wizard step
  - [ ] Copy in `en` and `ru` (`integrations.*`); follow writing rules
  - [ ] Vitest: form requires fields by kind; edit submit omits blank secrets
  - [ ] Matches design system (`Modal` / `Input` / `Select` / `Banner`)

**Plan:** Extend `IntegrationsPage.tsx` (+ tests); optional extract `IntegrationConfigFields.tsx` for
reuse if the wizard later links here. Branch: `feat/integrations-AEG-095-configure-ui`.

---

### AEG-096 — Enable, disable, delete, and incomplete-state UX

- **Status:** Ready
- **Depends on:** AEG-095
- **PRD:** REQ-INT-05
- **Acceptance:**
  - [ ] Admin can **enable / disable** an integration from the list (PATCH `enabled`)
  - [ ] Admin can **delete** with confirm; calls existing `DELETE /integrations/{id}`
  - [ ] Rows that cannot be registered (legacy empty config) show a clear status / banner, not only a
        failed toast on test — e.g. “Add credentials to finish setup”
  - [ ] Empty state tells admin to add Jira, Slack, or eXpress with credentials **on this page**
  - [ ] en + ru strings; Vitest for disable/delete affordances (admin-only)

**Plan:** Actions column + confirm pattern reused from Teams/Users. Branch:
`feat/integrations-AEG-096-lifecycle-ui`.

---

## Suggested order

```text
AEG-093 (validate + test errors)
  └── AEG-094 (redact + PATCH)
        └── AEG-095 (configure UI)
              └── AEG-096 (enable/disable/delete + incomplete state)
```

## Definition of Done (epic)

- [ ] From `/integrations` alone, admin can fully configure Jira, Slack, and eXpress and run Test
      connection — **wizard not required**
- [ ] Docs state independent admin pages as the preferred configuration model (this epic + setup +
      integrations README + PRD clarification)
- [ ] Secrets not returned in list responses
- [ ] `make lint type test` green on implementing PRs
- [ ] API + integration docs updated where behaviour changes
