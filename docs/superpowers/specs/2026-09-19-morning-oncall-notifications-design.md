# Morning on-call notifications — design

## Goal

Send the existing daily on-call announcement to Slack and eXpress while letting an administrator
configure the Slack team tag for each team and one shared eXpress group-chat destination. Every
message explicitly notifies the engineers currently on call.

## Scope and decisions

- Keep the current daily `publish_oncall` schedule at 03:00 UTC and retain rotation-change and
  manual **Publish now** publication.
- Slack remains team-scoped: a team keeps its Slack channel ID and gains an optional Slack user
  group ID. The rendered message starts with `<!subteam^ID>` when configured, then lists each
  on-call engineer as `<@USER_ID>` when their identity is available.
- eXpress uses one global group-chat ID, stored in the global eXpress integration configuration
  and edited on the Integrations admin page. The worker publishes every team to that destination
  and uses existing eXpress mention objects for each on-call engineer with a linked huid.
- The old `teams.express_chat_id` values are retained in the schema but are no longer selected as
  on-call announcement destinations. This avoids destructive migration and makes rollback safe.
- A team is eligible when it has a Slack channel or the global eXpress on-call chat is configured.
  A failed provider is logged without preventing the other provider from receiving its message.

## Data and API

- Add nullable `teams.slack_user_group_id` with a reversible migration. Add it to the Team model,
  scans, list/get/update queries, and the existing admin `PATCH /teams/{id}` response contract.
- Extend the eXpress integration JSON config with optional `oncall_group_chat_id`. It is shown only
  when editing a global eXpress integration; workspace-level integrations do not configure a
  competing destination.
- Update the on-call processor store contract to retrieve the global eXpress integration and parse
  its announcement chat ID. Existing eXpress connector credentials and provider construction stay
  unchanged.

## Processing flow

1. Daily, rotation, or manual publishing resolves the current on-call users for each eligible team.
2. Slack posts to that team's configured channel. It prepends the configured user-group mention and
   individually tags linked on-call users.
3. eXpress posts to the single global on-call chat and emits existing per-user mention objects for
   linked on-call users.
4. If either provider succeeds, the current on-call fingerprint is saved. If both attempted
   providers fail, the job is retried by the existing worker semantics.

## UI and documentation

- Team admin settings label the new field “Slack team tag ID (optional)” and explain that it is a
  Slack user-group ID, not an `@handle`.
- The global eXpress integration editor labels the shared setting “On-call group chat ID”.
- Add English and Russian translations; update API, data-model, and integration documentation so
  setup describes per-team Slack versus one shared eXpress destination.

## Validation

- Provider tests prove Slack sends the user-group mention plus individual user mentions and eXpress
  continues to emit mention objects.
- Processor tests cover a Slack-only team, global-eXpress-only team, both destinations, absent
  destinations, and a provider failure that does not suppress the other channel.
- API/service tests cover persistence and JSON output of the Slack team tag; web tests cover both
  admin forms and their submitted payloads.
- Run `make lint type test` before completion.
