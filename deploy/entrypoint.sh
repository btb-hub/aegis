#!/usr/bin/env bash
set -euo pipefail

API_BIN="${AEGIS_API_BIN:-/app/api}"
WORKER_BIN="${AEGIS_WORKER_BIN:-/app/worker}"
NGINX_BIN="${AEGIS_NGINX_BIN:-nginx}"
NGINX_CONF="${AEGIS_NGINX_CONF:-/etc/nginx/conf.d/default.conf}"
MIGRATIONS_PATH="${AEGIS_MIGRATIONS_PATH:-/migrations}"

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "entrypoint: DATABASE_URL is required" >&2
  exit 1
fi

echo "entrypoint: running migrations"
migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" up

export HTTP_ADDR="127.0.0.1:8080"

pids=()

cleanup() {
  local pid
  for pid in "${pids[@]:-}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done
  wait || true
}
trap cleanup EXIT
trap 'exit 143' TERM INT

echo "entrypoint: starting api on ${HTTP_ADDR}"
"$API_BIN" &
pids+=("$!")

echo "entrypoint: starting worker"
"$WORKER_BIN" &
pids+=("$!")

echo "entrypoint: starting nginx"
if [[ "$NGINX_BIN" == "nginx" ]]; then
  nginx -c /etc/nginx/nginx.conf -g "daemon off;" &
else
  # Host test stub: ignore conf, just run the fake binary
  "$NGINX_BIN" -c "$NGINX_CONF" &
fi
pids+=("$!")

# Wait for any child to exit; then fail
set +e
while true; do
  for pid in "${pids[@]}"; do
    if ! kill -0 "$pid" 2>/dev/null; then
      wait "$pid"
      status=$?
      echo "entrypoint: child pid=$pid exited status=$status" >&2
      exit 1
    fi
  done
  sleep 1
done
