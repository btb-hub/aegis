# Product requirements (MVP)

Numbered requirements win over stories. Stories reference these IDs.

## Cross-cutting

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-1 | Clean host to first successfully routed alert in ≤ 1 working day (wizard + docs). |
| NFR-2 | Alert list API p95 < 500 ms at 10k rows with default indexes. |
| NFR-3 | API errors return `{code, message, details}` JSON. |
| NFR-4 | Secrets only via env/config; never in code, logs, or committed fixtures. |
| NFR-5 | Non-IaC application code maintains ≥ 90% unit-test coverage on business logic (see [`10-agent-loop.md`](./10-agent-loop.md)). |

### Authentication & authorization

| ID | Requirement |
|----|-------------|
| REQ-AUTH-01 | Users sign in via OIDC only: Google, Slack, or eXpress. |
| REQ-AUTH-02 | No local passwords or self-hosted IdP in MVP. |
| REQ-AUTH-03 | Server-side session after OIDC callback; logout invalidates session. |
| REQ-AUTH-04 | RBAC roles: `admin`, `member`, `viewer` (minimum). |
| REQ-AUTH-05 | All mutating API routes require authenticated session except webhooks/callbacks. |

### Audit

| ID | Requirement |
|----|-------------|
| REQ-AUDIT-01 | Security-relevant actions (login, role change, integration change) written to audit log. |

### Localization

| ID | Requirement |
|----|-------------|
| REQ-I18N-01 | Web UI available in English (`en`, default) and Russian (`ru`). |
| REQ-I18N-02 | User selects locale via in-app language switcher; preference stored on `users.locale`. |
| REQ-I18N-03 | Unsigned visitors: locale from browser `Accept-Language` with fallback to `en`. |
| REQ-I18N-04 | Chat page templates (Slack, eXpress) use the recipient's saved locale. |
| REQ-I18N-05 | New UI or template strings ship in both locales in the same change. |
| REQ-I18N-06 | Dates, numbers, and relative times in the UI follow the active locale (`Intl`). |

### Design system

| ID | Requirement |
|----|-------------|
| REQ-DS-01 | Web UI uses shared tokens and components per [`12-design-system.md`](./12-design-system.md). |
| REQ-DS-02 | Severity colors (P1–P4, resolved) match the design system across list, detail, and badges. |
| REQ-DS-03 | One primary action per view; buttons and toasts use consistent labels per CLAUDE.md writing rules. |

---

## §1 Shifts calendar

| ID | Requirement |
|----|-------------|
| REQ-SHIFT-01 | Admins create teams and assign members with roles. |
| REQ-SHIFT-02 | Admins define recurring schedules (weekly rotation) per team. |
| REQ-SHIFT-03 | Admins create one-off overrides (swap or cover a slot). |
| REQ-SHIFT-04 | System resolves "who is on call now" for a team at any instant, including DST transitions. |
| REQ-SHIFT-05 | On-call slots are materialised ahead for fast reads. |
| REQ-SHIFT-06 | Calendar UI shows rotations and overrides; "now" indicator visible. |
| REQ-SHIFT-07 | API exposes current on-call per team. |

---

## §2 Incident management

| ID | Requirement |
|----|-------------|
| REQ-INC-01 | Generic alert webhook accepts Alertmanager/Grafana/Zabbix-style payloads. |
| REQ-INC-02 | Raw alerts stored; processing is async via `jobs` table. |
| REQ-INC-03 | Dedup/group alerts by configurable fingerprint within a time window. |
| REQ-INC-04 | Routing rules map alert labels to a team. |
| REQ-INC-05 | New incident created for a firing alert group; state machine: open → acknowledged → resolved. |
| REQ-INC-06 | Incident assigned to team's current on-call at creation. |
| REQ-INC-07 | Jira ticket auto-created and linked on incident creation. |
| REQ-INC-08 | On-call paged via enabled chat connectors (Slack, eXpress). |
| REQ-INC-09 | Acknowledge from chat or UI updates incident + timeline. |
| REQ-INC-10 | Escalation timer: if unacked within policy, one escalation step (re-page or next person). |
| REQ-INC-11 | Incident timeline records all state changes, pages, acks, ticket links. |

---

## §3 Alerting workspace

| ID | Requirement |
|----|-------------|
| REQ-ALERT-01 | Paginated alert list with filters (severity, team, status, time range, labels). |
| REQ-ALERT-02 | Full-text search on alert title/body. |
| REQ-ALERT-03 | Group by label key or severity. |
| REQ-ALERT-04 | Saved views (name + filter JSON) per user or team. |
| REQ-ALERT-05 | Inline analytics on current filter slice (counts by severity, top labels). |
| REQ-ALERT-06 | CSV export of current slice (streamed). |
| REQ-ALERT-07 | List performant at 10k+ rows (NFR-2). |

---

## §4 L2↔L3 transparency

| ID | Requirement |
|----|-------------|
| REQ-L2L3-01 | L2 can hand off incident to L3 team's current on-call in one action. |
| REQ-L2L3-02 | Handoff reassigns owner, pages L3, updates Jira assignee if configured. |
| REQ-L2L3-03 | L2 and L3 see identical incident timeline (no hidden events). |
| REQ-L2L3-04 | L3 can bounce back to L2 with reason. |
| REQ-L2L3-05 | Handoff events recorded for analytics (time to first L3 response). |

---

## §5 Integrations

| ID | Requirement |
|----|-------------|
| REQ-INT-01 | Connector interface: ticket vs chat providers; registry pattern. |
| REQ-INT-02 | Jira: create issue, link to incident, optional status sync inbound. |
| REQ-INT-03 | Slack: Block Kit page, signed interactive ack callback. |
| REQ-INT-04 | eXpress: BotX HTTP API, `/link` identity bootstrap, ack via bubble action. |
| REQ-INT-05 | Per-connector "Test connection" in admin/setup. |
| REQ-INT-06 | Connector failure does not block other connectors; retry with backoff in worker. |

---

## §6 Analytics & setup

| ID | Requirement |
|----|-------------|
| REQ-AN-01 | Dashboard answers: MTTA trend, MTTR trend, top noisy alerts, on-call load fairness, L2→L3 stats. |
| REQ-AN-02 | Drill-down from dashboard widgets to incident/alert lists. |
| REQ-AN-03 | Compare metrics to previous period (week/month). |
| REQ-AN-04 | Setup wizard: org basics → OIDC check → integrations → test alert. |
| REQ-AN-05 | "Send test alert" button in wizard fires sample webhook. |

---

## References

- Feature specs: [`features/`](./features/)
- Integrations: [`integrations/`](./integrations/)
- API: [`04-api-spec.md`](./04-api-spec.md)
- Localization: [`11-localization.md`](./11-localization.md)
- Design system: [`12-design-system.md`](./12-design-system.md), [`design_system.html`](./design_system.html)
