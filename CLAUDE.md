# CLAUDE.md — How to work in this repo

You are a coding agent working on Aegis. This file is your contract. Read it fully before touching
anything. The full version of the loop lives in [`docs/10-agent-loop.md`](./docs/10-agent-loop.md);
this is the short, always-on version.

## The loop (do this every time)

1. **Sync.** Check out and pull the latest `main`. Never start work on a stale branch.
2. **Pick.** Open `backlog/epics/` and take the highest-priority story with status `Ready` whose
   dependencies are all `Done`. An explicit user request defines the scope when no story is given.
   Without either a request or a Ready story, stop and report.
3. **Branch.** Create a new `codex/<short-slug>` branch off fresh `main`, unless the user specifies
   another name. One request or story per branch.
4. **Plan.** Write a 3–8 line plan as a comment in the story file (or PR description): what you'll
   change, which files, how you'll test. Keep it honest and short.
5. **Build.** Implement the story. Write tests alongside the code, not after.
6. **Verify.** Application changes require `make lint type test`. For documentation-only changes,
   check links, examples against source, Markdown/HTML parity, and `git diff --check`. Report the
   checks actually run and any limitations in the PR.
7. **Self-review.** Re-read the story's acceptance criteria and the Definition of Done below. Tick
   each one. If anything fails, you're not finished.
8. **PR.** Open a pull request using the template, link the story, set the story status to
   `In Review`. Keep the PR small — if it's growing past ~400 changed lines, split the story.
9. **Merge & loop.** After merge, set the story to `Done` and go back to step 1.

The loop is the product cadence: small branch, green gate, merged, repeat. Don't batch unrelated
changes. Don't leave a branch half-done to start another.

## Definition of Done

A story is done only when **all** of these hold:

- Acceptance criteria in the story are met, demonstrably.
- New code has tests. Bug fixes have a regression test that fails before and passes after.
- Non-IaC code meets **≥ 90% unit-test coverage on business logic** (handlers wired through services,
  parsers, validators, state machines — not `main`, generated sqlc, or thin config/DI glue). IaC
  (`deploy/`, Dockerfiles, migrations, CI YAML) is excluded. See [`docs/10-agent-loop.md`](./docs/10-agent-loop.md).
- Application changes pass `make lint type test`; documentation-only changes pass the checks above.
  No skipped tests without a reason in the PR.
- Public API changes are reflected in `docs/04-api-spec.md` and the OpenAPI schema.
- DB changes ship with a golang-migrate migration (up **and** down), named after the story.
- Secrets are read from config/env, never committed. No credentials in code, logs, or fixtures.
- User-facing strings follow the writing rules below and ship in **both** `en` and `ru` locale files
  (see [`docs/11-localization.md`](./docs/11-localization.md)).
- New web UI matches the design system ([`docs/12-design-system.md`](./docs/12-design-system.md),
  visual reference [`docs/design_system.html`](./docs/design_system.html)).
- New shared UI components include a Storybook story (`apps/web`, REQ-DS-04).
- The PR description says what changed and how it was verified.

## Conventions

- **Language/stack:** Go 1.25+ (`apps/api`, `apps/worker`, `pkg`), React + TS + Vite (`apps/web`),
  PostgreSQL 16 only (no Redis). Don't introduce a new framework or language without an ADR
  (`docs/` + a story).
- **Go style:** Gin HTTP layer, sqlc for queries, golangci-lint. Layout: `cmd/`, `internal/handler/`,
  `internal/service/`, `internal/repository/`, `pkg/`.
- **TS style:** `eslint` + `prettier`. Functional components, TanStack Query, `react-i18next` for UI copy.
- **UI design:** Follow [`docs/12-design-system.md`](./docs/12-design-system.md) and
  [`docs/design_system.html`](./docs/design_system.html). Reuse shared components; map tokens in
  Tailwind — no ad-hoc colors or one-off button styles in feature code. New shared components ship
  with a Storybook story (REQ-DS-04).
- **Tests:** `go test` (API/worker), `vitest` + React Testing Library (web). Integration tests for
  every external connector use recorded fixtures, never live calls in CI. `make test` enforces
  ≥ 90% coverage on business-logic packages (Go: `-coverprofile`; web: Vitest thresholds).
- **Commits:** Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`).
- **Migrations:** golang-migrate in `db/migrations/`, with matching up/down files. Check existing
  versions on fresh main before assigning the next number. The production image migrates at startup.
- **Config:** PostgreSQL is authoritative for app settings. Env retains database connection,
  installation token and local development safeguards. Existing installs require the explicit
  config-import tool before updated services start. Configure app settings and bots through
  Settings; only listener changes require restart. Workspace slots stay on workspace pages.
- **Auth:** OIDC (Google, Slack, eXpress), plus opt-in localhost dev sign-in. No local passwords.
  Account paging authorization is separate from sign-in; preserve existing connections on failure.
- **Chat delivery:** shared lifecycle posts and chat acknowledgements use durable outboxes processed
  by the worker. A callback receipt isn't an acknowledgement result. See
  [`docs/features/incident-chat-notifications.md`](./docs/features/incident-chat-notifications.md).
- **Errors are interface:** API errors return a structured body `{code, message, details}`. UI errors
  say what happened and how to fix it — never a bare stack trace, never "Something went wrong."

## Guardrails (hard stops)

- **Preserve requirements.** `docs/` and `backlog/` are the spec. Keep the README, operational guides,
  and connector docs current when behavior changes. Don't silently rewrite product requirements
  to match code; identify proposed requirement changes in the PR.
- **No live credentials in tests or CI.** Ever.
- **No new third-party network calls** from the request path without it being in the story.
- **Don't weaken security** to make a test pass (no disabling auth, no `verify=False`, no wildcard
  CORS in committed code).
- **Stay in scope.** Build the story in front of you. Note adjacent problems in the backlog; don't
  fix them in this PR.

## Writing (UI copy & docs)

Plain, active, sentence case. Name things by what the user controls, not how the system is built — a
person manages *notifications*, not *webhook configs*. Buttons say what happens (`Acknowledge`,
`Assign to on-call`), and the resulting toast uses the same word (`Acknowledged`). Empty states tell
the user what to do next. No emoji in product copy. Errors don't apologise and are never vague.

**Localization:** English is the source language; every new UI or chat-template string needs matching
keys in `apps/web/src/locales/en/` and `apps/web/src/locales/ru/` (and `pkg/i18n/messages/` for
worker templates). Russian is a faithful translation, not a rewrite.

**User guides:** update the affected English and Russian how-tos and their adjacent HTML copies
together. Distinguish personal pages, team Slack channels, and the global eXpress on-call group.
Link Account instructions to [`docs/features/paging-connections.md`](./docs/features/paging-connections.md).

## When you're unsure

Prefer the smallest change that satisfies the story. If the story is ambiguous, write down your
interpretation in the PR and proceed — don't stall. If two requirements conflict, the PRD
(`docs/01-prd.md`) wins over a story; a story wins over your assumption.
