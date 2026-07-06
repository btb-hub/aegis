# User guide by role

How to use Aegis day to day. This guide covers **access roles** (who can change configuration) and
**support tiers** (how on-call engineers work incidents). Everyone shares the same sign-in flow and
navigation; what you can do depends on your role and team membership.

For installation and first-time setup, see [`07-setup-deployment.md`](./07-setup-deployment.md).
For security details, see [`09-security.md`](./09-security.md).

---

## Two kinds of “role”

Aegis uses two separate concepts. Do not confuse them.

| Concept | Where it lives | Examples | What it controls |
|---------|----------------|----------|------------------|
| **Access role** | Your user account (`admin`, `member`, `viewer`) | IT admin, on-call engineer, manager | Whether you can change teams, integrations, and incidents |
| **Support tier** | A team’s `support_tier` (`noc`, `l1`, `l2`, `l3`) | Platform L2, Data L3, Platform NOC | Which escalation buttons appear and which handoff targets are valid |

You also belong to **teams** as a member or lead. Team membership determines who appears in rotation
pickers and who gets paged when an incident is assigned to that team.

---

## Getting started (all users)

### Sign in

1. Open the web app (for example `http://localhost:3000` in local dev).
2. Go to **Login** and choose **Google**, **Slack**, or **eXpress**.
3. After the identity provider redirects back, you land in the app with an active session.

On localhost only, **Dev sign in** is available when `DEV_AUTH_ENABLED=true`. Use it for testing
without registering OIDC apps.

### Navigation

The sidebar links to the main areas:

| Page | Route | Purpose |
|------|-------|---------|
| Shifts | `/shifts` | Who is on call now; rotation calendar |
| Teams | `/teams` | Team list, members, escalation paths |
| Incidents | `/incidents` | Open work items — ack, resolve, hand off |
| Alerts | `/alerts` | Raw monitoring signals; search and export |
| Dashboard | `/dashboard` | MTTA, MTTR, noise, load, handoffs |
| Integrations | `/integrations` | Jira, Slack, eXpress connectors |
| Setup | `/setup` | Guided first-time configuration |

### Account

Open **Account** (`/account`) from the header to:

- Edit display name and language (`en` / `ru`)
- See connected sign-in providers
- Link eXpress for paging (`/link` flow)
- View your access role (read-only)

---

## Access roles

### Admin

**Who:** IT administrators, platform owners, people who configure Aegis.

**Can do:**

| Area | Actions |
|------|---------|
| **Setup wizard** | Run all steps: health check, sign-in, workspace + L2/L3 teams, integrations, test alert |
| **Workspaces** | Create and edit projects; open workspace detail for routing rules |
| **Teams** | Create, edit, delete teams; set support tier (L1 / L2 / L3 / NOC); add/remove members |
| **Escalation paths** | On team detail: add or remove allowed handoff targets |
| **Routing rules** | On workspace detail: label matchers → target team, priority order |
| **Shifts** | Create weekly schedules and temporary overrides on `/teams/{id}/shifts` |
| **Integrations** | Add, edit, delete, and **Test connection** for Jira, Slack, eXpress; set global or per-workspace Jira project keys |
| **Incidents** | Same as member — acknowledge, resolve, hand off, bounce |
| **Alerts** | Search, filter, group, save views, export CSV |
| **Dashboard** | View all analytics widgets |

**Typical first-day workflow:**

1. Sign in as admin → open **Setup** (`/setup`).
2. Confirm API health and complete OIDC sign-in.
3. Create a **workspace** (for example Platform) with L2 and L3 teams and an L2→L3 escalation path.
4. Add **integrations** — global Jira credentials plus optional per-workspace `project_key` overrides.
5. Add a **routing rule** (for example `team=platform` → Platform L2).
6. Create a **schedule** on the L2 team shifts page and add team members.
7. Send a **test alert** from the wizard and confirm an incident appears under **Incidents**.

**Admin checklist after go-live:**

- Every workspace has at least one routing rule for incoming alerts.
- L2 teams have an L3 escalation path configured.
- On-call schedules cover all paging teams.
- Integrations pass **Test connection**.

---

### Member

**Who:** On-call engineers, L1/L2/L3 responders, NOC operators.

**Can do:**

