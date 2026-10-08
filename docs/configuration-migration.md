# Move production configuration into PostgreSQL

This is a manual rollout on the production host. Keep secrets out of the repository and chat.
The new API and worker never import application env values at startup or fall back to them.
An existing installation requires a completed import before updated services start, including
installations with saved teams, connectors, alerts, or workspaces but no users.

## Existing installations

1. Back up PostgreSQL, retain the original `.env` in protected storage, and record the current
   image version. Pause API/worker writers during cutover.
2. Apply all migrations, including `000022`, using your usual migration tool. The all-in-one
   image can run migrations without starting services. `DATABASE_URL` below must already be
   available in the operator's environment; don't execute or `source` the env file:

   ```bash
   docker run --rm --entrypoint /usr/local/bin/migrate ghcr.io/btb-hub/aegis:vNEW \
     -path /migrations -database "$DATABASE_URL" up
   ```

3. Validate the actual production file, mounted read-only:

   ```bash
   docker run --rm \
     -v /secure/original.env:/run/aegis-original.env:ro \
     --entrypoint /app/config-import ghcr.io/btb-hub/aegis:vNEW \
     --env-file /run/aegis-original.env --dry-run
   ```

   Output lists key names and statuses only: `importable`, `already stored`, `deployment-only`,
   `incomplete`, or `unsupported`. Values and connection strings never appear. Invalid settings
   report the offending key name and block import. Unsupported keys also block it. Explicitly
   account for each unknown key: add missing application keys to the code mapping, or retain
   unrelated proxy/tooling variables in a separate protected deployment file and remove them
   from the reviewed import input. Re-run dry-run against that reviewed file. Never silently
   drop an unknown application setting. The parser accepts quotes/comments/export prefixes;
   it doesn't expand variables or execute shell expressions.
4. Run the same command with `--apply` instead of `--dry-run`. Settings, connector updates,
   audit field names, and the installation/import marker commit together under a database lock.
   Concurrent imports serialize; repeated runs make no further changes.
5. Deploy API and worker. PostgreSQL is now authoritative, including disabled providers and
   removed values, even while stale app variables remain in the environment.
6. Verify Google/Slack/eXpress login, existing sessions, webhook intake with the existing secret,
   routing, paging, scheduled escalations, and Settings in both languages. For eXpress, enable
   verified SSO in **Settings → Sign-in and access**, complete and enable the global BotX
   connector in **Settings → Integrations**, then bind **Account → Paging messengers**.
7. Only after verification, remove migrated variables from the live deployment. Retain the
   original env securely for rollback. Keep `DATABASE_URL` and separate deployment/tooling
   variables. Existing installations don't need `AEGIS_BOOTSTRAP_TOKEN`.

Native alternative: build `go build -o config-import ./apps/api/cmd/config-import` from the
repository root, then run `./config-import --env-file /secure/original.env --dry-run` and
`--apply`. Process `DATABASE_URL` overrides the file's database connection. The API-only image
also includes `/app/config-import`; use your normal migration tool with that image.

## Explicit key mapping

| Original keys | Database destination |
| --- | --- |
| `GOOGLE_OIDC_CLIENT_ID`, `GOOGLE_OIDC_CLIENT_SECRET`, `GOOGLE_OIDC_REDIRECT_URL` | Google sign-in |
| `SLACK_OIDC_CLIENT_ID`, `SLACK_OIDC_CLIENT_SECRET`, `SLACK_OIDC_REDIRECT_URL` | Slack sign-in |
| `EXPRESS_OIDC_ISSUER`, `EXPRESS_OIDC_CLIENT_ID`, `EXPRESS_OIDC_CLIENT_SECRET`, `EXPRESS_OIDC_REDIRECT_URL` | eXpress sign-in |
| `ADMIN_EMAILS` | Sign-in and access |
| `SESSION_TTL`, `ALERT_FINGERPRINT_LABELS`, `INCIDENT_DEDUP_WINDOW`, `ESCALATION_TIMEOUT` | App behavior |
| `JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN`, `JIRA_PROJECT_KEY` | Existing global Jira integration table |
| `SLACK_BOT_TOKEN`, `SLACK_SIGNING_SECRET` | Existing global Slack integration table |
| `EXPRESS_BOT_ID`, `EXPRESS_BOT_HOST`, `EXPRESS_BOT_SECRET` | Existing global eXpress integration table |
| `PUBLIC_URL`, `HTTP_ADDR`, `WEBHOOK_SECRET` | Deployment settings |
| `SESSION_SECRET` | Compatibility storage, no runtime consumer or editable control |
| `DATABASE_URL`, `AEGIS_BOOTSTRAP_TOKEN`, `DEV_AUTH_ENABLED`, `DEV_AUTH_DEFAULT_ROLE`, `DEV_AUTH_EMAIL` | Deployment only |
| `SEED_DEV`, `AEGIS_API_BIN`, `AEGIS_WORKER_BIN`, `AEGIS_NGINX_BIN`, `AEGIS_NGINX_CONF`, `AEGIS_MIGRATIONS_PATH` | Tooling only |
| `AEGIS_API_URL`, `AEGIS_WEBHOOK_URL`, `ALERT_SIM_INTERVAL`, `ALERT_SIM_TEAM`, `ALERT_SIM_PROJECT` | Simulator tooling only |

Existing database fields win. Missing connector fields are filled without overwriting saved
credentials or changing existing enabled flags. New incomplete connectors are disabled and
visible for completion in Settings. Complete imported providers remain active; partial
provider values stay stored but disabled. Imported callbacks are preserved exactly. New
provider drafts default to the saved public URL plus `/auth/{provider}/callback`.
`deploy/.env.legacy.example` documents the old file format; don't use it for new deployments.

## Fresh installation

Deploy with `DATABASE_URL` and `AEGIS_BOOTSTRAP_TOKEN` generated from at least 32 random bytes
(for example `openssl rand -hex 32`). Open `/bootstrap` and submit the token through the form.
It grants one browser a 30-minute session limited to authentication setup. Enter the first
admin email, save provider credentials, register the displayed callback externally, and use
**Test sign-in** with that verified email. Provider activation, first admin/session creation,
and permanent bootstrap closure happen atomically. Existing installations can't use this
flow. Bootstrap can't reopen after completion, even if users are removed. Remove the token
from deployment after installation.

## Settings changes and rollback

Settings use revision checks. Reload after conflicting edits. Blank secret inputs retain
stored values. Provider credentials stay in drafts until a successful verified test by the
same admin browser session. Failed tests leave active credentials untouched. The last active
provider can't be disabled. Admin email pinning and last-admin protection still apply.

Changes affect subsequent requests/jobs. Existing session expiries, stored alert fingerprints,
and scheduled escalation timestamps aren't rewritten. Changing `HTTP_ADDR` requires an API
restart. The all-in-one nginx upstream expects port 8080; changing its port also requires
updating nginx. Changing `PUBLIC_URL` requires callback confirmation and invalidates pending
login, test, and paging attempts. Callbacks matching the old default follow the new URL;
custom callbacks remain explicit and must be reviewed.

For a binary rollback, restore the previous image and original env file. Leave the additive
database tables in place. Don't run the down migration during binary rollback: it deletes
settings, drafts, authorizations, and permanent installation state. Reconcile any settings
changed after cutover before returning to env-based binaries. Restore a backup only under
your usual data-recovery procedure.
