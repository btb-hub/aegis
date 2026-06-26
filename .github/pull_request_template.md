## Story

Link: <!-- e.g. AEG-001 in backlog/epics/EPIC-01-foundation.md -->

## Summary

<!-- What changed and why (1–3 sentences) -->

## Test plan

- [ ] `make lint type test` green locally
- [ ] <!-- story-specific verification steps -->

## Definition of Done

- [ ] Acceptance criteria in the story are met
- [ ] New code has tests (regression test for bug fixes)
- [ ] Business-logic unit-test coverage ≥ 90% (`make test`; NFR-5)
- [ ] Public API changes reflected in `docs/04-api-spec.md`
- [ ] DB changes include golang-migrate up **and** down
- [ ] No secrets in code, logs, or fixtures
- [ ] User-facing copy follows CLAUDE.md writing rules
- [ ] UI strings added in both `en` and `ru` locale files (see `docs/11-localization.md`)
- [ ] Web UI matches design system (`docs/12-design-system.md`, visual: `docs/design_system.html`)
- [ ] New shared UI components include a Storybook story (REQ-DS-04)