| Area | Actions |
|------|---------|
| **Incidents** | **Acknowledge**, **Resolve**, **Hand off** / **Escalate**, **Bounce** (when allowed) |
| **Shifts** | View on-call calendar and “on call now” banner |
| **Teams** | View teams, members, and escalation paths (read-only) |
| **Alerts** | Search, filter, group, save personal saved views, export CSV |
| **Dashboard** | View analytics |
| **Integrations** | View connector list (no add/edit/test) |
| **Account** | Edit profile and locale |

**Cannot do:** create teams, change schedules, edit routing rules, or manage integrations.

**Typical incident workflow:**

1. Receive a page in **Slack** or **eXpress** (or see a new incident in **Incidents**).
2. Click **Acknowledge** in chat or on the incident detail page.
3. Investigate using linked **alerts**, the **Jira** ticket, and the **timeline**.
4. If the issue needs a higher tier, click the tier-aware handoff button (for example **Hand off to L3**)
   and pick a configured target team.
5. When fixed, click **Resolve**.

**Acknowledge from chat:** Slack and eXpress page messages include an **Acknowledge** button. Use it
instead of opening the web app when you are on mobile or in a war room.

---

### Viewer

**Who:** IT managers, auditors, people who need visibility without changing state.

**Can do:**

| Area | Actions |
|------|---------|
| **Incidents** | View list, detail, timeline, linked alerts (read-only) |
| **Alerts** | Search, filter, export CSV |
| **Dashboard** | View all widgets and trends |
| **Shifts** | View calendars and current on-call |
| **Teams / workspaces** | View configuration (read-only) |
| **Integrations** | View connector list |
| **Account** | Edit own profile and locale |

**Cannot do:** acknowledge, resolve, hand off, bounce, or change any configuration.

Use **Dashboard** for MTTA/MTTR trends, noise analysis, on-call load fairness, and L2→L3 handoff
metrics. Click widget drill-down links to open filtered incident or alert lists.

---

## Support tiers (on-call operations)

Support tier is set on each **team**, not on your user account. When you are on call for a team,
you work incidents assigned to that team using the escalation paths configured by an admin.

### Tier overview

| Tier | Typical name | Role in the chain |
|------|--------------|-------------------|
| **NOC** | Network operations center | First triage from monitoring; escalates to L1 or L2 |
| **L1** | Helpdesk / first line | User-facing issues; escalates to L2 when needed |
| **L2** | Platform / application on-call | Owns most production incidents; hands off deep infra to L3 |
| **L3** | Specialist / senior engineering | Deep investigation; may **bounce** back to L2 with a note |

Teams without a tier badge are general groups (for example admin-only teams) — they do not appear
in tier-specific handoff pickers unless paths are configured.

### Escalation buttons

The incident detail page shows contextual actions based on the **owning team’s tier**:

| Owning tier | Primary escalation label | Example target |
|-------------|-------------------------|----------------|
| NOC | Escalate to L1 / Escalate to L2 | Platform L1, Platform L2 |
| L1 | Escalate to L2 | Platform L2 |
| L2 | Hand off to L3 | Platform L3 |
| L3 | Bounce to L2 | Prior L2 owner |

You only see targets that an admin configured on **Escalation paths** for that team. If no paths
exist, the picker is hidden and a helper message points you to **Teams**.

### Bounce

When L3 cannot resolve an issue or needs L2 context, click **Bounce to L2** (or **Bounce to L1**
when appropriate). A note is required. The incident reassigns to the prior owner; the note appears
in the timeline for everyone.

### Shared timeline

All tiers see the **same timeline events** on an incident — acknowledgements, handoffs, bounces,
and system events. There are no “internal only” rows in the current release. L2 and L3 both see the
full history (REQ-L2L3-03).

---

## Scenarios by persona

### IT admin — new project onboarding

You are adding a **Data** project alongside existing **Platform** work.

1. **Teams** → filter by workspace or create workspace **Data**.
2. Create **Data L2** (tier L2) and **Data L3** (tier L3) in that workspace.
3. On **Data L2** team detail → **Escalation paths** → add **Data L3**.
4. Open workspace **Data** → **Routing rules** → add rule `project=data` → **Data L2**.
5. **Integrations** → add workspace-scoped Jira integration with `project_key=DATA`.
6. **Shifts** → open **Data L2** calendar → create weekly rotation with team members.

Alerts with label `project=data` now route to Data L2, create Jira tickets in `DATA`, and hand off
only to Data L3.

