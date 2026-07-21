# All-in-one GHCR Image Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish one public multi-arch GHCR image (`ghcr.io/btb-hub/aegis`) that runs API + worker + web behind a single nginx port, migrates on start against external Postgres, for Kubernetes / `docker run` without Compose.

**Architecture:** Multi-stage `deploy/Dockerfile` builds Go api/worker and the web SPA, then packs them into `nginx:1.27-alpine` with `migrate`, migrations, and `entrypoint.sh`. Entrypoint migrates, starts api on `127.0.0.1:8080`, worker, and nginx on `:3000`; any child death exits the container. Tag pushes `v*` build/push via Buildx. Existing Compose + per-service Dockerfiles stay for local source builds.

**Tech Stack:** Docker Buildx, GHCR, nginx Alpine, golang-migrate v4.17.1, Go 1.25, Node 20, GitHub Actions.

**Spec:** [`docs/superpowers/specs/2026-07-21-all-in-one-ghcr-image-design.md`](../specs/2026-07-21-all-in-one-ghcr-image-design.md)

## Global Constraints

- Image name: `ghcr.io/btb-hub/aegis` (tags: git `v*` + `latest`).
- Platforms: `linux/amd64` and `linux/arm64`.
- Public pulls; publish only on `v*` tags (not every `main` push).
- One container process tree: api + worker + nginx; external Postgres only.
- Auto-migrate on start; fail closed if migrate fails.
- Single public HTTP port `3000`; API bound to loopback only.
- Do not change `deploy/Dockerfile.api`, `Dockerfile.worker`, `Dockerfile.web`, `docker-compose.yml`, or `nginx.web.conf`.
- Document single-replica only (no multi-replica migrate race).
- Helm remains out of scope; include a minimal K8s sketch in docs only.
- Work in the isolated git worktree; conventional commits; gate with `make lint type test` where applicable (image tasks use Docker smoke).

## File map

| File | Responsibility |
|------|----------------|
| `deploy/nginx.all-in-one.conf` | nginx on `:3000`; proxy API/auth/health to `127.0.0.1:8080` |
| `deploy/entrypoint.sh` | migrate → start children → signal + death handling |
| `deploy/entrypoint_test.sh` | Host-side smoke for entrypoint fail-fast / migrate invocation |
| `deploy/Dockerfile` | Multi-stage all-in-one image |
| `.github/workflows/publish-image.yml` | Buildx multi-arch push on `v*` tags |
| `Makefile` | Optional `image` / `image-smoke` helpers |
| `docs/07-setup-deployment.md` | Production image runbook |
| `docs/00-product-brief.md` | Adjust non-goal: image yes, Helm later |
| `README.md` | Link to production image docs |
| `deploy/k8s/aegis.yaml` | Minimal single-replica Deployment/Service sketch (optional but preferred for docs accuracy) |

---

### Task 1: All-in-one nginx config

**Files:**
- Create: `deploy/nginx.all-in-one.conf`
- Test: visual/config review + later Docker smoke (Task 4)

**Interfaces:**
- Consumes: none
- Produces: nginx config listening on `3000`, proxying `/api/`, `/auth/`, `/healthz`, `/readyz`, `/metrics` to `http://127.0.0.1:8080`

- [ ] **Step 1: Create nginx config**

Create `deploy/nginx.all-in-one.conf`:

```nginx
server {
    listen 3000;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /auth/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location = /healthz {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }

    location = /readyz {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }

    location = /metrics {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

- [ ] **Step 2: Confirm Compose nginx is untouched**

Run: `git diff -- deploy/nginx.web.conf`
Expected: empty (no changes)

- [ ] **Step 3: Commit**

```bash
git add deploy/nginx.all-in-one.conf
git commit -m "chore: add all-in-one nginx config for single-port image"
```

---

### Task 2: Entrypoint script + host tests

**Files:**
- Create: `deploy/entrypoint.sh`
- Create: `deploy/entrypoint_test.sh`
- Test: `deploy/entrypoint_test.sh`

**Interfaces:**
- Consumes: env `DATABASE_URL`; binaries `/app/api`, `/app/worker`; `migrate` on `PATH`; nginx config already installed in image
- Produces: process that exits non-zero if migrate fails or a child dies; exports `HTTP_ADDR=127.0.0.1:8080` for the API child

- [ ] **Step 1: Write the failing host test**

Create `deploy/entrypoint_test.sh`:

```bash
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bash deploy/entrypoint_test.sh`
Expected: FAIL (missing `entrypoint.sh` or missing env override support)

- [ ] **Step 3: Implement entrypoint**

Create `deploy/entrypoint.sh`:

```bash
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

