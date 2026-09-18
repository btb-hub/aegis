**How-to** · [Workflow](README.md) · [Set up](setup.md) · [Rules](routing.md) · [Paths](paths.md) · [Incidents](incidents.md) · [Connect](connect.md) · [Fix it](troubleshooting.md)

**English** · [Русский](ru/setup.md)

*Admin · first project*

# Set up the workflow so an alert has an owner.

Follow these screens in order. The running example is a **Platform** project
with an L2 team that takes the page and an L3 team that receives handoffs. You must be signed
in as **Admin** (check **Account** in the header). Members cannot create teams, schedules,
or routing rules.

[Sign in](#signin) | [Workspace](#workspace) | [Teams & tiers](#teams) | [Members](#members) | [Escalation](#path) | [Shifts](#shifts) | [Routing](#routing) | [Rules (examples)](routing.md) | [Paths (examples)](paths.md) | [Test alert](#prove)

<a id="signin"></a>

## 01 · Sign in as admin

Everyone uses the same login page. What you can change depends on the role on
your account, not on which button you used to sign in.

1. **Open the app and choose a provider**
   Go to **Login**. Choose **Google**, **Slack**, or **eXpress**. After the
   identity provider redirects back, you land in the app.

2. **Confirm you are Admin**
   Open **Account** from the header. The role badge must say **Admin**. If it says
   Member or Viewer, you can use the product but you cannot finish this manual — ask an
   existing admin to promote you on **Users**.

> **People must exist before you can add them to a team**
>
> Aegis does not invite by email. Each on-call engineer signs in once so their user row exists,
> then you add them on the team. Slack paging needs **Sign in with Slack** (or a linked Slack
> identity). eXpress paging needs a bind on **Account** — see
> [Connect](connect.md#paging).

<a id="workspace"></a>

## 02 · Create a workspace

A workspace is a project. It groups teams, routing rules, and which Jira
project tickets land in. One Aegis can have many workspaces (Platform and Data side by side).

1. **Sidebar → Workspaces**
   Click **Create workspace**. Fill **Workspace name** (for example Platform). Slug
   is optional. Click **Save workspace**.

2. **Open the workspace**
   You will configure teams, integrations, and routing on this detail page. Leave it
   open in a tab; you will come back for routing after teams exist.

> **Shortcut: Setup → Quick setup**
>
> On **Setup**, **Quick setup (new project)** creates the workspace, an L2 team, an L3
> team, and the L2→L3 path together. Then skip to [members](#members) and
> [shifts](#shifts). Use dedicated pages if the project already exists.

<a id="teams"></a>

## 03 · Create teams and set support tiers

Support tier lives on the **team**, not on the person. It controls which
escalation buttons appear on an incident.

| Tier | Typical name | Job in the chain |
| --- | --- | --- |
| NOC | Platform NOC | First triage from monitoring; escalate to L1 or L2 |
| L1 | Helpdesk | User-facing issues; escalate to L2 |
| L2 | Platform L2 | Owns most production incidents; hands off to L3 |
| L3 | Platform L3 | Specialists; can bounce back to L2 with a note |

1. **Sidebar → Teams → Create team**
   Name **Platform L2**. Set **Workspace** to Platform. Set **Support tier** to
   **L2**. Save. Repeat for **Platform L3** with tier **L3**.

2. **Optional chat channels**
   On the team, **Slack channel ID** and **eXpress chat ID** are where Aegis can
   announce who is on call. They are not required for the first routed incident.

Minimum for this manual: one L2
team and one L3 team in the same workspace. Add NOC or L1 later if that is how you triage.

<a id="members"></a>

## 04 · Add members

Team membership decides who appears in the rotation picker and who can be
assigned when that team owns an incident.

1. **Open the L2 team**
   Under **Members**, click **Add member**. Search by name or email. Choose
   **Member** or **Lead** (lead is organisational; it does not replace admin).

2. **Add the same people to L3 if they also take L3 pages**
   A person can be on more than one team. Keep L2 and L3 memberships honest so the wrong
   specialist is not in the L3 rotation.

> **Empty member picker**
>
> If search finds nobody, those people have not signed in yet. Ask them to open Login once,
> then retry **Add member**.

<a id="path"></a>

## 05 · Allow L2 to hand off to L3

A tier badge is not enough. You must add a **path** on the team that will
click the button. Full worked examples (NOC chain, shared L3):
[escalation paths](paths.md).

1. **Teams → Platform L2 → Escalation paths**
   Open the **source** team (L2), not L3. Under **Outgoing paths** click
   **Add path**.

2. **Fill the Add path form**
   #### Add path · team **Platform L2**

| Field | Value |
|-------|-------|
| Allow team from another workspace | Off |
| To team | Platform L3 |

Save. L2 outgoing lists Platform L3. Open Platform L3 — incoming lists
Platform L2. On an L2 incident you will see **Hand off to L3**.

Legal pairs only: NOC→L1 or L2,
L1→L2, L2→L3. If the select is empty, the L3 team is missing or has no tier — see
[paths: empty picker](paths.md#empty).

<a id="shifts"></a>

## 06 · Put someone on call

Routing assigns the incident to the team. The current on-call on that team
is who Aegis pages. Without a schedule, you can still get an incident, but paging has nobody
to target.

1. **Open the L2 shifts page**
   From the team, open shifts, or use sidebar **Shifts** and select Platform L2. If
   you see **This team has no on-call schedule yet**, continue.

2. **Create schedule**
   Click **Create schedule**. Set a name, **Timezone (IANA)** (for example
   `Europe/Moscow`), **Handoff weekday** and **Handoff time**, then
   **Participants (in rotation order)**. Save schedule.

3. **Confirm On call now**
   The banner at the top of the calendar must name a person. If it says
   **No one is on call right now**, the rotation does not cover this moment — add an
   **Add override** for today or fix handoff weekday/time.

Add a schedule on L3 as well
if L3 should be paged on handoff. Overrides cover one-off swaps without editing the rotation.

<a id="routing"></a>

## 07 · Route alerts to the L2 team

Create the rule on the **workspace** (Workspaces → Platform →
**Routing rules** → **Add routing rule**). One form = one label matcher. Full examples
(two projects, NOC vs L2, shared DevOps): [routing rules](routing.md).

#### Add routing rule · workspace **Platform**

| Field | Value |
|-------|-------|
| Allow team from another workspace | Off |
| Target team | Platform L2 |
| Label key | `team` |
| Label value | `platform` |
| Priority | `100` (higher number is tried first if two rules match) |

Grafana/Alertmanager must send that exact pair (case-sensitive). Extra labels
on the alert are fine. If you see rows in **Alerts** but nothing in **Incidents**,
copy a key/value from the alert detail into this form.

Webhook URL:
`{your Aegis origin}/api/v1/alerts/webhook`. Do not route new alerts at L3 — L3
gets work from a path, not from this rule.

<a id="prove"></a>

## 08 · Prove the chain

Connectors are optional for this check. Jira tickets and chat pages need
[Connect Jira, Slack, eXpress](connect.md) first.

1. **Send test alert**
   Open sidebar **Setup**, go to **Test alert**, click **Send test alert**. You
   should see a toast with an alert id.

2. **Open Incidents**
   A new incident should list for Platform L2, assigned to whoever the banner named.
   Open it: timeline, linked alert, and (if connected) a Jira link.

> **You are done with first-project setup when**
>
> Every paging team has a schedule that covers now. Every workspace has at least one routing
> rule. L2 teams have an L3 path. If you use Jira or chat, **Test connection** has passed
> on **Integrations**.

### Add another project later

Same screens, new names. Example: Data beside Platform.

- **Workspaces** → Create workspace **Data**.
- Create **Data L2** and **Data L3** with the right tiers.
- On Data L2, add path to Data L3.
- Routing: `project=data` → Data L2.
- On the Data workspace, set Jira to inherit global credentials with project key
  `DATA` (see [Connect](connect.md)).
- Create the Data L2 schedule.

---

Next: [work an incident](incidents.md) as on-call, or
[connect Jira and chat](connect.md) so tickets and pages fire.
