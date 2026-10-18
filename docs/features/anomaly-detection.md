# Optional anomaly detection module

Aegis can connect to the separately installed private `btb-hub/aegis-anomaly-detector` service.
Core builds, starts and runs without access to its repository. The module owns detection,
sources, discovery, history, charts, feedback and exports. Core supplies identity, workspace
organization, routing, incidents and existing Slack/eXpress delivery.

## Configure a connection

Install a released module image with its own PostgreSQL database and credentials. A separate
database may share the Aegis PostgreSQL server. The module applies its embedded migrations
at startup. Back up that database separately before upgrades. Keep the service reachable only
from the Aegis API and trusted operators.

Open **Settings → Anomaly detection** as an admin and save these fields together:

| Stored setting | Meaning |
| --- | --- |
| `AEGIS_ANOMALY_URL` | Module origin, such as `http://anomaly-detector:8080`, without a path, query or user credentials |
| `AEGIS_INSTALLATION_ID` | Stable installation identifier, 1–128 characters; configure the same value in the module |
| `AEGIS_ANOMALY_IDENTITY_SECRET` | At least 32 random bytes; shared only with this module |
| `AEGIS_ANOMALY_SERVICE_TOKEN` | Different credential containing at least 32 random bytes; module service authentication |
| `AEGIS_ANOMALY_ENABLED` | `true` grants detection and delivery; `false` keeps retained information read-only |

The module uses the same identity/install credentials, plus its `DATABASE_URL`, `AEGIS_URL`,
`ENCRYPTION_KEY` and `MODULE_ENABLED`. `AEGIS_URL` must reach the host API, for example
`http://api:8080`. Use the private module release guide for the full deployment configuration.
Host application settings live in PostgreSQL and take effect on subsequent requests. They
aren't loaded from env during ordinary startup. An existing dotenv deployment can add these
keys to its explicit config-import input; dry-run reports and settings responses redact secrets.
Blank secret inputs preserve saved values. Do not rotate the installation ID after recording data.
Rotate shared credentials on both services together, then use **Check saved connection**.

The optional deployment examples in `deploy/docker-compose.anomaly.yml` and
`deploy/anomaly-module.k8s.yaml` require a published private image and provisioned credentials.
They don't clone or build private source. They are opt-in, so core startup doesn't depend on the module.

## Authentication and entitlement

`/modules/anomaly-detector/` is an authenticated reverse proxy that preserves its URL prefix,
including assets and direct links. It strips browser credentials, forwarded identity headers and
cookies before adding a 60-second HS256 identity JWT in `X-Aegis-Identity`. Claims are issuer
`aegis`, audience `anomaly-detector`, user ID, installation, role, locale, accessible workspace IDs,
entitlement, issued-at and expiry. Existing Aegis roles have global workspace visibility; this
module doesn't invent additional membership rules.

Admins configure sources and metrics. Members can submit feedback at
`POST /modules/anomaly-detector/api/v1/history/{snapshot_id}/feedback`; viewers read only.
Every browser mutation also requires Aegis's saved origin. Disabling access blocks mutations
and signs retained read access with `entitled=false`.
Events already accepted by Aegis continue through its existing incident and notification workflow;
disabling module access doesn't cancel pending pages or escalation for those incidents.

The service-authenticated host entitlement endpoint returns a grant valid for 60 seconds,
even when disabled. The private service refreshes it and fails closed when no current grant
is available. A replaceable `EntitlementProvider` currently derives access from explicit config;
there is no billing integration. The host authenticates upstream capability requests with the
separate service bearer token. Contract mismatch, bad health or timeout shows a useful fallback
without blocking other Aegis pages. It doesn't require upstream entitlement to bootstrap a grant.
Capability checks have a three-second limit. Proxied requests have a ten-second limit,
except source connection tests and detection previews, which allow 150 seconds for the
module's bounded source queries and detection. Browser/machine mutation bodies are limited to 1 MiB.

## Alerts and recovery

Delivery is opt-in per metric in the module. Events enter
`POST /api/v1/modules/anomaly-detector/events` using the service bearer token, not a browser
session or the generic alert webhook. See the [OpenAPI contract](../openapi-anomaly-module.yaml).

Event IDs are deduplicated per installation using the hash of their canonical decoded payload.
An identical replay returns the original alert; reusing an ID with changed content returns `409`.
Each episode creates at most one alert and processing job. Increasing firing updates reuse it.
The SHA256 fingerprint includes installation, workspace, metric and series, and excludes the
episode and measured value. Reserved `aegis_*` identity labels replace caller-supplied values.
Routing rules must belong to the event's workspace; explicit cross-workspace team targets on
those rules retain their existing behavior. Separate episodes can link to an existing open
incident within the normal Aegis deduplication window.

Recovery resolves the original alert and appends `anomaly_recovered` to every linked incident's
timeline. It never closes or acknowledges an incident. Stale sequence events and delayed firing
events after recovery are retained as ignored receipts. A recovery received before initial firing
creates a resolved alert without an incident job. If recovery beats the pending worker, the worker
safely skips the already-cleared alert. If linking wins the race, recovery appends the timeline entry.
No timeline entry is needed when an incident never existed. Responders still close incidents manually.
Workspaces with retained anomaly episode history cannot be deleted; the API returns `409`.

## Operations

Back up the core database and module database separately, along with the module encryption key
and shared credentials in your secret manager. Upgrade using compatible contract-version-1
releases and apply core migration `000023_anomaly_module` before starting the updated API.
The migration has an up/down pair; rolling it down removes module receipts and episode mappings,
so back up first and stop ingress before a deliberate rollback. Existing alerts and incidents remain.

If connection checks fail, verify the saved module origin, network path, released image version,
service token and health. If screens are read-only, verify both the host access checkbox and the
module's current entitlement grant. Check workspace routing and source labels when an alert has
no incident. Never paste source credentials or service response bodies into tickets or logs.

Users can follow the [English guide](../how-to/anomaly-detection.md) or
[Russian guide](../how-to/ru/anomaly-detection.md).