export HTTP_ADDR="${HTTP_ADDR:-127.0.0.1:8080}"

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
```

Make executable: `chmod +x deploy/entrypoint.sh deploy/entrypoint_test.sh`

- [ ] **Step 4: Run tests to verify they pass**

Run: `bash deploy/entrypoint_test.sh`
Expected: `entrypoint_test.sh OK`

- [ ] **Step 5: Commit**

```bash
git add deploy/entrypoint.sh deploy/entrypoint_test.sh
git commit -m "feat: add all-in-one container entrypoint with migrate-on-start"
```

---

### Task 3: All-in-one Dockerfile

**Files:**
- Create: `deploy/Dockerfile`
- Test: `docker build` (Task 4 completes end-to-end smoke)

**Interfaces:**
- Consumes: `deploy/entrypoint.sh`, `deploy/nginx.all-in-one.conf`, `db/migrations`, `apps/*`, `pkg/*`
- Produces: image with `/app/api`, `/app/worker`, web assets, `/migrations`, `migrate`, entrypoint as `CMD`

- [ ] **Step 1: Create Dockerfile**

Create `deploy/Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS go-api
WORKDIR /src
COPY pkg/ pkg/
COPY apps/api/ apps/api/
WORKDIR /src/apps/api
ENV GOWORK=off
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM golang:1.25-alpine AS go-worker
WORKDIR /src
COPY pkg/ pkg/
COPY apps/worker/ apps/worker/
WORKDIR /src/apps/worker
ENV GOWORK=off
RUN CGO_ENABLED=0 go build -o /worker ./cmd/worker

FROM node:20-alpine AS web
WORKDIR /src
COPY apps/web/package.json apps/web/package-lock.json* ./apps/web/
WORKDIR /src/apps/web
RUN npm ci || npm install
COPY apps/web .
RUN npm run build

FROM nginx:1.27-alpine

ARG TARGETARCH
RUN apk add --no-cache curl ca-certificates \
  && curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-${TARGETARCH}.tar.gz" \
    | tar xz -C /usr/local/bin \
  && chmod +x /usr/local/bin/migrate

COPY --from=go-api /api /app/api
COPY --from=go-worker /worker /app/worker
COPY pkg/i18n/messages /app/pkg/i18n/messages
COPY --from=web /src/apps/web/dist /usr/share/nginx/html
COPY db/migrations /migrations
COPY deploy/nginx.all-in-one.conf /etc/nginx/conf.d/default.conf
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /app/api /app/worker

ENV HTTP_ADDR=127.0.0.1:8080
EXPOSE 3000
ENTRYPOINT ["/entrypoint.sh"]
```

- [ ] **Step 2: Build image locally (amd64 host)**

Run:

```bash
docker build -f deploy/Dockerfile -t aegis:local .
```

Expected: build succeeds (may take several minutes).

- [ ] **Step 3: Commit**

```bash
git add deploy/Dockerfile
git commit -m "feat: add all-in-one production Dockerfile"
```

---

### Task 4: Makefile helpers + Docker smoke against Postgres

**Files:**
- Modify: `Makefile`
- Test: local Docker smoke (manual in this task)

**Interfaces:**
- Consumes: image from Task 3; Postgres reachable via `DATABASE_URL`
- Produces: `make image` and `make image-smoke` targets

- [ ] **Step 1: Add Makefile targets**

Append to `Makefile` `.PHONY` list and add:

```makefile
.PHONY: ... image image-smoke

IMAGE_NAME ?= aegis:local

image:
	docker build -f deploy/Dockerfile -t $(IMAGE_NAME) .

# Requires a running Postgres and env file. Example:
#   make dev-db
#   DATABASE_URL=postgres://aegis:aegis@host.docker.internal:5432/aegis?sslmode=disable \
#   SESSION_SECRET=x WEBHOOK_SECRET=x PUBLIC_URL=http://localhost:3000 \
#   make image-smoke
image-smoke: image
	@test -n "$${DATABASE_URL}" || (echo "DATABASE_URL required" >&2; exit 1)
	@test -n "$${SESSION_SECRET}" || (echo "SESSION_SECRET required" >&2; exit 1)
	@test -n "$${WEBHOOK_SECRET}" || (echo "WEBHOOK_SECRET required" >&2; exit 1)
	@test -n "$${PUBLIC_URL}" || (echo "PUBLIC_URL required" >&2; exit 1)
	docker run --rm -d --name aegis-smoke -p 3000:3000 \
	  -e DATABASE_URL \
	  -e SESSION_SECRET \
	  -e WEBHOOK_SECRET \
	  -e PUBLIC_URL \
	  -e HTTP_ADDR=127.0.0.1:8080 \
	  $(IMAGE_NAME)
	@echo "Waiting for /healthz..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
	  if curl -fsS http://127.0.0.1:3000/healthz >/dev/null; then break; fi; \
	  sleep 2; \
	done
	curl -fsS http://127.0.0.1:3000/healthz
	curl -fsS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3000/ | grep -E '200|304'
	docker stop aegis-smoke
```

On Linux CI/agents without `host.docker.internal`, use the Postgres container IP or `--network` shared with Compose Postgres. Document the Darwin default in `docs/07-setup-deployment.md` (Task 6).

- [ ] **Step 2: Run smoke**

```bash
make dev-db
# Adjust host for your OS:
export DATABASE_URL='postgres://aegis:aegis@host.docker.internal:5432/aegis?sslmode=disable'
export SESSION_SECRET=smoke-session
export WEBHOOK_SECRET=smoke-webhook
export PUBLIC_URL=http://localhost:3000
make image-smoke
```

Expected: `/healthz` returns 200; root returns 200; container stops cleanly.

- [ ] **Step 3: Child-death check (manual)**

```bash
# Start container as in image-smoke but leave it running, then:
docker exec aegis-smoke sh -c 'kill $(pidof api) || kill $(pgrep -f /app/api)'
# Observe container exits (docker ps should not list it after a few seconds)
```

Expected: container exits non-zero / stops (entrypoint supervised).

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "chore: add make image and image-smoke targets"
```

---

### Task 5: GHCR publish workflow

**Files:**
- Create: `.github/workflows/publish-image.yml`
- Test: workflow YAML validation (actionlint if available) + dry review

**Interfaces:**
- Consumes: `deploy/Dockerfile` on tag `v*`
- Produces: `ghcr.io/btb-hub/aegis:<tag>` and `:latest` for `linux/amd64,linux/arm64`

- [ ] **Step 1: Create workflow**

Create `.github/workflows/publish-image.yml`:

```yaml
name: Publish image

on:
  push:
    tags:
      - "v*"

permissions:
  contents: read
  packages: write

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Docker metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ghcr.io/${{ github.repository_owner }}/aegis
          tags: |
            type=ref,event=tag
            type=raw,value=latest

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          file: deploy/Dockerfile
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
```

Note: `github.repository_owner` resolves to `btb-hub` for this repo, matching `ghcr.io/btb-hub/aegis`.

- [ ] **Step 2: Confirm CI gate workflow unchanged**

Run: `git diff -- .github/workflows/ci.yml`
Expected: empty

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/publish-image.yml
git commit -m "ci: publish multi-arch all-in-one image to GHCR on version tags"
```

---

### Task 6: Minimal K8s sketch + documentation

**Files:**
- Create: `deploy/k8s/aegis.yaml`
- Modify: `docs/07-setup-deployment.md`
- Modify: `docs/00-product-brief.md`
- Modify: `README.md`
- Modify: `docs/superpowers/specs/2026-07-21-all-in-one-ghcr-image-design.md` (status → Approved)

**Interfaces:**
- Consumes: published image name and port 3000
- Produces: operator-facing runbook for `docker run` and single-replica K8s

- [ ] **Step 1: Add minimal K8s manifest**

Create `deploy/k8s/aegis.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aegis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: aegis
  template:
    metadata:
      labels:
        app: aegis
    spec:
      containers:
        - name: aegis
          image: ghcr.io/btb-hub/aegis:latest
          ports:
            - containerPort: 3000
          envFrom:
            - secretRef:
                name: aegis-env
          readinessProbe:
            httpGet:
              path: /readyz
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /healthz
              port: 3000
            initialDelaySeconds: 15
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: aegis
spec:
  selector:
    app: aegis
  ports:
    - port: 80
      targetPort: 3000
```

- [ ] **Step 2: Update setup docs**

At the top of `docs/07-setup-deployment.md`, after the opening paragraph, insert a **Production image** section **before** the Compose quick start (keep Compose as local/dev):

```markdown
## Production image (GHCR)

Third parties and Kubernetes deploy from a **single public image** — API, worker, and web in one
container — with **external Postgres**. Docker Compose is for local source builds only.

### Pull and run

```bash
docker pull ghcr.io/btb-hub/aegis:vX.Y.Z

docker run --rm -p 3000:3000 \
  -e DATABASE_URL=postgres://user:pass@db-host:5432/aegis?sslmode=require \
  -e SESSION_SECRET=... \
  -e WEBHOOK_SECRET=... \
  -e PUBLIC_URL=https://aegis.example.com \
  # plus OIDC / connector vars from deploy/.env.example \
  ghcr.io/btb-hub/aegis:vX.Y.Z
```

Open `https://aegis.example.com` (or `http://localhost:3000` for a local trial). Probes:
`GET /healthz`, `GET /readyz` on port 3000.

Migrations run automatically on start. **Run a single replica** — concurrent migrate-on-start
across replicas is not supported yet.

### Kubernetes sketch

See [`deploy/k8s/aegis.yaml`](../deploy/k8s/aegis.yaml). Create a Secret named `aegis-env` with the
same keys as `.env`, point Ingress at Service port 80, and set `PUBLIC_URL` to the Ingress URL.
OIDC redirect URIs must match that host. Helm remains out of scope.

### Publishing (maintainers)

Push a version tag (`v0.1.0`, …). GitHub Actions builds `linux/amd64` + `linux/arm64` and pushes
`ghcr.io/btb-hub/aegis:<tag>` and `:latest`. After the first push, set the GHCR package visibility
to **Public** in the GitHub UI (one-time).
```

Also change the opening line from “MVP runs on **Docker Compose**” to clarify Compose is the local path and the production path is the GHCR image.

- [ ] **Step 3: Update product brief non-goal**

In `docs/00-product-brief.md`, replace:

```markdown
- Helm/Kubernetes deploy (Docker Compose only for MVP)
```

with:

```markdown
- Helm charts and multi-replica Kubernetes operators (single all-in-one image + sketch manifests are in-scope; Compose remains the local source-build path)
```

- [ ] **Step 4: Update README**

In `README.md`, near setup/run instructions (or in “Read in this order” notes for doc 07), add one line:

```markdown
Production/K8s: pull `ghcr.io/btb-hub/aegis` — see [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md)#production-image-ghcr.
```

(If README has no run section yet, add a short **Run** subsection with local Compose + production image link.)

- [ ] **Step 5: Mark design approved**

In the design spec header, set `**Status:** Approved`.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/aegis.yaml docs/07-setup-deployment.md docs/00-product-brief.md README.md \
  docs/superpowers/specs/2026-07-21-all-in-one-ghcr-image-design.md
git commit -m "docs: document all-in-one GHCR image and K8s sketch"
```

---

### Task 7: Final verification gate

**Files:** none new

- [ ] **Step 1: Run host entrypoint tests**

Run: `bash deploy/entrypoint_test.sh`  
Expected: `entrypoint_test.sh OK`

- [ ] **Step 2: Run project gate**

Run: `make lint type test`  
Expected: green (no regressions from docs/Makefile-only changes)

- [ ] **Step 3: Confirm Compose path still builds**

Run: `docker compose -f deploy/docker-compose.yml config`  
Expected: valid config; services still use `Dockerfile.api` / `worker` / `web`

- [ ] **Step 4: Spec coverage checklist**

Confirm each design requirement has a task deliverable:

| Spec item | Task |
|-----------|------|
| Single GHCR image | 3, 5 |
| API+worker+web together | 2, 3 |
| External Postgres | 2, 6 |
| Migrate on start | 2 |
| One port / nginx proxy | 1, 3 |
| Multi-arch + `v*` publish | 5 |
| Compose unchanged for local | 1, 3, 7 |
| Docs + single-replica note | 6 |
| Child death → container exit | 2, 4 |

- [ ] **Step 5: Final commit if any fixups**

Only if Step 1–3 required fixes; otherwise done.

---

## Self-review (plan author)

1. **Spec coverage:** All design sections §1–§8 mapped to Tasks 1–7; ops checklist remains human (GHCR public toggle, first tag).
2. **Placeholders:** None; concrete file contents and commands included.
3. **Consistency:** Image `ghcr.io/btb-hub/aegis`, port `3000`, `HTTP_ADDR=127.0.0.1:8080`, migrate path `/migrations`, workflow platforms amd64/arm64 match the spec.
