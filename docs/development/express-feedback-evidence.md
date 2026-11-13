# eXpress acknowledgment feedback

Plan for the November 13 report of empty bot bubbles after acknowledgment:
1. Reproduce the remaining group reply with shipped English and Russian catalogs.
2. Skip the redundant success reply in the configured on-call group; preserve personal results and guidance.
3. Guard blank bodies at the eXpress provider boundary before authentication or chat lookup.
4. Run regression tests red/green, then repository lint, type and test gates.

The previous fix, merged in PR #70, preserved empty translations and skipped
blank rendered feedback. Its tests used temporary blank catalogs. Shipped
`feedback_acknowledged` translations remain nonempty, so successful group clicks
still produced a separate private notification. This code path is reproducible;
the screenshot alone doesn't establish how the live client rendered that payload
or which worker version was deployed.

GitHub confirms PR #70 merged November 12 at 15:21:54 UTC and release v1.2.5 was
published at 15:22:24 UTC. The affected deployment's running version wasn't checked.

## Observed regression evidence

| Criterion | Owning contract | Behavioral evidence | Red then green |
| --- | --- | --- | --- |
| CHAT-FEEDBACK-GROUP | `pkg/incidentchat/runner.go::sendFeedback` | `pkg/incidentchat/delivery_test.go::TestFeedbackWithShippedTranslations` | Four EN/RU group-success cases failed with two HTTP requests instead of zero. All 26 cases passed after suppressing only successful configured-group feedback. |
| CHAT-FEEDBACK-EMPTY-PROVIDER | `pkg/integrations/express/incident_channel.go::SendAckFeedback` | `incident_channel_test.go::TestSendAckFeedbackBlankAndOriginalBody` | Six blank-body cases made unwanted HTTP requests before the guard. All eight cases passed afterward; original nonblank text and personal-chat lookup are preserved. |

```powershell
go test ./pkg/incidentchat -run TestFeedbackWithShippedTranslations -count=1
go test ./pkg/integrations/express -run '^TestSendAckFeedbackBlankAndOriginalBody$' -count=1
```

Each command ran before and after its implementation change. The earlier
sandbox socket failures weren't counted as regression evidence. Local HTTP tests
required elevated socket access. These tests use actual shipped translations and
provider serialization with fake storage and local HTTP endpoints. They don't
prove how a live eXpress client renders restricted notifications.

## Repository checks

- Lint and type targets passed in `make lint type test`: Go vet/build, ESLint,
  generated-style checks, Storybook production build, and TypeScript.
- The first run without PostgreSQL passed Go tests but failed API coverage at
  85.9% because database-backed tests were skipped. An initial isolated database
  run failed on missing tables; migration 24 then needed its documented
  transaction. These were test setup failures, not regression evidence.
- After applying all 24 migrations to disposable PostgreSQL 16, `make test`
  passed with `DATABASE_URL` and `TEST_DATABASE_URL` pointing at that instance,
  plus test session/webhook secrets and `PUBLIC_URL=http://localhost:8080`.
- Business coverage: shared packages 91.9%, API 90.6%, worker 91.3%.
- Web: 59 files / 352 tests passed, 95.44% statement coverage. Locale parity
  passed for 678 keys. Existing Storybook chunk-size and jsdom navigation
  warnings didn't fail the checks.

## Deployment boundary

The fix requires an updated worker binary/image. It doesn't change database
schemas or require translation edits. Existing queued feedback uses the new
policy after the worker restarts. Shared lifecycle delivery remains independent
and retains its existing retry behavior. No production deployment or live BotX
delivery was performed.
