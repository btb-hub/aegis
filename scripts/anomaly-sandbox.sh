#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
compose=(docker compose -f "$repo_root/devtools/anomaly-sandbox/compose.yml")
action=${1:-up}

case "$action" in
    up)
        module_path=${AEGIS_ANOMALY_SOURCE_DIR:-}
        if [[ -z "$module_path" ]]; then
            for candidate in "$repo_root/.worktrees/aegis-anomaly-detector" "$repo_root/../aegis-anomaly-detector"; do
                if [[ -f "$candidate/go.mod" && -f "$candidate/Dockerfile" ]]; then
                    module_path=$candidate
                    break
                fi
            done
        fi
        if [[ -z "$module_path" ]]; then
            module_path="$repo_root/.worktrees/aegis-anomaly-detector"
            mkdir -p -- "$repo_root/.worktrees"
            printf 'Cloning the private module (GitHub SSH access required)...\n'
            git clone git@github.com:btb-hub/aegis-anomaly-detector.git "$module_path"
        fi
        if [[ ! -f "$module_path/go.mod" || ! -f "$module_path/Dockerfile" ]]; then
            printf 'Invalid private module checkout: %s\nSet AEGIS_ANOMALY_SOURCE_DIR to its checkout directory.\n' "$module_path" >&2
            exit 1
        fi
        export AEGIS_ANOMALY_SOURCE_DIR
        AEGIS_ANOMALY_SOURCE_DIR=$(cd -- "$module_path" && pwd)
        "${compose[@]}" up -d --build
        if ! "${compose[@]}" wait bootstrap; then
            printf 'Bootstrap failed. Run make anomaly-sandbox-logs for details.\n' >&2
            exit 1
        fi
        printf '\nControls http://localhost:18081 | Aegis http://127.0.0.1:18080\n'
        ;;
    down) "${compose[@]}" down ;;
    status) "${compose[@]}" ps -a ;;
    logs) "${compose[@]}" logs --tail 100 ;;
    *) printf 'Usage: %s [up|down|status|logs]\n' "$0" >&2; exit 2 ;;
esac
