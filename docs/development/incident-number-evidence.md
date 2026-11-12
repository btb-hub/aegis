# Jira assignment and incident numbers

Plan for the explicit November 12 request:
1. Add regression tests for initial Jira assignment and Server/DC username lookup.
2. Preserve UUID identity and allocate permanent global incident numbers in migration 24.
3. Expose `display_id` to the web and use it in personal pages, lifecycle posts and feedback.
4. Test migration/backfill/concurrent inserts in isolated PostgreSQL 16, then run repository gates.
5. Review migration and integration boundaries independently before handoff.

Execution evidence is appended after the checks run. Tests use local HTTP fixtures, not live Jira or BotX.

## Contracts and observed TDD

| Criterion | Source contract | Tests | Observed result |
| --- | --- | --- | --- |
| JIRA-CREATE-ASSIGN | `apps/worker/internal/processor/notify_incident.go` | `TestNotifyIncidentAssignsCreatedTicket`, `TestNotifyIncidentReconcilesHandoffDuringCreationOrAssignment` | Red: zero assignments, then stale assignee for both handoff interleavings. Green after creation assignment and reconciliation. |
| JIRA-ASSIGN-LOOKUP | `pkg/integrations/jira/jira.go` | `TestServerAssigneeFallback` | Red: 422 for exact and hidden-email matches when email search returned `[]`. Green after issue-scoped assignable search; conflicting/inactive/fuzzy/ambiguous matches make no PUT. |
| INC-NUM-AC1 / INC-NUM-DB | `pkg/db/incident_number.go`, migration 24 | `TestIncidentDisplayID`, `TestIncidentNumberMigrationAndStore` | Red: empty label and missing number column. Green on real PostgreSQL 16. Additional empty-install/manual-dedup boundaries added afterward and passed. |
| INC-NUM-API | `apps/api/internal/service/incident.go` | `TestIncidentJSONDisplayID` | Red: absent display_id; green: INC-01 and unchanged UUID. |
| INC-NUM-UI | `apps/web/src/lib/incidentTypes.ts` | adapter, IncidentList and IncidentDetail suites | Red: 4 failures for missing mapping/label/accessibility. Green: 27 targeted tests; actions keep UUID. |
| INC-NUM-CHAT | worker mapping, `pkg/integrations/incident_channel.go`, SendPage, runner | mapper, DM provider, lifecycle and runner tests; escalation flow test | Red: UUID in text or missing label; green: INC-01 in personal/lifecycle/feedback text, UUID in actions. Legacy queued snapshots use current stored number. |

Commands below ran from the repository root unless a working directory is stated.
Local socket tests needed escalation because the Windows sandbox blocked localhost.
That sandbox failure was not counted as regression evidence. The first isolated
escalation test also lacked i18n setup; after loading templates, it failed on the
actual UUID label before implementation, then passed.

```powershell
go test ./pkg/integrations/jira -run TestServerAssigneeFallback -count=1
go test ./apps/worker/internal/processor -run TestNotifyIncidentAssignsCreatedTicket -count=1
go test ./apps/worker/internal/processor -run TestNotifyIncidentReconcilesHandoffDuringCreationOrAssignment -count=1
$env:TEST_DATABASE_URL='<isolated PostgreSQL 16 DSN>'
go test ./pkg/db -run 'TestIncident(DisplayID|NumberMigrationAndStore)' -count=1
go test ./apps/api/internal/service -run TestIncidentJSONDisplayID -count=1
go test ./pkg/integrations -run TestIncidentBodyUsesDisplayID -count=1
go test ./pkg/incidentchat ./pkg/integrations/express ./apps/worker/internal/processor -run 'Test(ChannelRunnerProvidersAndCurrentState|SendPageUsesFixture|IncidentRefCarriesDisplayID)' -count=1
go test ./pkg/integrations/slack -run TestSendPageUsesFixture -count=1
go test ./apps/worker/internal/processor -run TestEscalateProcessorPagesOpenIncident -count=1
# apps/web:
npm test -- --run src/lib/incidentsApi.test.ts src/components/incidents/IncidentList.test.tsx src/components/incidents/IncidentDetail.test.tsx
```

## Integration checks

- `make lint type test`: lint and type targets passed; all selected Go suites passed.
  Package business coverage was 91.2%, worker 91.8% at that run. The API coverage
  gate failed at 85.9% (3154/3671), below the required 90%.
- Clean `origin/main` a42ccdd was extracted under ignored `.cache/baseline-main`.
  Running the same API coverage suite and checker reproduced exactly 85.9%
  (3154/3671). No thresholds were changed and unrelated coverage was not expanded.
- The web steps skipped by make after that failure were run separately in
  `apps/web`: `npm run check:locales` passed (678 keys), and
  `npm test -- --run --coverage` passed 59 files / 352 tests, 95.44% statements.
- After review repairs: `go test ./pkg/incidentchat ./pkg/integrations/... ./apps/worker/internal/processor ./apps/api/internal/service -count=1` passed.
- Final `make lint type` passed again after review repairs, including Go vet/build,
  ESLint, styles, Storybook build and TypeScript. Final worker coverage rerun
  `go test '-coverprofile=apps/worker/coverage-worker.out' ./apps/worker/internal/processor -count=1`
  and `go run ./scripts/check_coverage.go apps/worker/coverage-worker.out` passed
  at 91.3% (440/482). The migration suite passed again after its empty/manual cases.
- OpenAPI YAML parsed and HTML report links resolved; `git diff --check` passed.
- PostgreSQL test passed chronological backfill, 12 concurrent inserts, create,
  list/get/fingerprint/linked reads, acknowledge, handoff, bounce, resolve, locked
  chat-ack read, down migration preserving rows, empty-install numbering, manual
  creation and dedup. It creates and removes a private schema, never deployed data.
- Independent Terra review requested changes for escalation labels and concurrent
  handoff assignment; both received regression tests and fixes. Re-review approved
  with no unresolved actionable findings. Reviewer inspected source, not live services.

## Rollout boundary

Migration 24 takes an exclusive lock while numbering existing rows. Apply it
before starting updated API/worker binaries. Existing UUID links stay valid.
Numbers are global, start at `INC-01`, grow past two digits and may have gaps.
Rolling migration 24 down removes labels and its owned sequence, preserving
incidents and UUID relations; updated binaries require the column, so roll back
the binaries too. A later up after down can renumber records.

Validation used local fixtures and an isolated database. No live Jira or BotX
calls, production migration or deployment were performed. The existing incident named in the request will receive its label when
the migration is applied; its actual number depends on production creation order.
