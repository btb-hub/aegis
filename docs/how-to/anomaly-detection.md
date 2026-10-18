# Anomaly detection

An administrator installs the separate module and its PostgreSQL database. Core Aegis remains usable without it.

## Connect the module

1. Open **Settings → Anomaly detection** as an admin.
2. Enter the module URL and stable installation ID. Use the same installation ID in the module.
3. Enter two different random secrets of at least 32 bytes: the identity signing secret and module service token. Match both values in the installed service.
4. Enable detection and alert delivery, then save all fields together. Blank secret fields keep saved credentials.
5. Choose **Check saved connection**. If it fails, check the service URL, network access, credentials and compatible release.

## Detect and investigate

Open **Anomaly detection** in the sidebar. Choose a workspace, connect a source, and configure a metric. Preview its history and detection parameters before enabling its schedule. Alert delivery is opt-in for each metric; detection results are retained even when delivery is off.

Admins configure sources and metrics. Members submit feedback on history snapshots. Viewers read history and charts. Prometheus and Mimir are distinct source types; a multitenant Mimir connection needs the configured tenant ID. Keep source credentials in the module configuration.

Configure Aegis routing rules in the metric's workspace so its labels match the intended team. Existing on-call routing and Slack/eXpress delivery then handle eligible events. Explicit cross-workspace team targets still follow the rule's configured behavior.

## Recovery and read-only access

Recovery resolves the original alert and adds an incident timeline entry. Responders close incidents manually. If the alarm clears before the worker processes it, no incident is created. Duplicate or reordered events cannot reopen a recovered episode.

Turn off **Enable detection and alert delivery** in Settings to pause new work and event delivery. Events already accepted by Aegis continue through its incident workflow, including pending notifications and escalation. Saved settings and history remain readable. If the module is offline, its page offers links back to Settings and Aegis; other pages still work.

See the [installation and operations guide](../features/anomaly-detection.md) for deployment, upgrades, credential rotation and backups.
