# Agent development loop

Full detail for the contract in [`CLAUDE.md`](../CLAUDE.md).

## Loop

1. **Sync** — `git checkout main && git pull`
2. **Pick** — explicit user request, or highest-priority `Ready` story in [`backlog/epics/`](../backlog/epics/) whose dependencies are `Done`
3. **Branch** — `codex/<short-slug>` off fresh `main`, unless the user specifies another name
4. **Plan** — 3–8 lines in story file or PR: files, approach, test plan
5. **Build** — vertical slice; tests with code
6. **Verify** — `make lint type test` for application changes; documentation checks below for docs-only requests
7. **Self-review** — acceptance criteria + Definition of Done
8. **PR** — template, link story, set story `In Review`
9. **Merge** — set story `Done`, loop

## Gate commands

```bash
make lint    # go vet, eslint, Storybook build
make type    # Go builds, tsc --noEmit
make test    # Go coverage gates, locale keys, Vitest coverage
```

Or combined: `make lint type test`.

For documentation-only changes, verify local links and anchors, compare instructions with current
source/configuration, keep affected EN/RU Markdown and HTML guides aligned, and run
`git diff --check`. Record the checks in the PR; no application tests are needed when only docs change.

## Test coverage (NFR-5)

All **non-IaC application code** must keep **≥ 90% unit-test coverage on business logic**:

| Included | Excluded |
|----------|----------|
| Service layer, domain parsers, validators, state transitions | `main`, generated sqlc, `deploy/`, Dockerfiles |
| HTTP handlers tested through services (httptest) | Raw migration SQL, GitHub Actions YAML |
| React components, hooks, formatters, i18n helpers | `vite.config`, ESLint config, locale JSON copy |

Coverage must reflect real behaviour (table-driven cases, error paths), not assertion-free mocks.
`make test` fails below threshold. CI runs the same gate.

## Story statuses

| Status | Meaning |
|--------|---------|
| `Ready` | Can be picked |
| `Blocked` | Waiting on dependency |
| `In Review` | PR open |
| `Done` | Merged |

Only **one** story should be `Ready` at a time unless explicitly parallelized by a human.

## When to split a story

- PR approaches ~400 lines changed
- Two unrelated concerns (e.g. API + unrelated UI polish)
- Migrations + feature can ship separately only if feature is behind flag

## UI stories

Before building or changing web UI:

1. Read [`12-design-system.md`](./12-design-system.md) for tokens and component rules.
2. Open [`design_system.html`](./design_system.html) in a browser for the visual spec.
3. Run Storybook (`cd apps/web && npm run storybook`) for component states.
4. Reuse components from `apps/web/src/components/ui/`; extend the library + Storybook if a pattern is missing.

## Definition of Done

- Acceptance criteria met with tests
- Application gate or documentation checks above pass; limitations recorded in the PR
- API changes in [`04-api-spec.md`](./04-api-spec.md)
- DB changes: golang-migrate up **and** down
- No secrets committed
- User-facing copy in both `en` and `ru` per [`11-localization.md`](./11-localization.md)
- Business-logic coverage ≥ 90% per NFR-5 above
- Web UI matches [`12-design-system.md`](./12-design-system.md) / [`design_system.html`](./design_system.html)
- PR describes verification steps

## Spec hierarchy

1. [`01-prd.md`](./01-prd.md) wins over stories
2. Stories win over agent assumptions
3. Do not rewrite spec to match code — propose spec PR instead

## References

- Roadmap: [`../backlog/roadmap.md`](../backlog/roadmap.md)
- PR template: [`.github/pull_request_template.md`](../.github/pull_request_template.md)
- Design system: [`12-design-system.md`](./12-design-system.md), [`design_system.html`](./design_system.html)
