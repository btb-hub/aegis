**How-to** · [Workflow](README.md) · [Set up](setup.md) · [Rules](routing.md) · [Paths](paths.md) · [Incidents](incidents.md) · [Connect](connect.md) · [Fix it](troubleshooting.md)

**English** · [Русский](ru/connect.md)

*Admin · connectors*

# Connect Jira, Slack, and eXpress from the UI.

Tickets and pages are not required for an incident to exist, but they are how
the rest of the company sees the work. Configure connectors on **Integrations** and on
each workspace. You do not need the setup wizard to change one credential.

[Global connectors](#global) | [Workspace slots](#slots) | [Paging identity](#paging) | [When a connector is skipped](#skip)

<a id="global"></a>

## 01 · Save global connectors

Sidebar → **Integrations**. These credentials are the default every
workspace can inherit.

1. **Jira**
   Click **Add integration** or **Configure** on the Jira row. Fill
   **Jira base URL**, **Jira email**, **Jira API token**, and a default
   **Jira project key**. **Save integration**, then **Test connection**.

2. **Slack**
   Bot token and signing secret. Test connection. On-call DMs use the user’s Slack user ID
   (from Sign in with Slack). Team announcements use the channel ID on the team.

3. **eXpress**
   Bot ID, host, and secret key. Test connection. Users who signed in with Google or Slack
   still bind eXpress on **Account** before they can be paged there.

> **Checkpoint**
>
> Each row you care about shows as enabled and Test connection succeeds. Incomplete rows
> show **Add credentials to finish setup** — open **Configure**.

<a id="slots"></a>

## 02 · Per workspace: inherit or custom

Every workspace has a Jira, Slack, and eXpress slot. Open the workspace →
**Integrations**.

| Mode | When to use it |
| --- | --- |
| **Inherit** | Use the global connector. Jira can still override **project key** for this<br>workspace (Platform → `OPS`, Data → `DATA`). Slack can override<br>channel. Switching back to Inherit deletes workspace-only secrets for that slot. |
| **Custom** | This workspace has its own complete credentials and does not mix in global ones. |

Status on the slot:
**Ready**, **Using global**, **Needs setup**, **Missing — no global**, or
**Disabled**. A disabled slot skips that provider at runtime without failing the incident.

<a id="paging"></a>

## 03 · Paging identity on Account

SSO sign-in and paging are separate. A Google login does not, by itself,
give Aegis a Slack DM target.

- Each on-call user opens **Account**.
- Slack: **Connect Slack** / sign in with Slack so **Slack user ID** is set.
- eXpress: **Generate link code** and complete the `/link` flow in eXpress
  so **eXpress user ID** is set.
- Language (**English** / **Русский**) is also on Account — pages follow the
  recipient’s language.

<a id="skip"></a>

## 04 · A missing connector does not block ingest

If Jira or chat is not ready, Aegis still stores the alert and can still
open an incident. The timeline gets an `integration_skipped` event with a reason
(slot disabled, no global, incomplete custom config). Fix the slot; do not re-send the
webhook unless you also want a duplicate alert.

> **Setup wizard**
>
> **Setup** can save the same Jira / Slack / eXpress forms. Day-two changes belong on
> **Integrations** and the workspace panel.

---

Connector contracts: [docs/integrations](../integrations/README.md).
Back to [set up the workflow](setup.md).
