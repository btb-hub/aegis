# EPIC-14 — Alert operations, routing discoverability, and public chat ingress

**Phase:** 13 (post-MVP UX)  
**Exit:** An admin finds routing from **Alerts** (not only by knowing to open a workspace). An operator
can **assign**, **create an incident**, or **resolve** from an alert row. BotX/Slack/webhook traffic
reaches Aegis without interactive Google/IAP login — documented as a path skip or a second public
host, with copyable callback URLs on Integrations.

## Decision — alerts stay signals; work stays on incidents

Routing rule CRUD stays on **Workspaces → workspace detail**. This epic does **not** add a Routing
nav or a second rule editor. Alerts remain raw signals; Assign and Create incident always create or
update an **incident**. Resolve on a linked alert resolves that incident; resolve on an unlinked
firing alert marks the **alert** `resolved` (silence, no incident).

Machine callbacks already skip the Gin session. The remaining gap is the **outer** proxy: if Google
IAP (or similar) wraps `PUBLIC_URL`, BotX never reaches `/api/v1/callbacks/express/bot`. Split public
ingress there — not with a second Gin listener.

## Problem

EPIC-11/12 shipped workspace-scoped routing UI (`Add routing rule` on `/workspaces/:id`, AEG-086).
EPIC-04 shipped auto incident create when a rule matches. EPIC-03 shipped BotX JWT callbacks. EPIC-05
shipped a read-only Alerts workspace (filter, search, export). EPIC-13 is credential admin on
`/integrations` — it does **not** cover ingress.

Colleague report after using the product:

1. **No “create routing rule” button.** Test alerts arrive, but nothing routes until a rule exists.
   “It seems to be inside the workspace, but I could not find a configure button. Maybe I missed it.”
2. **Incoming alerts are inert.** Cannot assign, create an incident, or resolve from the list.
3. **eXpress integration is awkward.** BotX needs HTTP that is not behind Google auth, or a reverse
   proxy (they already have a proxy and will use it).

Today that maps to:

1. **Alerts has no CTA.** [`AlertsPage`](../../apps/web/src/pages/AlertsPage.tsx) has no header
   actions. The user guide says “check Workspaces → routing rules” but `/alerts` does not point there.
2. **No alert-row actions.** Alert list JSON has no `incident_id`. There is no `POST /incidents`, no
   assign endpoint, and no `POST /alerts/{id}/resolve`. `process_alert` errors with
   `no routing rule matched alert labels` and leaves the alert `firing` and unlinked.
3. **One origin in production.** All-in-one nginx and k8s Ingress put UI and `/api/` on `PUBLIC_URL`.
   IAP on that host blocks BotX even though Gin does not require a session on callbacks.

## Solution (three tracks)

1. **Routing discoverability** — admin **Configure routing** on `/alerts` → `/workspaces`. No new CRUD.
2. **Alert operations** — list exposes `incident_id`; manual create, assign, and unlinked resolve APIs;
   row actions for admin/member.
3. **Public ingress** — document IAP/path skip (or second host) for callbacks, webhook, and health;
   show copyable BotX and Slack URLs on Integrations.

**Out of scope:** New routing CRUD on Alerts; EPIC-13 credential forms; second HTTP listener in the
API process; BotX/Slack protocol changes; eXpress SmartApp or OIDC rewrite; auto-resolving alerts when
an incident resolves; making IAP a required deploy component; viewer mutating alerts/incidents.

**Sequencing note:** Workspace routing UI (`WorkspaceDetailPage`, **Add routing rule**) is on `main`
even if EPIC-11 still lists AEG-086 as Ready. EPIC-13 (AEG-093–096) can land in parallel; AEG-103’s
URL strip does not wait on credential forms. This epic uses **AEG-097+**. Pick one story at a time
unless a human parallelizes; later IDs stay `Ready` so the loop can take them in graph order.

---

### AEG-097 — Configure routing CTA on Alerts

