# Slack Bot Settings Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan task by task after the user approves implementation. Steps use checkbox syntax for tracking.

**Goal:** An Aegis admin can find, configure, test, and use the Slack bot from Integrations or workspace settings, including working Acknowledge callbacks with credentials saved in the UI.

**Architecture:** Keep the existing integrations table, CRUD endpoints, shared credential form, and inherit/custom resolver. Make Slack setup visible before a global integration exists. Resolve callback verification credentials from the incident's workspace using the same rules as outbound paging.

**Tech stack:** Go/Gin, PostgreSQL, React/TypeScript, react-i18next, Vitest and Go testing.

**Spec:** User request: Aegis settings do not allow adding/configuring the Slack bot. Slack sign-in/account-linking configuration is outside this request. Existing workspace semantics are defined in `docs/superpowers/specs/2026-07-14-workspace-integration-slots-design.md`.

## Evidence and limits

- Pulled `main` with a fast-forward to `13704df` before investigation.
- `/integrations` already implements an admin-only generic Add integration flow with a Slack choice and bot token/signing secret fields. The initial choice is the first missing kind, normally Jira.
- Workspace slots default to Inherit. `IntegrationConfigFields` returns no fields for inherited Slack. The workspace editor offers no direct route to configure a missing global Slack bot.
- Admin controls disappear for members/viewers without an explanation of how to obtain configuration access.
- Slack callback registration receives `cfg.SlackSigningSecret()` from the environment. Saving `signing_secret` in the integration row does not update that value.
- The Slack reference documents `/callbacks/slack/interactive`; the registered and proxied route is `/api/v1/callbacks/slack/interactive`.
- Baseline: 22 targeted frontend tests pass, and API service/handler suites pass.
- No local API/web server was running. The exact deployed screen, role, and deployment version have not been reproduced; UI discoverability is a supported explanation, not a confirmed deployment diagnosis.

## Global constraints

- Preserve one global connector per kind and stable workspace slots.
- Preserve Inherit and Custom behavior, including clearing workspace secrets when switching to Inherit.
- Configuration mutations remain admin-only; never enable edit controls for non-admin users.
- Secrets stay masked in responses and blank in edit forms; omitted/blank edit secrets retain saved values.
- Bot setup must work without `/setup`, environment changes, or an API restart.
- Do not add an OAuth bot-installation flow or change Slack OIDC configuration.
- Existing `auth.test` verifies token authentication; do not present it as proof that delivery, channel membership, or interactive callbacks work.

## Review focus

1. An empty inventory or an inventory containing only workspace slots still exposes global Slack setup.
2. An inherited workspace with no global bot provides a useful next action; Custom still permits independent credentials.
3. Members/viewers see an admin-access explanation and cannot mutate credentials or run an admin-only connection test.
4. Callback signatures from another workspace, disabled slots, incomplete configs, and old secrets are rejected before incident mutation.
5. Editing/rotating a saved secret works without restart and never reveals credentials in UI/API responses.

### Task 1: Expose global Slack setup directly

**Files:**
- Modify `apps/web/src/pages/IntegrationsPage.tsx`.
- Modify `apps/web/src/pages/IntegrationsPage.test.tsx`.
- Modify `apps/web/src/components/integrations/IntegrationConfigFields.tsx` and its test file for setup guidance.
- Modify `apps/web/src/locales/en/common.json` and `apps/web/src/locales/ru/common.json`.

**Interfaces:** Continue using `POST /api/v1/integrations`, `PATCH /api/v1/integrations/:id`, and `POST /api/v1/integrations/:id/test` with the existing payload shapes.

- [x] Add a regression test for direct Slack setup from an empty inventory. Use the existing `authAdmin`, `jsonResponse`, and `renderPage` helpers:

```tsx
it('opens Slack credentials directly when no global bot exists', async () => {
  mockFetch({ items: [] });
  renderPage();
  fireEvent.click(await screen.findByRole('button', { name: 'Configure Slack' }));
  expect(screen.getByLabelText('Slack bot token')).toBeInTheDocument();
  expect(screen.getByLabelText('Slack signing secret')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Save integration' })).toBeDisabled();
});
```

