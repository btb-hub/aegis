**How-to** · [Workflow](README.md) · [Set up](setup.md) · [Rules](routing.md) · [Paths](paths.md) · [Incidents](incidents.md) · [Connect](connect.md) · [Fix it](troubleshooting.md)

**English** · [Русский](ru/incidents.md)

*On-call · member*

# Work an incident from the page to resolve.

You do not configure workspaces on this path. You acknowledge, investigate,
hand off when you need a higher tier, and resolve. Admin setup lives in
[Set up the workflow](setup.md).

need **member or admin** · viewers **cannot ack**

[Get the page](#page) | [Acknowledge](#ack) | [Hand off](#handoff) | [Bounce](#bounce) | [What each role can do](#roles)

<a id="page"></a>

## 01 · Get the page

Incidents show a permanent label such as `INC-01` in the web app and chat.
The same label stays with the incident through handoff and resolution.


When you are primary on a team, a new incident for that team pages you in chat
and appears under **Incidents**.

| Channel | What you see | How you ack |
| --- | --- | --- |
| Slack | DM (and team channel, if configured) | **Acknowledge** on the message |
| eXpress | BotX direct message | Bubble action |
| Web | **Incidents** list | **Acknowledge** on the incident |

If a DM is missing, check **Account → Paging messengers** and confirm the
bot can reach you. For eXpress, open a personal chat with the bot first.
Paging connections are separate from sign-in. See [paging identity](connect.md#paging).

### Find and keep the right incident open

Use the status filter for **All**, **Open**, **Acknowledged**, or **Resolved**.
On a phone it is a **Status** dropdown; wider screens show status buttons.
A wide desktop shows the queue and detail side by side. On smaller screens,
select an incident to open its detail, then use **Back to incidents** to return
to the queue with the same status filter.

Selecting an incident adds its UUID to the page URL as `incident_id`; the readable
`INC-01` label is for people. A direct link opens that incident even when its
status is outside the queue filter. After acknowledgement or resolution, the
selected detail stays open even if the incident leaves the filtered queue.
Changing the status filter clears the explicit selection and returns to the queue
on smaller screens. If loading fails, use **Retry** in the affected panel.

<a id="ack"></a>

## 02 · Acknowledge, then investigate

1. **Acknowledge**
   Click **Acknowledge** in chat or on the incident. The timeline records who and when.
   Use the chat button when you are on a phone or in a war room.

2. **Read the same facts everyone else sees**
   Incident detail: linked **alerts**, Jira ticket (if connected), and the
   **timeline**. There are no hidden L3-only rows. L2 and L3 see the same history.
   The Jira link uses the current Jira base URL for the owning team's workspace,
   including any context path. It follows that workspace's custom or inherited
   connector. If the slot is disabled or incomplete, no Jira link is shown; ask an
   administrator to check **Integrations**.

3. **Resolve when it is fixed**
   Click **Resolve**. Prefer resolving in Aegis so MTTA/MTTR on **Dashboard** stay
   honest. Jira can stay your write-up surface.

<a id="handoff"></a>

## 03 · Hand off to the next tier

The button label follows the owning team’s tier. You only see teams an admin
put on that team’s **Escalation paths**.

| Owning team tier | Button | Typical target |
| --- | --- | --- |
| NOC | Escalate to L1 / Escalate to L2 | Helpdesk or Platform L2 |
| L1 | Escalate to L2 | Platform L2 |
| L2 | Hand off to L3 | Platform L3 |
| L3 | Bounce to L2 | Prior L2 owner |

1. **Hand off to L3**
   On an L2-owned incident, click **Hand off to L3**, pick the configured L3 team,
   confirm **Hand off**. Aegis reassigns, pages that team’s on-call, and (when Jira
   is mapped) can update the ticket assignee.

> **No handoff button**
>
> The owning team has no outgoing path. Ask an admin to add one on
> **Teams** → that team → **Escalation paths**. You cannot invent a target from the incident
> screen.

<a id="bounce"></a>

## 04 · Bounce back

When L3 cannot take it or needs L2 context, click **Bounce to L2**
(or bounce to L1 when that is the path). A note is required. The incident returns to the prior
owner; the note is on the shared timeline.

<a id="roles"></a>

## 05 · Who can change what

| Action | Admin | Member | Viewer |
| --- | --- | --- | --- |
| Ack / resolve / hand off / bounce | Yes | Yes | No |
| View incidents, alerts, dashboard, shifts | Yes | Yes | Yes |
| Create teams, schedules, routing, integrations | Yes | No | No |

Managers: use **Dashboard**
for MTTA, MTTR, noise, on-call load, and L2→L3 handoff time. Drill-down opens the filtered list.
Full role tables: [user-guide.md](../user-guide.md).

---

Alerts vs incidents: raw signals live on **Alerts**; incidents are the work items you ack.
If you have alerts and no incidents, the workflow is not fully set up —
[fix-it](troubleshooting.md).


## Follow incidents in the on-call channel

Slack posts lifecycle messages to the team's configured channel. eXpress uses
the global on-call group. Opening and escalation messages tag on-call users and
include an acknowledge button while the incident is open. Personal pages continue.
Message links open the selected incident, including resolved incidents.

A receipt means the acknowledgement request was queued. For a successful click
in the configured eXpress on-call group, the shared acknowledged post confirms
the change; there is no extra private success reply. Slack, personal eXpress chats,
and unsuccessful or repeated clicks still receive private results. Repeated clicks
report that the incident is already acknowledged or resolved. If your identity
isn't linked, follow the linking guidance in the response.

See [Incident channel notifications](../features/incident-chat-notifications.md).