- **Status:** Ready
- **Depends on:** Workspace routing UI on `main` (`WorkspaceDetailPage`)
- **PRD:** REQ-SLV-10, REQ-INC-04
- **Acceptance:**
  - [ ] Given an **admin** on `/alerts`, when the page loads, then the header shows **Configure routing**
        as a **secondary/ghost** control (not the page primary — Export stays in the filter bar)
  - [ ] Given that admin activates **Configure routing**, when they have exactly one workspace, then
        navigate to `/workspaces/{id}` (routing section / **Add routing rule**); otherwise `/workspaces`
  - [ ] Given a **member** or **viewer** on `/alerts`, when the page loads, then **Configure routing**
        is not shown
  - [ ] Given the Alerts empty state, when an admin sees it, then copy still tells them to configure
        routing via Workspaces (header CTA is enough; do not add a second rule editor)
  - [ ] Locale keys in `en` and `ru` (e.g. `alerts.configure_routing`); writing rules; no emoji
  - [ ] Vitest: admin sees the button and navigates (one vs many workspaces); member does not
  - [ ] When this story ships, replace the user-guide “later story” sentence so admins open routing
        from Alerts as well as Workspaces

**Plan:** `PageHeader` `actions` on `AlertsPage.tsx`. Branch:
`feat/alerts-AEG-097-configure-routing-cta`.

---

### AEG-098 — `incident_id` on alert list

- **Status:** Ready
- **Depends on:** AEG-033 (alert list)
- **PRD:** REQ-ALERT-01, REQ-ALERT-08
- **Acceptance:**
  - [ ] Given any JSON that serializes a list/group **sample** alert (`GET /alerts`, grouped buckets’
        sample), when the row is returned, then it includes `incident_id` (`uuid` or `null`)
  - [ ] Given `incident_id`, when computed, then it is the **most recently linked open or acknowledged**
        incident; if several are open/acked, use the latest link; if the alert is linked **only** to
        resolved incidents, then `incident_id` is `null` (treat as unlinked for create/resolve)
  - [ ] Given an alert with no `incident_alerts` rows, when listed, then `incident_id` is `null`
  - [ ] CSV export may omit the column in this story (list/UI is the contract)
  - [ ] Web `AlertItem` type and tests updated; [`docs/04-api-spec.md`](../docs/04-api-spec.md) documents
        the field and precedence
  - [ ] Unit tests for the join / precedence rule (including resolved-only links)

**Plan:** Extend list query in `apps/api` (sqlc + `AlertService`); map in handler JSON. Branch:
`feat/alerts-AEG-098-incident-id`.

---

### AEG-099 — Manual `POST /incidents` from an alert

- **Status:** Ready
- **Depends on:** AEG-023 (worker create path), AEG-098
- **PRD:** REQ-INC-03, REQ-INC-05, REQ-INC-06, REQ-INC-07, REQ-INC-08, REQ-ALERT-08
- **Acceptance:**
  - [ ] Given a **firing** alert with `incident_id` null, when `POST /incidents` is called with
        `{ "alert_id", "team_id" }` and optional `assignee_id`, then an incident is created, linked,
        and enqueues the **same jobs** as `process_alert` (timeline, notify/Jira/chat, escalation).
        The API request path does **not** call Jira/Slack/eXpress directly
  - [ ] Given `assignee_id` omitted, when created, then assignee is the team's current on-call
        (nullable if none). Given `assignee_id` set, then that user must exist and be a **member of
        `team_id`** or `400`; page that user (not a create-then-reassign that pages on-call first)
  - [ ] Given unknown `alert_id`, when posted, then `404`
  - [ ] Given this alert already has an open/acked `incident_id`, when posted, then `409`
  - [ ] Given another **open** incident with the same fingerprint (REQ-INC-03), when posted, then
        **link** this alert to that incident (no second incident) and return that incident; do not
        ignore team_id silently — if teams differ, `409` with `details.incident_id`
  - [ ] Given two concurrent creates for the same alert, when both commit, then one succeeds and the
        other `409` (row lock or unique link)
  - [ ] Given a non-firing alert or missing `team_id`, when posted, then `400`
  - [ ] Given a **viewer** session, when posted, then `403`; admin and member allowed
  - [ ] After success or unlinked resolve (AEG-101), pending `process_alert` jobs for that alert must
        no-op (already linked / not firing) rather than create a duplicate
  - [ ] Unit tests for create, link-on-fingerprint, 409, 400, 404; no live connectors in CI
  - [ ] API spec updated

**Plan:** Shared create/link helper used by worker and API; enqueue notify jobs. Branch:
`feat/incidents-AEG-099-create-from-alert`.

---

### AEG-100 — `POST /incidents/{id}/assign`

