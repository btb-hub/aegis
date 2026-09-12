**How-to** · [Workflow](README.md) · [Set up](setup.md) · [Rules](routing.md) · [Paths](paths.md) · [Incidents](incidents.md) · [Connect](connect.md) · [Fix it](troubleshooting.md)

**English** · [Русский](ru/routing.md)

*Admin · routing rules*

# How a label on an alert chooses a team.

A routing rule is one matcher: **this label key equals this value → send the
incident to that team**. You create the rule on the **workspace** that owns the project,
not on the team. If no rule matches, the alert stays on **Alerts** and never becomes an
incident.

[Where to click](#where) | [What each field means](#form) | [Example: Platform](#ex1) | [Example: two projects](#ex2) | [Example: NOC first](#ex3) | [Example: shared team](#ex4) | [Read labels from an alert](#read)

<a id="where"></a>

## 01 · Where to create a rule

1. **Open the workspace, not Teams**
   Sidebar **Workspaces** → click **Platform** (or your project). Scroll to
   **Routing rules**. Click **Add routing rule**. The modal is
   **Add routing rule** / **Save rule**.

2. **One rule = one matcher**
   The form has a single **Label key** and **Label value**. Need two conditions
   (`team=platform` and `env=prod`)? Create two rules only if you
   want either to match. The UI does not AND two keys on one rule. Extra labels on the
   alert are ignored — they do not have to appear on the rule.

<a id="form"></a>

## 02 · What each field means

| Field on the form | What to type |
| --- | --- |
| **Allow team from another workspace** | Leave off unless the target team lives in a different workspace (shared DevOps). When<br>off, **Target team** lists only teams in this workspace. |
| **Target team** | Who owns the incident. Almost always the L2 team (or NOC if they take first look).<br>Not L3 — L3 receives work via an escalation path, not from routing. |
| **Label key** | Exact label name from the monitoring payload. Common: `team`,<br>`project`, `service`. Must match spelling and case. |
| **Label value** | Exact value. `platform` does not match `Platform` or<br>`platform-prod`. |
| **Priority** | Integer. Default `100`. **Higher number is tried first** when two rules<br>both match the same alert. Equal priority: the team with the lower id wins (do not rely<br>on that — give them different numbers). |

> **Matching is global**
>
> Aegis evaluates rules from every workspace together. A high-priority rule on Data can
> steal a Platform alert if both match the same labels. Keep matchers specific
> (`team=…` or `project=…`), not a bare `severity=critical`
> unless you really want all criticals in one team.

<a id="ex1"></a>

## 03 · Example 1 — one Platform rule

Grafana sends labels `alertname=HighCPU`,
`team=platform`, `severity=warning`. You want Platform L2 to own it.

On workspace **Platform**, **Add routing rule**, fill exactly:

#### Add routing rule · workspace **Platform**

| Field | Value |
|-------|-------|
| Allow team from another workspace | Off |
| Target team | Platform L2 |
| Label key | `team` |
| Label value | `platform` |
| Priority | `100` |

The rule matches because the alert has `team=platform`.
`alertname` and `severity` are extra and do not need their own rules.
Click **Save rule**. The table shows matchers `team=platform`, team Platform L2,
priority 100.

<a id="ex2"></a>

## 04 · Example 2 — Platform and Data side by side

Same Grafana, two projects. Alerts use `project=platform` or
`project=data`.

#### Rule A · workspace **Platform**

| Field | Value |
|-------|-------|
| Target team | Platform L2 |
| Label key | `project` |
| Label value | `platform` |
| Priority | `100` |

#### Rule B · workspace **Data** (create this rule on the Data workspace)

| Field | Value |
|-------|-------|
| Target team | Data L2 |
| Label key | `project` |
| Label value | `data` |
| Priority | `100` |

Same priority is fine: an alert cannot match both values at once. Do not put both
rules on the Platform workspace unless Data L2 is a cross-workspace target.

<a id="ex3"></a>

## 05 · Example 3 — NOC takes critical, L2 takes the rest

You have Platform NOC (tier NOC) and Platform L2. Critical alerts should go
to NOC first; everything else with `team=platform` goes to L2.

#### Rule A · higher priority · workspace **Platform**

| Field | Value |
|-------|-------|
| Target team | Platform NOC |
| Label key | `severity` |
| Label value | `critical` |
| Priority | `200` |

#### Rule B · catch-all for the team · workspace **Platform**

| Field | Value |
|-------|-------|
| Target team | Platform L2 |
| Label key | `team` |
| Label value | `platform` |
| Priority | `100` |

Alert `team=platform` + `severity=critical` matches both
rules. `200` is tried first → NOC. A warning with only `team=platform`
matches only rule B → L2. Warning: rule A also matches *Data* criticals if they use
the same `severity=critical` label. Prefer
`team=platform-noc` in Grafana if you need a hard split.

<a id="ex4"></a>

## 06 · Example 4 — route to a team in another workspace

DevOps lives in workspace **Ops**. Platform still receives the Grafana
alerts and should hand ownership to DevOps.

#### Add routing rule · workspace **Platform**

| Field | Value |
|-------|-------|
| Allow team from another workspace | On |
| Target team | DevOps (Ops) |
| Label key | `team` |
| Label value | `devops` |
| Priority | `100` |

The select shows `DevOps (Ops)` only after the checkbox is on. The
incident is owned by DevOps; Jira/Slack follow the Ops workspace slots. The rule still
appears on the Platform routing table because that is where you created it.

<a id="read"></a>

## 07 · If you do not know the label names

- Send one real alert (or **Setup** → **Send test alert**).
- Open **Alerts**. Open the row. Read the labels on the detail.
- Copy one key/value pair into **Add routing rule**. Save.
- Send again (or wait for the next fire). You should now see an incident.

> **Checkpoint**
>
> The routing table on the workspace lists at least one rule. A new matching alert appears
> under **Incidents**, assigned to the target team. Next: that team needs
> [escalation paths](paths.md) and someone on call.

---

Continue with [escalation paths](paths.md) so L2 can click
**Hand off to L3**. Full first-project order: [set up the workflow](setup.md).