### On-call L2 engineer — production incident

1. Page arrives in Slack while you are primary on **Platform L2**.
2. Tap **Acknowledge** in Slack.
3. Open **Incidents**, select the incident, review alerts and Jira link.
4. Fix or mitigate; post updates in Jira as usual.
5. If you need database specialists, click **Hand off to L3** → **Platform L3**.
6. When resolved, click **Resolve** in the web app (or ask L3 to resolve after bounce).

### NOC operator — triage from monitoring

1. Alert fires; routing sends incident to **Platform NOC**.
2. Acknowledge and check whether it is a known false positive.
3. If it needs application team: **Escalate to L2** → **Platform L2**.
4. If it is user-facing: **Escalate to L1** → **Platform L1**.
5. Timeline records the full chain for later review on **Dashboard**.

### IT manager — weekly review

1. Sign in with **viewer** access (or member if you also respond).
2. Open **Dashboard** — compare MTTA/MTTR to the previous period.
3. Check **On-call load** for unfair paging distribution.
4. Review **L2→L3 handoffs** — median time to L3 first response.
5. Drill into noisy alert fingerprints via **Alerts** → filter → export CSV for a postmortem.

---

## Alerts vs incidents

| | **Alerts** | **Incidents** |
|---|-----------|---------------|
| **What** | Raw signals from monitoring (webhook) | Actionable work items created by Aegis |
| **Who creates** | Grafana, Alertmanager, Zabbix, test tools | Worker after a routing rule matches |
| **Your job** | Search, analyse noise, export for reports | Acknowledge, resolve, escalate |
| **Page** | `/alerts` | `/incidents` |

If you see alerts but no incidents, an admin needs a **routing rule** that matches your alert labels.
Use **Setup** or ask an admin to check **Workspaces** → routing rules.

---

## Chat and paging

| Channel | Sign in | Get paged | Acknowledge |
|---------|---------|-----------|-------------|
| **Slack** | Sign in with Slack (OIDC) | DM to linked `slack_user_id` | Button on page message |
| **eXpress** | Sign in with eXpress (OIDC) | BotX direct message | Bubble action |
| **Web** | Any provider | — | **Incidents** detail buttons |

Link eXpress for paging on **Account** if you sign in with Google or Slack but page via eXpress.

---

## Quick reference

### What each access role can change

| Action | Admin | Member | Viewer |
|--------|:-----:|:------:|:------:|
| Ack / resolve / handoff / bounce incident | ✓ | ✓ | — |
| View incidents, alerts, dashboard | ✓ | ✓ | ✓ |
| Create teams, schedules, overrides | ✓ | — | — |
| Configure routing rules & escalation paths | ✓ | — | — |
| Manage integrations | ✓ | — | — |
| Run setup wizard test alert | ✓ | — | — |
| Save alert filter views | ✓ | ✓ | ✓ |
| Export alerts CSV | ✓ | ✓ | ✓ |
| Edit own account | ✓ | ✓ | ✓ |

### Where to configure what

| I want to… | Go to… |
|------------|--------|
| See who is on call | **Shifts** |
| Add someone to a rotation | **Teams** → team → **Shifts** → schedule (admin) |
| Route alerts to a team | **Workspaces** → routing rules (admin) |
| Allow L2 → L3 handoff | **Teams** → L2 team → escalation paths (admin) |
| Connect Jira / Slack / eXpress | **Integrations** (admin) or **Setup** wizard |
| Work an active outage | **Incidents** or acknowledge from chat |
| Find noisy alerts | **Alerts** or **Dashboard** → noise widget |
| Review response-time trends | **Dashboard** |

---

## Related docs

- Setup and deployment: [`07-setup-deployment.md`](./07-setup-deployment.md)
- Workspaces and tiers (design): [`features/support-levels-and-workspaces.md`](./features/support-levels-and-workspaces.md)
- Incident lifecycle: [`features/incident-management.md`](./features/incident-management.md)
- L2↔L3 handoff and timeline: [`features/l2-l3-transparency.md`](./features/l2-l3-transparency.md)
- Shifts and overrides: [`features/shifts-calendar.md`](./features/shifts-calendar.md)
- Alert search and export: [`features/alerting.md`](./features/alerting.md)
- Integrations: [`integrations/`](./integrations/)
