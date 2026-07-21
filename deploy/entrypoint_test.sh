#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Fake migrate + children that record invocations
mkdir -p "$TMP/bin" "$TMP/app"
cat >"$TMP/bin/migrate" <<'EOF'
#!/usr/bin/env bash
echo "migrate $*" >>"${FAKE_LOG}"
if [[ "${MIGRATE_FAIL:-0}" == "1" ]]; then exit 1; fi
exit 0
EOF
chmod +x "$TMP/bin/migrate"

for name in api worker nginx; do
  cat >"$TMP/app/$name" <<EOF
#!/usr/bin/env bash
echo "$name start HTTP_ADDR=\${HTTP_ADDR-} args=\$*" >>"\${FAKE_LOG}"
# Stay alive until killed unless DIE_AFTER is set
if [[ -n "\${DIE_AFTER:-}" && "$name" == "\${DIE_NAME:-api}" ]]; then
  sleep "\$DIE_AFTER"
  exit 1
fi
while true; do sleep 60; done
EOF
  chmod +x "$TMP/app/$name"
done

# Patch entrypoint paths for host test by wrapping
cat >"$TMP/run-entrypoint" <<EOF
#!/usr/bin/env bash
set -euo pipefail
export PATH="$TMP/bin:\$PATH"
export FAKE_LOG="$TMP/log"
# Rewrite absolute paths used by entrypoint via env overrides the real script supports
export AEGIS_API_BIN="$TMP/app/api"
export AEGIS_WORKER_BIN="$TMP/app/worker"
export AEGIS_NGINX_BIN="$TMP/app/nginx"
export AEGIS_NGINX_CONF="/dev/null"
export AEGIS_MIGRATIONS_PATH="/migrations"
exec bash "$ROOT/deploy/entrypoint.sh"
EOF
chmod +x "$TMP/run-entrypoint"

# Test 1: missing DATABASE_URL
set +e
env -u DATABASE_URL FAKE_LOG="$TMP/log" "$TMP/run-entrypoint" >"$TMP/out" 2>&1
rc=$?
set -e
if [[ $rc -eq 0 ]]; then
  echo "FAIL: expected non-zero without DATABASE_URL"
  exit 1
fi
grep -qi "DATABASE_URL" "$TMP/out"

# Test 2: migrate failure does not start children
: >"$TMP/log"
set +e
MIGRATE_FAIL=1 DATABASE_URL="postgres://x" FAKE_LOG="$TMP/log" "$TMP/run-entrypoint" >"$TMP/out" 2>&1
rc=$?
set -e
if [[ $rc -eq 0 ]]; then
  echo "FAIL: expected non-zero when migrate fails"
  exit 1
fi
grep -q "migrate " "$TMP/log"
if grep -q "api start" "$TMP/log"; then
  echo "FAIL: api started despite migrate failure"
  exit 1
fi

echo "entrypoint_test.sh OK"
