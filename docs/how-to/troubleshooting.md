**How-to** · [Workflow](README.md) · [Set up](setup.md) · [Rules](routing.md) · [Paths](paths.md) · [Incidents](incidents.md) · [Connect](connect.md) · [Fix it](troubleshooting.md)

**English** · [Русский](ru/troubleshooting.md)

*User · stuck states*

# When the workflow is incomplete.

Match what you see in the product. Most “Aegis is broken” reports are a missing
routing rule, an empty rotation, or a person who never signed in. Software install problems
(Docker, ports, `.env`) are in
[07-setup-deployment.md](../07-setup-deployment.md), not here.

<a id="table"></a>

## 01 · Symptom → what to fix

| You see | What it means | Where to click |
| --- | --- | --- |
| Rows in **Alerts**, nothing in **Incidents** | Webhook worked. No routing rule matched the labels. | **Workspaces** → the project → **Add routing rule**. Compare label key/value to the alert. On Alerts you can also open **Configure routing**. |
| Incident exists, **On call now** is empty | Team has no schedule covering this moment. | **Shifts** for that team → **Create schedule** or **Add override**. |
| No **Hand off to L3** (or escalate) button | Owning team has no outgoing escalation path. | **Teams** → owning team → **Escalation paths** → **Add path**. |
| Handoff picker is empty | Path exists but the target team is gone, wrong tier, or in another workspace without the cross-workspace checkbox. | Fix the path on the source team. Enable **Allow team from another workspace** only if you intend a shared L3. |
| Member picker finds nobody | Those people have never signed in. | Ask them to use **Login** once, then **Add member** again. |
| You are not admin and cannot create teams | Your access role is Member or Viewer. | **Account** shows the role. An admin promotes you on **Users**. |
| Incident, but no Jira ticket | Jira slot not ready for that workspace (or skipped). | **Integrations** and the workspace **Integrations** panel. Timeline may show `integration_skipped`. |
| Incident, but no Slack / eXpress page | Connector skipped, or the on-call user has no paging identity. | Test connection. User opens **Account** and links Slack or eXpress. |
| Test alert toast succeeds, still no incident | Test payload labels do not match your rule. | Open the new row on **Alerts**, read labels, align the routing matcher. Worker must be running — if alerts never move, ask whoever operates the host. |
| Wrong Jira project | Workspace still using the global project key. | Workspace → Integrations → Jira **Inherit** with a project key override, or Custom. |

<a id="alerts"></a>

## 02 · Alerts are not incidents

This distinction is the whole product.

|  | Alerts | Incidents |
| --- | --- | --- |
| What | Raw monitoring signals | Owned work items |
| Who creates them | Grafana, Alertmanager, Zabbix, test alert | Aegis, after a routing rule matches |
| Your job | Search, noise, CSV | Ack, resolve, escalate |
| Page | **Alerts** | **Incidents** |

<a id="checklist"></a>

## 03 · Admin go-live checklist

- Each project has a **workspace**.
- L2 and L3 teams exist with the correct **support tier**.
- Paging teams have **members** who have signed in.
- L2 has an **escalation path** to L3.
- **On call now** names a person on every paging team.
- Each workspace has at least one **routing rule** that matches real alert labels.
- **Test connection** passed for every connector you rely on.
- On-call people have Slack and/or eXpress IDs on **Account**.

> **Still installing the app?**
>
> This manual assumes you can already open Aegis in a browser. Host, Docker, secrets, and
> OIDC app registration: [Setup & deployment](../07-setup-deployment.md).

---

Walk the happy path again: [set up the workflow](setup.md).
