# Design: All-in-one GHCR image for third-party and Kubernetes deploy

**Date:** 2026-07-21  
**Status:** Draft for review  
**Context:** Aegis today ships only source builds via Docker Compose. Third parties cannot pull a published image; production targets Kubernetes, not Compose. Helm remains post-MVP.

## Goals

1. Publish one public multi-arch container image to GHCR that third parties can pull without cloning the repo.
2. Run API, worker, and web in a **single container** with one HTTP port.
3. Use **external Postgres**; apply migrations automatically on container start.
4. Keep the existing multi-service Compose stack for local source development.

## Non-goals

- Docker Compose as the production distribution path.
- Embedding Postgres in the image.
- Helm chart, HPA, or multi-replica orchestration (document single-replica for this MVP image).
- Separate published images per service (api / worker / web).
- Publishing images on every `main` push (version tags only).

## Decisions (from brainstorming)

| Topic | Choice |
|-------|--------|
| Registry | GitHub Container Registry (`ghcr.io`) |
| Visibility | Public pulls (no auth required to pull) |
| Platforms | `linux/amd64` and `linux/arm64` |
| Publish trigger | Git tags matching `v*` → semver tag + `latest` |
| Distribution shape | **One image**, all processes started together |
| Database | External Postgres only |
| Migrations | Auto-migrate on container start |
| Ingress surface | One port: nginx serves UI and proxies API/auth |
| Process model | Entrypoint script (no s6 / supervisord) |
| Local dev | Existing Compose + per-service Dockerfiles unchanged |

## Architecture

```text
ghcr.io/btb-hub/aegis:<semver>
        │
        ▼
┌─────────────────────────────────────────┐
│  entrypoint.sh                          │
│    1. migrate up (fail → exit)          │
│    2. start api  (127.0.0.1:8080)       │
│    3. start worker                      │
│    4. start nginx (0.0.0.0:3000)        │
│    5. wait; any child death → exit ≠ 0  │
└─────────────────────────────────────────┘
        │                    │
        │ HTTP :3000         │ DATABASE_URL
        ▼                    ▼
   Ingress / docker      External Postgres
```

**Image name:** `ghcr.io/btb-hub/aegis`  
**Tags:** immutable `vX.Y.Z` (or whatever git tag was pushed) plus moving `latest` on each version tag publish.

---

## §1 — Image contents

**New files (repo):**

| Path | Role |
|------|------|
| `deploy/Dockerfile` | Multi-stage all-in-one image |
| `deploy/entrypoint.sh` | Migrate, start children, signal handling |
| `deploy/nginx.all-in-one.conf` | Listen `:3000`; proxy API routes to `127.0.0.1:8080` |

**Image layers (conceptual):**

1. Build Go API and worker binaries (same base approach as existing Dockerfiles).
2. Build web static assets (`npm run build`).
3. Runtime base: Alpine (or nginx Alpine) with:
   - `/app/api`, `/app/worker`
   - web assets under nginx html root
   - `db/migrations` at `/migrations`
   - `migrate` CLI binary
   - `entrypoint.sh`, nginx config

**Keep unchanged for local/dev:** `deploy/Dockerfile.api`, `Dockerfile.worker`, `Dockerfile.web`, `docker-compose.yml`, `nginx.web.conf` (Compose still uses hostname `api`).

---

## §2 — Entrypoint behavior

1. **Validate** that `DATABASE_URL` is set before migrate (fail fast with a clear message). Full app config validation remains in each binary at process start.
2. **Migrate:** `migrate -path /migrations -database "$DATABASE_URL" up`. Any non-zero exit aborts the container; do not start API/worker/nginx.
3. **Start API** with `HTTP_ADDR=127.0.0.1:8080` (or equivalent so the API is not exposed outside the container).
4. **Start worker** with the same env (reads `DATABASE_URL` and connector secrets as today).
5. **Start nginx** listening on `0.0.0.0:3000`.
6. **Signals:** trap `SIGTERM`/`SIGINT`, forward to children, wait for exit, then exit.
7. **Supervision:** if any child exits unexpectedly, stop remaining children and exit non-zero so Kubernetes restarts the pod. No in-container silent restart loops.

