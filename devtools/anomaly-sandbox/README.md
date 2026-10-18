# Local Aegis and Mimir sandbox

This runs a separate local stack with synthetic data. It includes the actual Aegis API/worker/UI, private anomaly module, PostgreSQL with separate module credentials, Mimir 3.0.0, Prometheus 3.5.0 and a small control/exporter/chat-sink service. All published ports bind to loopback. It uses its own Compose project and named volumes.

You need Docker Desktop and a checkout of `git@github.com:btb-hub/aegis-anomaly-detector.git`. The script finds `.worktrees/aegis-anomaly-detector` or a sibling checkout automatically. It does not change your normal `.env` or running application containers. The fixed credentials and dev login are for this sandbox only.

From the Aegis root in PowerShell:

```powershell
./scripts/anomaly-sandbox.ps1 up
# Or select a private checkout explicitly:
./scripts/anomaly-sandbox.ps1 up -ModulePath C:/path/to/aegis-anomaly-detector
```

On other shells, set `AEGIS_ANOMALY_SOURCE_DIR` to the absolute private checkout path and use `docker compose -f devtools/anomaly-sandbox/compose.yml up -d --build`, then `docker compose -f devtools/anomaly-sandbox/compose.yml wait bootstrap`.

Open [controls](http://localhost:18081), then **Sign in to Aegis**. [Aegis](http://127.0.0.1:18080) has a Local sandbox team, routing for `team=platform`, Local Dev User on call, local Slack/eXpress connectors and an enabled interval metric. Use the `127.0.0.1` Aegis link to keep its session cookie separate from the usual `localhost` development app. Bootstrap records resource IDs in its own volume, preserving renamed resources and edited connector settings when you restart. If module access is paused, startup succeeds and leaves retained module data untouched.

1. Wait for startup/bootstrap to finish. Prometheus scrapes two series every five seconds and sends them to a real tenant-isolated Mimir through remote write.
2. Click **Trigger spike**. `api-1` rises from about 10 to 35; `api-2` stays at 10. The detector reads Mimir, records the anomaly and delivers an Aegis alert. Allow roughly 15–40 seconds.
3. Open Incidents and acknowledge the incident. Captured Slack/eXpress payloads appear on the controls page, without real chat delivery.
4. Click **Restore healthy values**. Within roughly 30–60 seconds, the alert resolves and the incident gets an Anomaly recovered entry. The incident remains acknowledged until you close it manually.
5. **Trigger dip** exercises the lower bound. **Send ordinary alert** exercises the normal webhook/routing path without anomaly detection.
6. In the module, preview/chart both series and submit feedback. Discovery creates two per-series metrics with delivery off. Disabled AR, S-H-ESD and intersection examples are available for previews after at least 2–3 minutes of warmup. These live samples are for interaction testing; numerical correctness is covered separately by the 197 golden cases.

Resolve a recovered incident manually before starting another scenario if you want a fresh incident and page. Otherwise Aegis can group the next alert into the existing open/acknowledged incident. The exporter is scraped every five seconds; module range queries use ten-second steps so a one-day observation window stays below Mimir's 11,000-point per-series API limit.

The disabled statistical preview examples use `last_over_time(anomaly_demo_value{service="checkout"}[1m])` to bridge short scrape gaps during sandbox restarts. The enabled interval metric reads the raw query. Without that explicit PromQL choice, statistical previews correctly reject irregular/missing samples instead of silently filling them.

| Service | URL or configuration |
| --- | --- |
| Controls and captured messages | http://localhost:18081 |
| Aegis | http://127.0.0.1:18080 |
| Prometheus query UI | http://localhost:19090/query |
| Mimir on host | http://localhost:19009/prometheus |
| Mimir source inside module | `http://mimir:9009`, API prefix `/prometheus` |
| Mimir tenant | `local-sandbox` in `X-Scope-OrgID` |
| Query | `anomaly_demo_value{service="checkout"}` |
| Exporter | http://localhost:18081/metrics |

The control script also accepts `spike`, `dip`, `healthy`, `alert`, `status` and `logs`. `down` stops the stack and retains configuration, samples, history and captured messages. `up` resumes it. If ports 18080, 18081, 19009 or 19090 are occupied, stop that conflicting sandbox before starting this one. Don't change ports without also updating PUBLIC_URL and the control page links/origin configuration.

The existing alert simulator still works with this stack:

```powershell
docker compose -f devtools/anomaly-sandbox/compose.yml run --rm --build alert-simulator -once -team platform
```

To query Mimir directly from PowerShell:

```powershell
Invoke-RestMethod -Headers @{'X-Scope-OrgID'='local-sandbox'} -Uri 'http://localhost:19009/prometheus/api/v1/query?query=anomaly_demo_value'
```

For source failure/suppression testing, stop only `mimir` using this Compose file, inspect module Health, then start it again. For restart recovery, restart `anomaly-detector`; PostgreSQL retains state. Pausing module access in Aegis Settings retains read-only history and stops new detection/delivery. Ordinary accepted host alerts continue through the host workflow.

Mimir uses its documented single-process filesystem/classic configuration for local development, without Kafka. See [Mimir getting started](https://grafana.com/docs/mimir/latest/get-started/) and [Prometheus remote-write configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write). This is a development environment, not a production deployment recipe. The on-call override lasts 30 days and is refreshed when the seed job reruns.
