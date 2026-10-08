# Settings / Настройки

Admins manage application configuration at `/settings`. The setup wizard and its browser
progress storage are removed. `/setup` redirects here; `/integrations` redirects to
`/settings?section=integrations`, retaining configuration query parameters.

| Section | What to configure |
| --- | --- |
| Sign-in and access / Вход и доступ | Google, Slack, eXpress provider drafts; admin email rules |
| Integrations / Интеграции | Global Jira, Slack and eXpress bot settings; existing connection tests |
| App behavior / Поведение приложения | Session duration, deduplication window, escalation timeout, fingerprint labels |
| Deployment / Развёртывание | Public URL, listening address, webhook secret |
| Diagnostics / Диагностика | API/database health, provider status, test alert, connection review |

Teams, users, workspaces, overrides, and personal Account connections keep their existing pages.
New eXpress account errors distinguish missing authentication, missing bot configuration, and a
disabled bot. Admins receive links to the relevant Settings section.

Secret inputs are empty and show whether a secret exists. Blank means keep the stored value;
type a new value to replace it. `SESSION_SECRET` is retained solely for compatibility and isn't
editable. Writes validate values, record field names without values in the audit log, and
reject stale revisions. Browser mutations require the application's origin.

Save provider changes as a draft, register the displayed callback with the provider, and click
**Test sign-in / Проверить вход** using your verified admin email. Signed ID tokens must pass
issuer, audience, expiry, nonce and state verification. Tests are bound to your browser session,
provider, and draft/settings revisions. They expire in five minutes and can be used once.
Tests never create users or connect paging accounts. Activate only after a successful test;
failed tests leave current sign-in untouched. You can't disable the last active provider.

Each API request and worker job reads database settings. Changes apply to the next operation.
Existing session expiry dates, fingerprints, and scheduled escalation dates aren't rewritten.
Listener changes take effect after restarting the API. Public URL changes require confirmation
showing the resulting callbacks and cancel pending login, test and paging attempts. Imported
custom callback URLs remain explicit; callbacks matching the old default follow the new URL.

Fresh installations use the limited 30-minute token form at `/bootstrap`. The verified first
admin sign-in atomically activates the provider, creates the admin/session, and permanently
closes installation access. Existing installations must use the manual
[configuration import](../configuration-migration.md) instead.

## Русский

Откройте **Настройки** с ролью администратора. Сохраните параметры входа как черновик,
зарегистрируйте показанный URL обратного вызова у провайдера, затем нажмите **Проверить вход**
и войдите с подтверждённой почтой администратора. Только после успешной проверки активируйте
провайдера. Неудачная проверка не меняет текущие параметры входа.

Для привязки eXpress настройте SSO в **Вход и доступ**, заполните и включите глобального бота
в **Интеграции**, затем откройте **Аккаунт → Мессенджеры для пейджинга**. Команды, пользователи
и параметры рабочих областей остаются на своих страницах. Проверка состояния и тестовый алерт
доступны в **Диагностика**. Мастер настройки удалён.

Пустое поле секрета сохраняет существующее значение. Изменения применяются к следующим запросам
и заданиям; изменение адреса прослушивания требует перезапуска API. Для изменения публичного
URL нужно подтвердить новые адреса обратного вызова. Незавершённые авторизации будут отменены.
Перед обновлением существующей установки выполните резервное копирование и ручной импорт
производственного `.env` по [инструкции](../configuration-migration.md). После проверки удалите
перенесённые переменные из рабочего окружения, а оригинальный файл сохраните для отката.
