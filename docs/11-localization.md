# Localization (i18n)

MVP supports **English** (`en`, default) and **Russian** (`ru`). All user-facing product copy is
translated in both locales before a story ships.

## Scope

| Surface | Localized? | Notes |
|---------|------------|-------|
| Web UI | Yes | Labels, buttons, toasts, empty states, errors shown in the browser |
| Chat pages (Slack, eXpress) | Yes | Templates use the **recipient's** saved locale |
| API error `code` | No | Stable machine identifier (`VALIDATION_ERROR`, etc.) |
| API error `message` | English only | UI maps `code` (+ optional `details`) to localized strings |
| Audit log, metrics, logs | English | Operator/debug text |
| Jira ticket title/body | English | External system; alert text is source data |
| Setup wizard | Yes | Same rules as web UI |
| CSV export column headers | Yes | Uses exporter's active locale |

**Not in MVP:** additional locales, RTL layout, per-team default locale, crowd-sourced translation
workflow.

## Requirements

See PRD `REQ-I18N-*` in [`01-prd.md`](./01-prd.md).

## Locale resolution (web)

1. If the user is signed in, use `users.locale`.
2. Else if `localStorage` has `aegis_locale` (set by the language switcher), use it.
3. Else parse `navigator.languages` / `Accept-Language`; first match among `en`, `ru`.
4. Fallback: `en`.

Changing locale in the UI updates `localStorage` immediately. Signed-in users also call
`PATCH /auth/me` so the preference survives across devices.

## Web app (`apps/web`)

**Library:** `react-i18next` + `i18next`.

**Layout:**

```text
apps/web/src/locales/
  en/
    common.json       # shared: nav, actions, errors, dates
    shifts.json
    incidents.json
    alerts.json
    settings.json
    wizard.json
  ru/
    common.json
    shifts.json
    ...
```

- Keys are dot-paths in JSON (`incidents.acknowledge`, `common.save`).
- **No hard-coded user-facing strings** in TSX except developer-only labels.
- Namespace per feature area; load lazily per route where practical.
- Pluralisation and interpolation via i18next (`{{count}}`, `_plural` keys).

**Formatting:**

- Dates/times: `Intl.DateTimeFormat` with active locale; always show timezone (user preference
  field is post-MVP — default to browser timezone).
- Numbers: `Intl.NumberFormat`.
- Relative time ("5 minutes ago"): `Intl.RelativeTimeFormat` or a small helper wired to the active
  locale.

**Language switcher:**

- Persistent control in the app shell (header or user menu): **English** / **Русский**.
- Switching does not reload the page; `i18n.changeLanguage` + persist.

**Tests:**

- Vitest: render a component with `en` and `ru` providers; assert key strings differ (smoke, not
  full copy review).
- CI: script or test that every key in `en/**/*.json` exists in `ru/**/*.json` (and vice versa for
  leaf keys).

## Backend (`apps/api`, `apps/worker`)

Shared message catalog for outbound chat templates:

```text
pkg/i18n/
  i18n.go           # T(locale, key, vars...) string
  messages/
    en.json
    ru.json
```

- Worker loads recipient `users.locale` when building Slack Block Kit / eXpress bubbles.
- Unknown locale → `en`.
- Same key naming discipline as web where messages mirror UI (`page.acknowledge_button`).

API handlers do **not** localise `message` in error JSON; the web client translates by `code`.

## Data model

`users.locale` — `text`, not null, default `'en'`, check `locale IN ('en', 'ru')`.

Added in the initial `users` migration (see [`03-data-model.md`](./03-data-model.md)).

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/auth/me` | Includes `locale` |
| PATCH | `/auth/me` | Body `{ "locale": "en" \| "ru" }` |

Invalid locale → `400` with `code: INVALID_LOCALE`.

## Copy rules (both locales)

Follow [`CLAUDE.md`](../CLAUDE.md) writing rules in **English** first; Russian is a faithful
translation, not a rewrite.

- Buttons name the action (`Acknowledge` / `Подтвердить`, not "OK").
- Toasts reuse the same verb as the button.
- Errors state what happened and what to do next.
- No emoji in product copy.
- Sentence case in English; Russian follows typographic norms for the locale.

**English is the source of truth** for new keys: add `en` string, then `ru` in the same PR.

## Agent workflow

1. Any story that adds UI or chat template strings **must** add keys to both locale files.
2. PR checklist: both `en` and `ru` updated; missing-key test green.
3. Do not land a feature with English-only UI.

## References

- PRD: [`01-prd.md`](./01-prd.md) — `REQ-I18N-*`
- Architecture: [`02-architecture.md`](./02-architecture.md)
- API: [`04-api-spec.md`](./04-api-spec.md)
- Design system (pickers): [`12-design-system.md`](./12-design-system.md), [`design_system.html`](./design_system.html)
- Story: [AEG-054](../backlog/epics/EPIC-01-foundation.md) in EPIC-01
