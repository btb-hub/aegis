**How-to** · [Workflow](README.md) · [Set up](setup.md) · [Rules](routing.md) · [Paths](paths.md) · [Incidents](incidents.md) · [Connect](connect.md) · [Fix it](troubleshooting.md)

**English** · [Русский](ru/README.md)

*User manual*

# How to set up the on-call workflow.

This is the product manual — what you click in Aegis so a firing alert becomes
an owned incident, a Jira ticket, and a page to the person who is actually on call. It is not
a developer install guide. You need an **admin** account to configure the workflow; on-call
engineers use [Work an incident](incidents.md).

who **admin** · pages **Workspaces, Teams, Shifts** · then **routing + connectors**

[Set up the workflow](setup.md) | [Routing rules](routing.md) | [Escalation paths](paths.md) | [Work an incident](incidents.md) | [Jira, Slack, eXpress](connect.md) | [When it does not route](troubleshooting.md)

<a id="picture"></a>

## 01 · What you are building

One workspace is one project (Platform, Data, Payments). Teams in that
workspace take the pages. A routing rule decides which team owns a new alert. Escalation paths
decide where L2 can hand off.

```
monitoring  →  alert (labels)  →  routing rule  →  incident
                                              │
                                              ├─ assign current on-call on the target team
                                              ├─ open Jira ticket (if Jira is connected)
                                              └─ page Slack / eXpress (if connected)

on-call  →  Acknowledge  →  (optional) Hand off to L3  →  Resolve
```

Until routing, a schedule, and
at least one team exist, alerts can land in **Alerts** without ever becoming an incident.
That is expected — not a bug.

<a id="pick"></a>

## 02 · Pick the job

Start with set-up if this is a new project. The other pages assume that work
is already done.

- **[Set up the workflow](setup.md)** — *Admin · first time*. Sign in, create a workspace, L2 and L3 teams, members, an escalation path, a weekly
rotation, a routing rule, then prove it with a test alert.
- **[Routing rules with examples](routing.md)** — *Admin · matchers*. Filled-in forms: one Platform rule, two projects, NOC vs L2 by priority, shared DevOps
in another workspace. Higher priority number is tried first.
- **[Escalation paths with examples](paths.md)** — *Admin · handoff*. Add path on the source team. L2→L3, NOC→L1→L2→L3, Data L2 using Platform L3. Why the
Hand off button is missing.
- **[Work an incident](incidents.md)** — *On-call*. Acknowledge from Slack, eXpress, or the web. Investigate, hand off to L3, bounce back,
resolve. Same timeline for every tier.
- **[Connect Jira, Slack, eXpress](connect.md)** — *Admin · connectors*. Save global credentials, Test connection, then inherit or customise per workspace
(Jira project key, Slack channel).
- **[When the workflow is incomplete](troubleshooting.md)** — *Stuck*. Alerts without incidents, empty on-call banner, missing Hand off button, nobody getting
paged.

<a id="order"></a>

## 03 · The order that works

Do it in this order. Skipping ahead is how you get alerts that never become
incidents, or a handoff picker with no targets.

1. **Workspace** — A project boundary for teams and routing.
2. **Teams + tiers** — L2 owns the page. L3 is the handoff target.
3. **Members + shifts** — Someone must be on call when an alert fires.
4. **Routing** — Match a label (for example `team=platform`) to L2.
5. **Connect + test** — Jira / chat, then **Send test alert**.

> **Setup wizard is optional**
>
> Sidebar **Setup** walks the same ground as a checklist. Prefer the dedicated pages —
> **Workspaces**, **Teams**, **Shifts**, **Integrations** — when you need to change
> one thing later. **Quick setup** on the wizard creates a workspace, L2, L3, and the
> default L2→L3 path in one step.

---

Role reference (what admin vs member can change):
[docs/user-guide.md](../user-guide.md).
Installing the software itself:
[docs/07-setup-deployment.md](../07-setup-deployment.md).
Styled HTML for a docs host lives next to these files.