- [x] Run `npm test -- --run src/pages/IntegrationsPage.test.tsx` in `apps/web`; confirm the new assertion fails because no direct Configure Slack button exists.
- [x] Add a persistent Slack setup summary before the inventory filters. Show Not configured, Missing credentials, Disabled, or Configured based on the global Slack row. Derive its state from fetched data; do not create placeholder database rows. Render its action only after loading succeeds.
- [x] Have Configure Slack call `setEditor(emptyEditor('slack'))` when no global Slack row exists, or `openEdit(globalSlack)` when it does. Keep the generic Add integration flow for other kinds.
- [x] For members/viewers, replace the configuration action with “Ask an administrator to configure Slack.” Explain the same restriction near existing read-only integration inventory. Disable/hide Test connection for non-admins because the endpoint requires admin access.
- [x] Add Slack field guidance: obtain the installed app's Bot User OAuth Token and Signing Secret; enable Interactivity with Request URL `<public Aegis origin>/api/v1/callbacks/slack/interactive`. Display the URL as a copyable read-only value derived from the browser's public origin. Explain that Slack must reach that URL over HTTPS and that bot setup and Sign in with Slack are separate.
- [x] Test submitting both credentials produces `kind: 'slack'`, `enabled: true`, and config `{bot_token, signing_secret}`. Test inventory with only workspace Slack slots, existing global Slack, loading/error states, non-admin access, omitted edit secrets, and displayed callback path. Use row/dialog-scoped queries to avoid duplicate labels.
- [x] Run the page and shared-field suites, `npm run typecheck`, and `npm run check:locales`; commit the independently usable setup flow.

### Task 2: Connect inherited workspace settings to global setup

**Files:**
- Modify `apps/web/src/components/integrations/WorkspaceSlotsPanel.tsx`.
- Modify `apps/web/src/pages/IntegrationsPage.tsx`.
- Modify `apps/web/src/pages/IntegrationsPage.test.tsx` and `apps/web/src/pages/WorkspaceDetailPage.test.tsx`.
- Modify English and Russian locale files.

**Interfaces:** Link to `/integrations?configure=slack`. The destination opens the Slack editor only after a successful inventory fetch and admin identity resolution. Consume the parameter once using router search-parameter APIs; switching filters or reloading the inventory must not reopen a closed dialog.

- [x] Add a workspace-editor regression test: an admin opening an inherited Slack slot with `slot_status: 'missing'` sees “Configure global Slack” linking to `/integrations?configure=slack`. Confirm it fails before implementation.
- [x] Add that link and explain “Inherit uses the global Slack bot. Choose Custom to use a separate bot for this workspace.” Keep the inherited form free of secrets.
- [x] Implement the destination behavior using the same create/edit action from Task 1. Reopening an existing global bot must edit rather than create another row.
- [x] Test Custom reveals bot token and signing secret fields, saving sends PATCH to the workspace slot, and switching back to Inherit keeps the existing confirmation and deletes workspace-only secrets.
- [x] Test deep links with an empty inventory, an existing bot, fetch failure, and a non-admin. Test closing the dialog followed by an inventory refresh does not reopen it.
- [x] Run the relevant page/workspace tests, typecheck, and locale check; commit the workspace setup flow.

### Task 3: Use saved Slack credentials for callbacks

**Files:**
- Modify `apps/api/internal/handler/integration.go` (Slack callback handler).
- Modify `apps/api/internal/service/integration.go` and `integration_test.go`.
- Modify `apps/api/internal/service/incident.go` and `incident_test.go`.
- Modify `apps/api/cmd/api/main.go`.
- Update callback constructor calls and fixtures in `apps/api/internal/handler/incidents_test.go`, `incidents_extra_test.go`, `phase2_handlers_test.go`, `phase2_coverage_test.go`, and `phase2_coverage_extra_test.go`.
- Modify `docs/integrations/slack.md`, `docs/how-to/connect.md`, and `docs/07-setup-deployment.md`.

**Interfaces:**

```go
func (s *IncidentService) WorkspaceID(ctx context.Context, incidentID uuid.UUID) (uuid.UUID, error)
func (s *IntegrationService) SlackSigningSecret(ctx context.Context, workspaceID uuid.UUID) (string, error)
func NewSlackCallbackHandler(incidents *service.IncidentService, integrations *service.IntegrationService) *SlackCallbackHandler
```

