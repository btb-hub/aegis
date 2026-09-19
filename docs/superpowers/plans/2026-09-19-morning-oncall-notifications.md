# Morning On-Call Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver daily on-call announcements that tag each Slack team, tag individual on-call engineers in both providers, and send all eXpress announcements to one admin-configured group chat.

**Architecture:** Store Slack user-group IDs on teams and retain team Slack channel IDs. Store the shared eXpress on-call group-chat ID in the global eXpress integration configuration. The worker enumerates all teams, uses the workspace-selected Slack connector for each team, and uses only the global eXpress connector/destination for shared announcements.

**Tech Stack:** Go 1.25, PostgreSQL 16 and golang-migrate, Gin, React 18 + TypeScript + Vitest, existing Slack and eXpress HTTP providers.

**Spec:** `docs/superpowers/specs/2026-09-19-morning-oncall-notifications-design.md`

## Global Constraints

- Keep the existing daily schedule at 03:00 UTC; retain rotation-change and manual publication.
- Do not add dependencies or make live connector calls in tests.
- Preserve `teams.express_chat_id` data, but never use it as the announcement destination.
- Use Slack user-group IDs and render them as `<!subteam^ID>`; individual Slack and eXpress identities remain optional.
- Add every user-facing string in both English and Russian locale files.
- Complete with `make lint type test` green and update public API/data-model/integration documentation.

---

### Task 1: Persist and edit the Slack user-group tag

**Files:**
- Create: `db/migrations/000019_slack_team_tag.up.sql`, `db/migrations/000019_slack_team_tag.down.sql`
- Modify: `pkg/db/models.go`, `pkg/db/teams.go`, `apps/api/internal/handler/team.go`, `apps/api/internal/service/team.go`
- Modify: `apps/web/src/lib/teamTypes.ts`, `apps/web/src/pages/TeamDetailPage.tsx`
- Modify: `apps/web/src/locales/en/common.json`, `apps/web/src/locales/ru/common.json`
- Test: `apps/api/internal/handler/team_test.go`, `apps/api/internal/service/team_test.go`, `apps/web/src/pages/TeamDetailPage.test.tsx`

**Interfaces:**
- Produces `db.Team.SlackUserGroupID *string` and JSON property `slack_user_group_id`.
- Consumes the existing admin `PATCH /api/v1/teams/:id` endpoint and its optional-string normalization.

- [ ] **Step 1: Write failing API and UI tests**

```go
func TestTeamHandlerUpdatesSlackUserGroupID(t *testing.T) {
    patchBody := bytes.NewBufferString(`{"slack_user_group_id":"S012TAG"}`)
    // PATCH an admin team and assert response["slack_user_group_id"] == "S012TAG".
}
```

```tsx
it('saves the Slack team tag ID', async () => {
  // Enter S012TAG in the team notification settings and assert PATCH includes
  // { slack_user_group_id: 'S012TAG' }.
});
```

- [ ] **Step 2: Run focused tests and verify the expected failure**

Run: `cd apps/api && go test ./internal/handler ./internal/service -run SlackUserGroupID` and `cd apps/web && npm test -- --run src/pages/TeamDetailPage.test.tsx`

Expected: the request field, model property, and labelled form control do not exist.

- [ ] **Step 3: Implement the smallest persistence/API/UI slice**

```sql
ALTER TABLE teams ADD COLUMN slack_user_group_id TEXT;
```

```go
type Team struct { /* existing fields */ SlackUserGroupID *string `json:"slack_user_group_id,omitempty"` }
// Add slack_user_group_id to every team SELECT/RETURNING scan, UpdateTeamChannels argument,
// PATCH request field, and TeamJSON only when non-nil.
```

```tsx
// TeamDetailPage sends slack_user_group_id with the other notification settings.
// Use the shared Input component and explanatory EN/RU copy stating that this is a user-group ID.
```

- [ ] **Step 4: Re-run focused tests and verify they pass**

Run: same commands as Step 2.

Expected: tag values round-trip, empty input becomes null, and the form sends the configured ID.

- [ ] **Step 5: Commit the vertical slice**

```bash
git add db/migrations/000019_slack_team_tag.* pkg/db/{models,teams}.go apps/api/internal/{handler/team.go,handler/team_test.go,service/team.go,service/team_test.go} apps/web/src/{lib/teamTypes.ts,pages/TeamDetailPage.tsx,pages/TeamDetailPage.test.tsx,locales/en/common.json,locales/ru/common.json}
git commit -m "feat: configure Slack on-call team tags"
```

### Task 2: Configure one eXpress on-call group chat in admin

**Files:**
- Modify: `apps/web/src/components/integrations/IntegrationConfigFields.tsx`
- Modify: `apps/web/src/locales/en/common.json`, `apps/web/src/locales/ru/common.json`
- Test: `apps/web/src/components/integrations/IntegrationConfigFields.test.tsx`, `apps/web/src/pages/IntegrationsPage.test.tsx`

**Interfaces:**
- Produces optional integration config JSON `{"oncall_group_chat_id":"group-1"}` on global eXpress integrations.
- Consumes the existing global-versus-workspace integration editor; workspace eXpress settings omit this field.

- [ ] **Step 1: Write failing configuration-form tests**

```tsx
it('round-trips the global eXpress on-call chat ID', () => {
  const form = configFormFromItem('express', { bot_id: 'bot', host: 'https://cts', oncall_group_chat_id: 'group-1' });
  expect(buildConfigPayload('express', form, { workspaceOnly: false, keepBlankSecrets: true }))
    .toMatchObject({ oncall_group_chat_id: 'group-1' });
});
```