- **Status:** Ready
- **Depends on:** AEG-024, AEG-099
- **PRD:** REQ-INC-06, REQ-ALERT-08
- **Acceptance:**
  - [ ] Given an open or acknowledged incident, when `POST /incidents/{id}/assign` with
        `{ "user_id" }` is called, then `assignee_id` updates and a timeline event is recorded.
        **Same team only** — do not change `team_id` here
  - [ ] Given `team_id` in the body that differs from the incident’s owning team, when assigned,
        then `400` with a message to use **handoff** (`POST /incidents/{id}/handoff`) instead
  - [ ] Given `user_id` is not a member of the owning team (or is unknown), when assigned, then `400`
  - [ ] Given a **resolved** incident, when assigned, then `409`
  - [ ] Admin and member allowed; viewer `403`
  - [ ] Unit tests; API spec updated

**Plan:** `IncidentService.Assign` — person on the current team only. Branch:
`feat/incidents-AEG-100-assign`.

---

### AEG-101 — `POST /alerts/{id}/resolve` (unlinked)

- **Status:** Ready
- **Depends on:** AEG-098
- **PRD:** REQ-ALERT-08
- **Acceptance:**
  - [ ] Given a **firing** alert with `incident_id` null, when `POST /alerts/{id}/resolve` is called,
        then the alert status becomes `resolved` and **no** incident is created
  - [ ] Given the alert is already linked to an incident, when called, then `409` (client must use
        `POST /incidents/{id}/resolve`)
  - [ ] Given the alert is already `resolved`, when called, then `409`
  - [ ] Admin and member allowed; viewer `403`
  - [ ] Resolving an **incident** still does **not** auto-resolve linked alerts (unchanged)
  - [ ] Unit tests; API spec updated

**Plan:** `AlertService.ResolveUnlinked` in `apps/api/internal/service/alert.go`. Branch:
`feat/alerts-AEG-101-resolve-unlinked`.

---

### AEG-102 — Alert row actions UI

- **Status:** Ready
- **Depends on:** AEG-097, AEG-098, AEG-099, AEG-100, AEG-101
- **PRD:** REQ-ALERT-08, REQ-DS-01, REQ-DS-03, REQ-I18N-01, REQ-I18N-05
- **Acceptance:**
  - [ ] Given an admin or member on `/alerts` with **group-by off**, when the table renders, then each
        row has an overflow (or equivalent) with **Assign**, **Create incident**, and **Resolve** —
        not three primary buttons. Dialogs have one primary
  - [ ] Given **group-by is on**, when the grouped table renders, then row actions are hidden
        (ungroup to act on a single alert)
  - [ ] Given `incident_id` is set, when the row renders, then **Create incident** is hidden and
        **Open incident** goes to `/incidents/{id}`
  - [ ] Given an unlinked **resolved** alert, when the row renders, then Assign, Create incident, and
        Resolve are hidden
  - [ ] Given `incident_id` is null and status is firing, when the user chooses **Create incident**,
        then a team picker (required) submits `POST /incidents` without `assignee_id`
  - [ ] Given **Assign** on an unlinked firing alert, when submitted, then a **single**
        `POST /incidents` includes `team_id` and `assignee_id` (no create-then-reassign). Given
        linked, then `POST /incidents/{id}/assign` with `user_id` only; team change is **handoff**
        on the incident, not this menu
  - [ ] Given **Resolve** on a linked alert, when confirmed, then `POST /incidents/{id}/resolve`.
        The alert row may still show status **firing** (incident resolve does not flip alert status);
        copy/toast: **Resolved** refers to the incident. Given unlinked firing, then
        `POST /alerts/{id}/resolve`
  - [ ] Given a **viewer**, when the table renders, then row actions are absent
  - [ ] Toasts use the same verbs as the buttons (`Assigned`, `Incident created`, `Resolved`)
  - [ ] en + ru; Vitest for linked vs unlinked vs grouped vs resolved visibility
  - [ ] If a new shared overflow is required, it ships with a Storybook story (REQ-DS-04); otherwise
        reuse existing `Button` / `Modal`

**Plan:** `AlertTable.tsx` + `AlertsPage.tsx`; `alertsApi.ts` / `incidentsApi.ts`. Branch:
`feat/alerts-AEG-102-row-actions`.

---

### AEG-103 — Public ingress docs and callback URLs

