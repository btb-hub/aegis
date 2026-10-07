# Incident channel notifications

Aegis posts new incidents, acknowledgements, resolutions, and escalations to
the team's Slack channel and the global eXpress on-call group. Existing personal
pages and escalation DMs continue to work independently.

Each event is a separate message with incident ID, severity, title, owning team,
event time in UTC, and a link that selects the incident in Aegis. Acknowledgement
and resolution posts identify the actor. Opening and escalation posts tag the
configured Slack user group or the incident team's current eXpress on-call users.
They include an acknowledge button while the incident is still open. Missing
on-call identities do not prevent a channel post.

Slack uses the team's enabled workspace slot, including Inherit/Custom rules.
eXpress uses the enabled global connector's `oncall_group_chat_id`. No new
settings or historical backfill are required. Handoffs and bounces are outside
this feature. Channel language follows the assignee, with English as the default.

## Delivery and acknowledgement

Migration 000021 adds independent channel and acknowledgement outboxes. The
timeline trigger queues creation, acknowledgement, and resolution events in the
same transaction as the state change. An escalation job queues one channel event
per job before sending its existing DMs. Event details and channel destinations
are captured when queued. Events preserve order per incident and destination.

The worker processes channel messages, acknowledgements, and feedback in
independent loops, so slow message delivery doesn't delay incident acknowledgement.
It returns failed deliveries to the queue for up to five attempts, with delays
of 1, 5, 30, and 120 seconds. Permanent provider/configuration failures stop
immediately. Provider `Retry-After` deadlines can extend these delays. A failed destination doesn't repeat a successful destination's post
or an existing DM. Expiring leases recover work after a worker restart.

Bot endpoints authenticate and durably queue acknowledgement requests before
returning receipt responses: HTTP 200 for Slack, HTTP 202 for eXpress. Receipt
doesn't mean the incident has been acknowledged. The worker sends a separate
result after the database commits. Slack uses an ephemeral response; eXpress
restricts the response in the originating chat to the clicking user's HUID.
Feedback uses the linked user's language. Unlinked users receive linking guidance.
Repeated clicks report the existing acknowledged or resolved state without
another state transition. Existing linked-user eligibility is preserved.

eXpress deduplication uses the command `sync_id`; Slack uses a hash of the verified
interaction body. Persisted outcomes keep feedback retries from repeating the
action. New eXpress channel/feedback sends use the synchronous BotX v4 endpoint
so recorded delivery success means the send finished. Existing DM sends retain
their existing endpoint.

## Deployment and troubleshooting

Apply database migrations before rolling out API and worker changes. The API
accepts actions; the worker must be running to process them and send feedback.
Invite the bots to configured channels and allow outbound HTTPS to the provider.

Inspect `incident_channel_events` for immutable event snapshots,
`incident_channel_deliveries` for per-destination attempts, status, external
reference and errors, and `chat_ack_requests` for processing outcomes and feedback
status. Failed entries remain available for inspection. Logs identify incidents,
events, providers, destinations and attempts. The callback response URL is a
credential and is stored only in the acknowledgement queue, never logged.

Delivery is at least once: if a provider delivers a message but the connection
drops before its confirmation is recorded, a retry can produce a duplicate.