- [ ] **Step 2: Run the focused web tests and verify failure**

Run: `cd apps/web && npm test -- --run src/components/integrations/IntegrationConfigFields.test.tsx src/pages/IntegrationsPage.test.tsx`

Expected: the form drops `oncall_group_chat_id` and the labelled input is absent.

- [ ] **Step 3: Implement only the global eXpress field**

```tsx
export type IntegrationConfigForm = { /* existing fields */ oncall_group_chat_id: string };
// Populate and serialize the field for kind === 'express'. Render the Input only when
// workspaceOnly is false; do not make it required for connector validity.
```

- [ ] **Step 4: Re-run focused web tests and verify pass**

Run: same command as Step 2.

Expected: a global eXpress edit preserves/submits the chat ID; workspace settings do not present a competing destination.

- [ ] **Step 5: Commit configuration UI**

```bash
git add apps/web/src/components/integrations/{IntegrationConfigFields.tsx,IntegrationConfigFields.test.tsx} apps/web/src/pages/IntegrationsPage.test.tsx apps/web/src/locales/{en,ru}/common.json
git commit -m "feat: configure shared eXpress on-call chat"
```

### Task 3: Route every team announcement to Slack and the shared eXpress chat

**Files:**
- Modify: `pkg/integrations/slack/slack.go`, `pkg/integrations/slack/slack_test.go`
- Modify: `apps/worker/internal/processor/publish_oncall.go`, `apps/worker/internal/processor/publish_oncall_test.go`

**Interfaces:**
- Change `onCallAnnouncer.AnnounceOnCall` to accept `slackUserGroupID string` only for Slack, or introduce a small `onCallDestination{ChannelID, TeamTagID string}` value used by both providers.
- `PublishOnCallProcessor` obtains the global eXpress integration with `GetIntegrationByKind(ctx, "express")`, reads `oncall_group_chat_id`, and constructs that provider from global configuration; it does not inspect `team.ExpressChatID`.
- `ListTeams` replaces `ListTeamsWithChatChannels` in daily and rotation publication so global eXpress can announce teams without a Slack channel.

- [ ] **Step 1: Write failing provider and processor tests**

```go
func TestAnnounceOnCallMentionsConfiguredSlackUserGroupAndEngineers(t *testing.T) {
    // Assert chat.postMessage text contains <!subteam^S012TAG> and <@U123>.
}

func TestPublishOnCallUsesGlobalExpressChatForTeamWithoutSlack(t *testing.T) {
    // Global express config has oncall_group_chat_id=group-1; team has no channels.
    // Assert eXpress announcer receives group-1 and the fingerprint is recorded.
}
```

- [ ] **Step 2: Run focused Go tests and verify the expected failure**

Run: `cd pkg && go test ./integrations/slack -run OnCall` and `cd apps/worker && go test ./internal/processor -run PublishOnCall`

Expected: Slack text lacks the group mention and the processor skips a team with no team-level eXpress chat.

- [ ] **Step 3: Implement the minimal routing change**

```go
// Slack: prefix names with "<!subteam^" + teamTagID + ">" when teamTagID is non-empty.
// Worker: list all teams; use team.SlackChannelID for Slack and the parsed global
// eXpress config oncall_group_chat_id for eXpress; retain soft-failure/fingerprint semantics.
// Treat missing global integration or blank chat ID as no eXpress destination, not a job error.
```

- [ ] **Step 4: Re-run focused Go tests and verify pass**

Run: same commands as Step 2.

Expected: Slack tags the configured group and people; every team can publish through the shared
eXpress chat; either provider may fail without suppressing the other; legacy Express team IDs do not receive calls.

- [ ] **Step 5: Commit worker and provider behavior**

```bash
git add pkg/integrations/slack/{slack.go,slack_test.go} apps/worker/internal/processor/{publish_oncall.go,publish_oncall_test.go}
git commit -m "feat: publish on-call to shared eXpress chat"
```

### Task 4: Publish the contract and run the full quality gate

**Files:**
- Modify: `docs/03-data-model.md`, `docs/04-api-spec.md`, `docs/integrations/express.md`, `docs/how-to/setup.md`
- Verify: `Makefile` targets only; no production code changes

**Interfaces:**
- Documents `teams.slack_user_group_id`, `PATCH /teams/{id}` field, the global eXpress config key, and daily 03:00 UTC behavior.

- [ ] **Step 1: Write documentation assertions as review checks**

```text
Confirm the data model names slack_user_group_id and documents legacy express_chat_id as unused for on-call publication.
Confirm the API example shows slack_user_group_id and the integration guide assigns oncall_group_chat_id to global eXpress configuration.
```

- [ ] **Step 2: Update the four user-facing documents**

```markdown
Slack: set a channel and optional user-group ID per team.
eXpress: set one global on-call group chat ID in the global integration.
```

- [ ] **Step 3: Verify documentation and all automated checks**

Run: `git diff --check && make lint type test`

Expected: no whitespace errors; Go, web, locale, type, lint, coverage, and test gates all pass.

- [ ] **Step 4: Commit documentation and verification-ready state**

```bash
git add docs/03-data-model.md docs/04-api-spec.md docs/integrations/express.md docs/how-to/setup.md
git commit -m "docs: describe on-call notification destinations"
```