- **Status:** Ready
- **Depends on:** AEG-019, AEG-018, AEG-020
- **PRD:** REQ-INT-07, REQ-AUTH-05
- **Acceptance:**
  - [ ] Given [`docs/07-setup-deployment.md`](../docs/07-setup-deployment.md), when an admin deploys
        behind Google IAP (or any interactive identity-aware proxy) on the **UI origin**, then the doc
        requires a **path skip** or **second public host** for `/api/v1/callbacks/`,
        `/api/v1/alerts/webhook`, `/healthz`, `/readyz`, and `/metrics`. Kube probes should keep using
        the in-cluster Service, not the public IAP host
  - [ ] Given a second public host, when `PUBLIC_URL` is set, then it is the **machine-reachable**
        origin (BotX/Slack/webhooks), not the IAP-wrapped UI origin. Copyable URLs use that value
        (API may expose it; do not use `window.location.origin` if that is the IAP host)
  - [ ] Given [`docs/integrations/express.md`](../docs/integrations/express.md) (and Slack equivalent),
        when the BotX webhook URL is documented, then it is `{PUBLIC_URL}/api/v1/callbacks/express/bot`
        (Slack: `{PUBLIC_URL}/api/v1/callbacks/slack/interactive`) and must not sit behind IAP.
        Gin already verifies BotX JWT / Slack signing secret / webhook secret
  - [ ] Given [`docs/09-security.md`](../docs/09-security.md), when public vs session routes are
        listed, then callbacks/webhook/health/metrics are called out as machine ingress
  - [ ] Given `/integrations`, when a Slack or eXpress **row exists**, then the admin sees those
        copyable callback URLs even if credentials are still incomplete (EPIC-13)
  - [ ] Example nginx (or Ingress) snippet lives under `deploy/` or the setup doc — same location
        style as `deploy/nginx.all-in-one.conf`
  - [ ] Explicit non-goal in the PR: **no** second Gin HTTP listener; **no** weakening of callback
        verification
  - [ ] en + ru for any new Integrations copy; no live BotX in CI

**Plan:** Docs first, then a read-only URL field on `IntegrationsPage.tsx`. Branch:
`feat/integrations-AEG-103-public-ingress`.

---

## Dependency graph

```
Workspace routing UI on main
  └── AEG-097 (Configure routing CTA)

AEG-033 (alert list — Done)
  └── AEG-098 (incident_id on list)
        ├── AEG-099 (POST /incidents)
        │     └── AEG-100 (assign)
        └── AEG-101 (unlinked alert resolve)
              └── AEG-097 + AEG-098–101 → AEG-102 (row actions UI)

AEG-018/019 (callbacks — Done)
  └── AEG-103 (ingress docs + copyable URLs)   // parallel from the start
```

**Suggested order:** AEG-097 and AEG-098 and AEG-103 in parallel; then AEG-099 → AEG-100 → AEG-101;
AEG-102 last.

---

## Out of scope

- New Routing nav or duplicating rule CRUD on Alerts (AEG-086 stays the editor)
- Reopening EPIC-13 credential validation/redaction/forms
- Second HTTP listener in `cmd/api`
- Changing BotX JWT, Slack signature, or webhook secret schemes
- eXpress SmartApp embedded view; OIDC login rewrite
- Auto-resolve linked alerts when an incident resolves
- Requiring Google IAP in production
- Viewer assign / create / resolve

---

## Definition of done (epic)

- [ ] Admin can open routing from Alerts without already knowing the Workspaces URL
- [ ] Alert rows support Assign, Create incident, and Resolve per the Decisions table below
- [ ] Deploy docs forbid IAP on callback/webhook/health paths; Integrations shows copyable URLs
- [ ] PRD REQ-ALERT-08 and REQ-INT-07 are implemented by the stories above
- [ ] `make lint type test` green on implementing PRs; API spec + user guide updated where behaviour
      changes
- [ ] No credentials in docs examples beyond placeholders

## Decisions (locked)

| # | Question | Decision |
|---|----------|----------|
| 1 | Routing UX | Discoverability only: **Configure routing** → `/workspaces` |
| 2 | Create / Assign | Always an **incident**; team required on create; default assignee = on-call; assign is same-team only (team change = handoff) |
| 3 | Resolve | Linked → incident resolve; unlinked firing → alert `resolved`, no incident |
| 4 | eXpress / IAP | Path skip or second host; no second Gin listener; keep callback verification |
| 5 | Roles | Admin + member mutate; viewer read-only |