---

## §3 — Nginx routing (single port)

Public port: **3000**.

| Path | Target |
|------|--------|
| `/api/` | `http://127.0.0.1:8080` |
| `/auth/` | `http://127.0.0.1:8080` |
| `/healthz`, `/readyz`, `/metrics` | `http://127.0.0.1:8080` (for probes and ops) |
| `/` (and SPA fallback) | static web assets |

Forward `Host`, `X-Real-IP`, `X-Forwarded-For`, `X-Forwarded-Proto` as in the current Compose nginx config.

**Config:** `PUBLIC_URL` must be the externally reachable URL that hits port 3000 (Ingress host / TLS terminator). Session cookies and OIDC redirects depend on this.

---

## §4 — Health and readiness

- **Liveness / readiness probes** target `http://<pod>:3000/healthz` and `http://<pod>:3000/readyz`.
- `/readyz` remains DB-aware (existing API behavior).
- Because migrate runs before nginx starts accepting traffic, a newly started container is not probe-reachable until schema apply and API start succeed (or the pod fails and restarts).

**MVP scale note:** Auto-migrate on start is safe for **one replica**. Multiple replicas racing migrate is out of scope; docs must say run a single replica (or a separate migrate Job later).

---

## §5 — Publish pipeline

**Workflow:** `.github/workflows/publish-image.yml`

| Trigger | Action |
|---------|--------|
| Push tag `v*` | Buildx multi-arch; push `ghcr.io/btb-hub/aegis:<tag>` and `:latest` |
| PR / `main` | No image publish (existing `ci.yml` gate only) |

**Auth:** `GITHUB_TOKEN` with `packages: write` (or org-standard GHCR credentials).  
**Visibility:** package set to **public** once in GHCR UI (document as a one-time ops step).

**Optional:** attach a short GitHub Release body with `docker run` and a minimal K8s manifest snippet (nice-to-have; not required for first ship).

---

## §6 — Consumer experience

**Minimal run (Docker):**

```bash
docker run --rm -p 3000:3000 \
  -e DATABASE_URL=postgres://... \
  -e SESSION_SECRET=... \
  -e WEBHOOK_SECRET=... \
  -e PUBLIC_URL=https://aegis.example.com \
  # …OIDC and connector env as in deploy/.env.example \
  ghcr.io/btb-hub/aegis:vX.Y.Z
```

**Kubernetes (sketch, not a Helm chart):** one Deployment (replicas: 1), one Service on port 3000, Ingress → Service; env via Secret/ConfigMap; external Postgres URL.

**Docs updates:**

- `docs/07-setup-deployment.md` — “Production image” section (pull, env, probes, single-replica caveat).
- `docs/00-product-brief.md` — clarify image-based deploy exists; Helm still later.
- README — link to pull/run instructions.

---

## §7 — Testing / verification

1. Local `docker build` of `deploy/Dockerfile`; run against a local/external Postgres; confirm migrate applies, UI loads on `:3000`, `/api` and `/auth` proxy correctly.
2. Process contract: stop the API child → container exits non-zero.
3. After first tag publish: `docker pull ghcr.io/btb-hub/aegis:<tag>` without auth.
4. Existing `make lint type test` / Compose path remains green and unchanged in behavior.

---

## §8 — Implementation outline (for the follow-on plan)

1. Add Dockerfile, entrypoint, nginx all-in-one config.
2. Add publish workflow; document GHCR public package step.
3. Update setup/deployment docs and README.
4. Manual smoke on amd64 (and arm64 if available) before first production tag.

## Open ops checklist (not code)

- [ ] Confirm GHCR package namespace `btb-hub/aegis` and public visibility.
- [ ] First release tag convention (`v0.1.0` vs calendar) agreed by maintainers.
- [ ] Production `PUBLIC_URL` and OIDC redirect URIs point at the Ingress host on port 443 → Service 3000.