- [x] Add a handler regression test saving global Slack credentials and an inherited workspace slot in the repository fixture, with `SLACK_SIGNING_SECRET` empty. Sign an interactive acknowledgement with the database secret; expect HTTP 200 and the incident acknowledged. Confirm it fails with the current environment-only handler.
- [x] Implement `IncidentService.WorkspaceID` by loading the incident with `GetIncidentByID`, then its team with existing `GetTeam`, returning `team.WorkspaceID`. Propagate mapped lookup errors; do not add a database schema change or expose a public lookup endpoint.
- [x] Implement `IntegrationService.SlackSigningSecret` by loading the workspace Slack slot and optional global Slack row, calling `resolve.Resolve` with the existing mode/enabled/config values, mapping unavailable results with `integrationResolveError`, and decoding `intslack.Config.SigningSecret`. Never choose the first Slack row or try secrets belonging to other workspaces.
- [x] In the callback handler, read the raw body, parse the acknowledgement only to identify the incident, and resolve its workspace/config. Verify the signature against the unchanged raw body before performing any acknowledgement or user mutation. Invalid payloads and unavailable credentials must terminate the request. Preserve timestamp and HMAC validation.
- [x] Wire the handler to `integrationsSvc` in `main.go`. Database config becomes authoritative, consistent with outbound paging; do not fall back to the environment when a slot is disabled or Custom credentials differ.
- [x] Add tests for Custom mode, missing/disabled global config, disabled/missing workspace slots, incorrect/cross-workspace signatures, expired timestamps, malformed payloads, unknown incidents, config rotation without restart, and repository errors. Assert rejected callbacks do not acknowledge incidents or cancel escalation jobs.
- [x] Correct the documented callback path and explain setup through Integrations, Slack `chat:write`, bot installation/channel membership, and Interactivity. Remove the claim that `SLACK_BOT_TOKEN` configures outbound paging: outbound credentials are read from integration rows. Explain that setting only environment variables does not replace saving the bot through the UI.
- [x] Run `go test ./internal/service ./internal/handler` in `apps/api` and `go test ./integrations/slack ./integrations/resolve` in `pkg`; commit the callback fix with documentation.

## Final validation

- [x] Run `npm test -- --run`, `npm run typecheck`, `npm run lint`, and `npm run check:locales` in `apps/web`.
- [x] Run the API service/handler and Slack/resolver suites listed above.
- [ ] In a running Aegis environment, use an admin account and an empty global Slack inventory. Configure the bot through the visible Slack action, reload to verify persistence, and run Test connection. Repeat through the inherited workspace link and through Custom mode.
- [ ] With a test Slack app/workspace and a linked paging user, deliver an incident DM and acknowledge it. Confirm this works with the environment signing secret unset. Rotate the saved secret and confirm only the new signature works without restarting the API.
- [ ] Verify member/viewer accounts receive the admin-access explanation and API mutation/test requests are rejected. Inspect responses to ensure secrets are redacted.

Real delivery and callback validation requires a Slack app, workspace access, a reachable public HTTPS origin, and a linked Slack user. Unit tests can validate the settings and verification paths without those credentials; report live verification separately.

## Current documentation consulted

- https://docs.slack.dev/tools/python-slack-sdk/web
- https://docs.slack.dev/authentication/installing-with-oauth
- https://docs.slack.dev/authentication/sign-in-with-slack

Implementation completed on `codex/slack-bot-settings`.

Automated verification: `make lint type test` passed, including 297 frontend tests, API and
worker suites, Storybook build, and coverage gates. API handler/service coverage is 90.1%/90.6%;
frontend statement coverage is 94.6%. Regression tests cover saved credentials, inherited and
Custom slots, secret rotation without restart, disabled/missing/incomplete configurations,
cross-workspace signatures, stale/malformed requests, lookup failures, and read-only accounts.
Live Aegis/Slack validation above remains unperformed: no Slack app/workspace credentials or
public HTTPS runtime were supplied.


## Implementation decisions and review

- Used the existing checkout on `codex/slack-bot-settings`, preserving installed dependencies.
  If isolation in a separate checkout is needed later, move the branch to a worktree.
- Followed the approved plan rather than selecting an unrelated backlog story; its documentation
  corrections are part of the requested implementation and remain reviewable in the PR.
- The independent final review found no Critical, Important, or Minor issues. It set aside three
  pre-existing behaviors: future timestamp validation, generic Add integration availability during
  loading, and workspace read-only panels without a separate admin-access explanation.
- Retained the existing timestamp helper as required by the plan. Future-dated signed requests
  remain an adjacent limitation; changing that check should have its own regression coverage.
- Kept the generic Add integration behavior; the new direct Slack action and deep link wait for
  successful loading. The generic action can still be attempted before inventory success.
- Kept workspace detail controls read-only; the Integrations inventory provides the admin-access
  explanation. Visitors who remain only on workspace detail may need to navigate there for it.
