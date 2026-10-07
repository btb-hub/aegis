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
   Click **Add integration** or **Configure**. Choose **Server / Data Center** or **Cloud**
   separately from authentication. Server/DC normally uses **Bearer** with a PAT; Cloud uses
   **Basic** with email and API token. Fill the base URL (including any context path) and project
   key. **Save integration**, then **Test connection**. A passing test checks authentication;
   send a routed alert to verify ticket creation, assignment, and board visibility.

2. **Slack**
   Click **Configure Slack** in the Slack bot summary. Install your Slack app with `chat:write`,
   then save its **Bot User OAuth Token** and **Signing Secret**. Enable **Interactivity** in the
   Slack app and use the displayed Request URL: `https://<public-aegis-origin>/api/v1/callbacks/slack/interactive`.
   Slack must reach this URL over HTTPS. **Test connection** checks the token; verify a real
   message and its **Acknowledge** button separately. On-call DMs use the user's linked Slack
   user ID; team announcements use the channel ID on the team, where the bot must have access.
   Saved credentials take effect without environment changes or an API restart.
   Bot configuration is separate from Sign in with Slack.

3. **eXpress**
   Save bot ID, CTS host, and secret key, then **Test connection**. Set the global
   **eXpress on-call group chat ID** for shared announcements and incident updates. Add the bot
   to that group. Responders connect eXpress on **Account → Paging messengers** and open a
   personal chat with the bot for private pages.

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

For Slack in **Inherit** mode, click **Configure global Slack** in the slot editor to set up
or edit the global bot. Choose **Custom** and save bot token/signing secret to use a separate
bot for this workspace. Teams use their workspace's bot.

Only administrators can save credentials or test connections. Members/viewers can inspect the
inventory and ask an administrator to complete setup.

Status on the slot:
**Ready**, **Using global**, **Needs setup**, **Missing — no global**, or
**Disabled**. A disabled slot skips that provider at runtime without failing the incident.

<a id="paging"></a>

## 03 · Paging identity on Account

SSO sign-in and paging are separate. A Google login does not, by itself,
give Aegis a Slack DM target.

- Each on-call user opens **Account → Paging messengers** and clicks **Connect** for Slack
  or eXpress. Finish browser authorization and return to Account; no bot command or manual ID
  is needed. This connects paging without adding a sign-in provider.
- **Replace** keeps the existing connection until authorization succeeds. **Disconnect** asks
  for confirmation and stops pages through that messenger, without removing sign-in identities.
- If Connect is unavailable, ask an administrator to configure that provider's OIDC credentials
  and enabled global bot. Slack authorization must use the global bot's workspace; eXpress SSO
  must provide a verified email that resolves to one active messenger user.
- Language (**English** / **Русский**) is also on Account; personal pages and private chat
  results follow the recipient's language.

Administrator prerequisites and legacy API compatibility: [Paging connections](../features/paging-connections.md).

### Shared channels and a real delivery check

Set the team's **Slack channel ID** and optional **Slack team tag ID** (a Slack user group ID) on its team page.
The team's workspace bot must be able to post there. eXpress shared posts use the **global**
on-call group, even when personal pages use workspace credentials.

Create a routed test incident and verify the Jira issue, personal page, and shared post. Click
**Acknowledge** in chat and wait for the private result, then resolve in Aegis and confirm the
channel update. Opening, acknowledgement, resolution, and escalation produce separate posts;
handoffs and bounces are outside this shared-channel feature. The worker must be running.

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
