# ADMIN_EMAILS Bootstrap + User Role Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let operators become admin on a deployed instance via `ADMIN_EMAILS`, then let admins change other users’ roles through `PATCH /api/v1/users/{id}` and a `/users` UI.

**Architecture:** Parse `ADMIN_EMAILS` into a normalized set on config load. On every OIDC `CompleteLogin`, promote matching users to `admin` (with audit). Day-2 adds store/service/handler role updates with last-admin and env-pin guards, surfaces `role_pinned` on the users list, and ships an admin Users page.

**Tech Stack:** Go 1.22+ (`apps/api`, `pkg/config`, `pkg/db`), Gin, PostgreSQL via pgx, React + TypeScript + Vite (`apps/web`), Vitest, react-i18next.

**Spec:** [`docs/superpowers/specs/2026-07-20-admin-emails-bootstrap-design.md`](../specs/2026-07-20-admin-emails-bootstrap-design.md)

## Global Constraints

- Roles are only `admin` | `member` | `viewer` (`pkg/rbac`).
- Env var name is `ADMIN_EMAILS` (unprefixed), comma-separated, trim + lower-case.
- Env promotion runs on OIDC login only — not on `GET /auth/me`, middleware, or DevAuth.
- DevAuth stays independent (`?role=` / `DEV_AUTH_DEFAULT_ROLE`); do not apply `ADMIN_EMAILS` to the `dev` login path.
- API errors use `{code, message, details}`; UI never shows bare stack traces.
- User-facing copy in both `en` and `ru`.
- Ship as **two PRs**: Phase 1 (Tasks 1–4) then Phase 2 (Tasks 5–8). Gate each PR with `make lint type test`.
- Work only in the isolated git worktree; conventional commits; do not rewrite Done backlog ACs — append new stories.

## File map

| File | Responsibility |
|------|----------------|
| `pkg/config/config.go` | `AdminEmails map[string]struct{}`, parse + `IsAdminEmail` |
| `pkg/config/config_test.go` | Config parse / validation tests |
| `pkg/db/users_role.go` (new) | `UpdateUserRole`, `CountUsersByRole`, `WriteAuditLog` |
| `apps/api/internal/service/auth.go` | Promote after `ResolveOIDCLogin` |
| `apps/api/internal/service/auth_mock_users_test.go` | Mock `UpdateUserRole` + audit hook |
| `apps/api/internal/service/auth_test.go` | Login promotion tests |
| `deploy/.env.example`, `deploy/.env.local.example` | Document `ADMIN_EMAILS` |
| `docs/07-setup-deployment.md`, `docs/09-security.md` | First-admin runbook + security note |
| `apps/api/internal/service/user.go` | `UpdateUserRole` with guards |
| `apps/api/internal/handler/user.go` | `PATCH /users/{id}`, `role_pinned` on list |
| `docs/04-api-spec.md` | Document PATCH + `role_pinned` |
| `apps/web/src/pages/UsersPage.tsx` (+ test/stories) | Admin users table + role select |
| `apps/web/src/App.tsx`, `AppShell.tsx`, locales | Route + admin nav |
| `backlog/epics/EPIC-09-teams-users-shifts.md` | Append Ready stories |

---

# Phase 1 — Bootstrap (PR 1)

Unblocks production: set env, restart API, sign in once.

### Task 1: Parse `ADMIN_EMAILS` in config

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go`
- Modify: `deploy/.env.example`
- Modify: `deploy/.env.local.example`

**Interfaces:**
- Produces: `Config.AdminEmails map[string]struct{}`; `func (c *Config) IsAdminEmail(email string) bool`

- [ ] **Step 1: Write failing config tests**

Add to `pkg/config/config_test.go`:

```go
func TestLoadAdminEmails(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ADMIN_EMAILS", " Alice@Co.com , bob@co.com,, ")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.IsAdminEmail("alice@co.com"))
	require.True(t, cfg.IsAdminEmail("BOB@CO.COM"))
	require.False(t, cfg.IsAdminEmail("other@co.com"))
	require.Len(t, cfg.AdminEmails, 2)
}

func TestLoadAdminEmailsEmpty(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ADMIN_EMAILS", "")

	cfg, err := Load()
	require.NoError(t, err)
	require.Empty(t, cfg.AdminEmails)
	require.False(t, cfg.IsAdminEmail("anyone@co.com"))
}

func TestLoadAdminEmailsRejectsInvalid(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ADMIN_EMAILS", "not-an-email")

	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "ADMIN_EMAILS")
}
```

- [ ] **Step 2: Run tests — expect fail**

Run: `cd /path/to/worktree && go test ./pkg/config/ -run TestLoadAdminEmails -count=1`

Expected: FAIL (unknown field / undefined `IsAdminEmail`).

- [ ] **Step 3: Implement parsing**

In `pkg/config/config.go`:

1. Add field to `Config`:
   ```go
   AdminEmails map[string]struct{}
   ```
2. In `Load()`, after building `cfg`:
   ```go
   adminEmails, err := parseAdminEmails(os.Getenv("ADMIN_EMAILS"))
   if err != nil {
       return nil, err
   }
   cfg.AdminEmails = adminEmails
   ```
3. Add helpers:
   ```go
   func parseAdminEmails(raw string) (map[string]struct{}, error) {
       out := make(map[string]struct{})
       for _, part := range strings.Split(raw, ",") {
           email := strings.ToLower(strings.TrimSpace(part))
           if email == "" {
               continue
           }
           if !strings.Contains(email, "@") {
               return nil, fmt.Errorf("invalid ADMIN_EMAILS entry %q: must contain @", part)
           }
           out[email] = struct{}{}
       }
       return out, nil
   }

   func (c *Config) IsAdminEmail(email string) bool {
       if c == nil || len(c.AdminEmails) == 0 {
           return false
       }
       _, ok := c.AdminEmails[strings.ToLower(strings.TrimSpace(email))]
       return ok
   }
   ```

- [ ] **Step 4: Re-run tests — expect pass**

Run: `go test ./pkg/config/ -run 'TestLoadAdminEmails|TestLoadSuccess' -count=1`

Expected: PASS.

- [ ] **Step 5: Document env var**

Add to both `deploy/.env.example` and `deploy/.env.local.example` (near core secrets):

```bash
# Comma-separated emails forced to admin on OIDC login (always-on)
ADMIN_EMAILS=
```

- [ ] **Step 6: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go deploy/.env.example deploy/.env.local.example
git commit -m "feat: parse ADMIN_EMAILS allowlist from config"
```

---

### Task 2: Store `UpdateUserRole` + `WriteAuditLog`

**Files:**
- Create: `pkg/db/users_role.go`
- Create: `pkg/db/users_role_test.go` (optional unit with real DB only if suite exists; prefer service-level mocks — if no DB test harness for store, skip store tests and cover via service mocks in Task 3)

**Interfaces:**
- Produces:
  ```go
  func (s *Store) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (User, error)
  func (s *Store) CountUsersByRole(ctx context.Context, role string) (int, error)
  func (s *Store) WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error
  ```
- Note: `actorID` nullable — for env promotion use pointer to the user’s own ID.

- [ ] **Step 1: Add store methods**

Create `pkg/db/users_role.go`:

```go
package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *Store) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (User, error) {
	q := `UPDATE users SET role = $2 WHERE id = $1 RETURNING ` + userSelectColumns
	return scanUser(s.pool.QueryRow(ctx, q, id, role))
}

func (s *Store) CountUsersByRole(ctx context.Context, role string) (int, error) {
	const q = `SELECT COUNT(*) FROM users WHERE role = $1`
	var n int
	err := s.pool.QueryRow(ctx, q, role).Scan(&n)
	return n, err
}

func (s *Store) WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO audit_log (actor_id, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5)`, actorID, action, resourceType, resourceID, payload)
	return err
}
```

Match `writeAuditLogTx` column types in `pkg/db/identity.go` (if `actor_id` is `NOT NULL`, pass non-nil UUID — use the promoted user’s ID).

- [ ] **Step 2: Confirm compile**

Run: `go build ./pkg/db/`

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add pkg/db/users_role.go
git commit -m "feat: add UpdateUserRole and WriteAuditLog store helpers"
```

---

### Task 3: Promote matching emails on OIDC login

**Files:**
- Modify: `apps/api/internal/service/auth.go` (`UserRepository` + `CompleteLogin`)
- Modify: `apps/api/internal/service/auth_mock_users_test.go`
- Modify: `apps/api/internal/service/auth_test.go`

**Interfaces:**
- Consumes: `Config.IsAdminEmail`, `UpdateUserRole`, `WriteAuditLog`
- Extends `UserRepository`:
  ```go
  UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (db.User, error)
  WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error
  ```

- [ ] **Step 1: Write failing auth tests**

In `apps/api/internal/service/auth_test.go`, add a mock OIDC that returns a configurable email, or set up users mock with a pre-existing member and custom exchanger:

```go
type emailOIDC struct {
	email string
	sub   string
}

func (m *emailOIDC) AuthCodeURL(provider, state string) (string, error) {
	return "https://idp.example/authorize?state=" + state, nil
}

func (m *emailOIDC) Exchange(ctx context.Context, provider, code string) (*OIDCUserInfo, error) {
	return &OIDCUserInfo{Sub: m.sub, Email: m.email, DisplayName: "User"}, nil
}

func TestCompleteLoginPromotesAdminEmail(t *testing.T) {
	cfg := &config.Config{
		SessionTTL:  24 * time.Hour,
		AdminEmails: map[string]struct{}{"admin@co.com": {}},
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := newIdentityMockUsers()
	sessions := &mockSessions{byHash: map[string]db.Session{}}
	svc := NewAuthService(cfg, users, sessions, &emailOIDC{email: "Admin@Co.com", sub: "sub-admin"})

	token, user, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, "admin", user.Role)
	require.Equal(t, "admin", users.users[user.ID].Role)
	require.NotEmpty(t, users.audits) // see mock step
	require.Equal(t, "user.role_changed", users.audits[0].Action)
}

func TestCompleteLoginDoesNotPromoteUnlisted(t *testing.T) {
	cfg := &config.Config{
		SessionTTL:  24 * time.Hour,
		AdminEmails: map[string]struct{}{"admin@co.com": {}},
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := newIdentityMockUsers()
	sessions := &mockSessions{byHash: map[string]db.Session{}}
	svc := NewAuthService(cfg, users, sessions, &emailOIDC{email: "member@co.com", sub: "sub-m"})

	_, user, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)
	require.Equal(t, "member", user.Role)
	require.Empty(t, users.audits)
}

func TestCompleteLoginAdminEmailAlreadyAdminNoAudit(t *testing.T) {
	cfg := &config.Config{
		SessionTTL:  24 * time.Hour,
		AdminEmails: map[string]struct{}{"admin@co.com": {}},
		OIDC: map[string]config.OIDCProvider{
			"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"},
		},
	}
	users := newIdentityMockUsers()
	existing := db.User{
		ID: uuid.New(), Provider: "google", ProviderSub: "sub-a",
		Email: "admin@co.com", DisplayName: "A", Role: "admin", Locale: "en",
	}
	users.users[existing.ID] = existing
	users.byEmail["admin@co.com"] = existing.ID
	users.identities["google:sub-a"] = db.UserIdentity{ID: uuid.New(), UserID: existing.ID, Provider: "google", ProviderSub: "sub-a"}

	sessions := &mockSessions{byHash: map[string]db.Session{}}
	svc := NewAuthService(cfg, users, sessions, &emailOIDC{email: "admin@co.com", sub: "sub-a"})

	_, user, err := svc.CompleteLogin(context.Background(), "google", "code")
	require.NoError(t, err)
	require.Equal(t, "admin", user.Role)
	require.Empty(t, users.audits)
}
```

Adjust mock field names to match whatever you add in Step 2.

- [ ] **Step 2: Extend `identityMockUsers`**

In `auth_mock_users_test.go`, add:

```go
type mockAudit struct {
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
	Details      map[string]any
}

// on identityMockUsers:
audits []mockAudit

func (m *identityMockUsers) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (db.User, error) {
	user, ok := m.users[id]
	if !ok {
		return db.User{}, errors.New("not found")
	}
	user.Role = role
	m.users[id] = user
	return user, nil
}

func (m *identityMockUsers) WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error {
	m.audits = append(m.audits, mockAudit{ActorID: actorID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Details: details})
	return nil
}
```

- [ ] **Step 3: Run tests — expect fail**

Run: `go test ./apps/api/internal/service/ -run 'TestCompleteLoginPromotes|TestCompleteLoginDoesNot|TestCompleteLoginAdminEmail' -count=1`

Expected: FAIL (interface missing methods and/or no promotion).

- [ ] **Step 4: Implement promotion in `CompleteLogin`**

1. Extend `UserRepository` in `auth.go` with `UpdateUserRole` and `WriteAuditLog` signatures from Task 2.
2. After `user = result.User` and **before** session creation:

```go
if s.cfg.IsAdminEmail(user.Email) && user.Role != string(rbac.RoleAdmin) {
	oldRole := user.Role
	updated, err := s.users.UpdateUserRole(ctx, user.ID, string(rbac.RoleAdmin))
	if err != nil {
		return "", db.User{}, err
	}
	actorID := user.ID
	if err := s.users.WriteAuditLog(ctx, &actorID, "user.role_changed", "user", user.ID, map[string]any{
		"old_role": oldRole,
		"new_role": string(rbac.RoleAdmin),
		"reason":   "admin_emails_env",
	}); err != nil {
		return "", db.User{}, err
	}
	user = updated
}
```

Import `pkg/rbac` if not already imported.

Do **not** add this block to `DevLogin`.

- [ ] **Step 5: Run tests — expect pass**

Run: `go test ./apps/api/internal/service/ -count=1`

Expected: PASS (full package).

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/service/auth.go apps/api/internal/service/auth_test.go apps/api/internal/service/auth_mock_users_test.go
git commit -m "feat: promote ADMIN_EMAILS users to admin on OIDC login"
```

---

### Task 4: Phase 1 docs + backlog story + verify

**Files:**
- Modify: `docs/07-setup-deployment.md`
- Modify: `docs/09-security.md`
- Modify: `backlog/epics/EPIC-09-teams-users-shifts.md` (append story; do not edit Done ACs)

- [ ] **Step 1: Add “First admin” section to setup docs**

In `docs/07-setup-deployment.md`, after OIDC env tables (or under a new **Production first admin** heading):

```markdown
## First admin (production)

OIDC users are created as `member`. To grant admin on a deployed instance:

1. Set `ADMIN_EMAILS=you@company.com` (comma-separated for multiple) in `.env`.
2. Restart the API so config reloads.
3. Sign in with OIDC using that email.
4. Confirm `GET /auth/me` returns `"role": "admin"`.

Emails in `ADMIN_EMAILS` are re-asserted as admin on every OIDC login. Removing an email from the list stops re-promotion but does not demote automatically.

### Recovery without restart

If you cannot change env yet:

```sql
UPDATE users SET role = 'admin' WHERE email = 'you@company.com'
RETURNING id, email, role;
```

Session middleware reloads role from the DB on each request.
```

- [ ] **Step 2: Security note**

In `docs/09-security.md` Authorization section, add:

```markdown
- `ADMIN_EMAILS` — standing allowlist: matching OIDC logins are forced to `admin` and audited as `user.role_changed` / `reason=admin_emails_env`. Treat as access-defining ops config.
```

- [ ] **Step 3: Append backlog story**

Append under EPIC-09 (new story id next free, e.g. `AEG-072`):

```markdown
### AEG-072 — ADMIN_EMAILS first-admin bootstrap

- **Status:** Ready → set to In Review when PR opens
- **Depends on:** AEG-064
- **PRD:** REQ-AUTH-04, REQ-AUDIT-01
- **Acceptance:**
  - [ ] `ADMIN_EMAILS` parsed (trim, lower-case); invalid tokens fail config load
  - [ ] OIDC login promotes listed emails to `admin` with audit; unlisted unchanged; already-admin no-op
  - [ ] DevAuth path unchanged
  - [ ] Documented in setup + security + `.env.example`
```

- [ ] **Step 4: Full gate**

Run: `make lint type test`

Expected: green.

- [ ] **Step 5: Commit**

```bash
git add docs/07-setup-deployment.md docs/09-security.md backlog/epics/EPIC-09-teams-users-shifts.md
git commit -m "docs: document ADMIN_EMAILS first-admin bootstrap"
```

- [ ] **Step 6: Open PR 1** (when executing for real)

Branch name suggestion: `feat/auth-AEG-072-admin-emails-bootstrap`  
Title: `feat: ADMIN_EMAILS first-admin bootstrap`  
Link story AEG-072; set status In Review.

---

# Phase 2 — Role management (PR 2)

Depends on Phase 1 merged (or same branch continued only if user insists on one PR — prefer separate).

### Task 5: `UserService.UpdateUserRole` with guards

**Files:**
- Modify: `apps/api/internal/service/user.go`
- Modify: `apps/api/internal/service/user_test.go`

**Interfaces:**
- Extends repository used by `UserService`:
  ```go
  GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
  UpdateUserRole(ctx context.Context, id uuid.UUID, role string) (db.User, error)
  CountUsersByRole(ctx context.Context, role string) (int, error)
  WriteAuditLog(ctx context.Context, actorID *uuid.UUID, action, resourceType string, resourceID uuid.UUID, details map[string]any) error
  ```
- `UserService` gains `cfg *config.Config` (or `isAdminEmail func(string) bool`) via constructor change:
  ```go
  func NewUserService(repo UserListRepository, cfg *config.Config) *UserService
  ```
- Produces:
  ```go
  func (s *UserService) UpdateUserRole(ctx context.Context, actorID, targetID uuid.UUID, role string) (db.User, error)
  func (s *UserService) IsRolePinned(email string) bool
  ```

- [ ] **Step 1: Write failing service tests**

In `user_test.go` (extend existing mock):

```go
func TestUpdateUserRolePromotes(t *testing.T) { /* member -> admin, audit reason admin_api */ }
func TestUpdateUserRoleLastAdmin(t *testing.T) {
	// one admin in count; demote -> apperrors with Code "last_admin", Status 409
}
func TestUpdateUserRolePinnedByEnv(t *testing.T) {
	// target email in AdminEmails; request member -> Code "admin_emails_pinned"
}
func TestUpdateUserRoleIdempotent(t *testing.T) {
	// same role -> no audit
}
func TestUpdateUserRoleInvalid(t *testing.T) {
	// bad role -> VALIDATION_ERROR
}
```

Use `apperrors.New("last_admin", "Cannot demote the last admin", http.StatusConflict)` and `apperrors.New("admin_emails_pinned", "This user is pinned to admin by ADMIN_EMAILS. Remove the email from ADMIN_EMAILS and restart the API, then demote.", http.StatusConflict)`.

- [ ] **Step 2: Run — expect fail**

Run: `go test ./apps/api/internal/service/ -run TestUpdateUserRole -count=1`

Expected: FAIL.

- [ ] **Step 3: Implement**

```go
func (s *UserService) UpdateUserRole(ctx context.Context, actorID, targetID uuid.UUID, role string) (db.User, error) {
	parsed, err := rbac.Parse(role)
	if err != nil {
		return db.User{}, apperrors.Validation("invalid role", map[string]any{"role": role})
	}
	target, err := s.repo.GetUserByID(ctx, targetID)
	if err != nil {
		return db.User{}, apperrors.NotFound("user")
	}
	if target.Role == string(parsed) {
		return target, nil
	}
	if s.cfg.IsAdminEmail(target.Email) && parsed != rbac.RoleAdmin {
		return db.User{}, apperrors.New("admin_emails_pinned",
			"This user is pinned to admin by ADMIN_EMAILS. Remove the email from ADMIN_EMAILS and restart the API, then demote.",
			http.StatusConflict)
	}
	if target.Role == string(rbac.RoleAdmin) && parsed != rbac.RoleAdmin {
		n, err := s.repo.CountUsersByRole(ctx, string(rbac.RoleAdmin))
		if err != nil {
			return db.User{}, err
		}
		if n <= 1 {
			return db.User{}, apperrors.New("last_admin", "Cannot demote the last admin", http.StatusConflict)
		}
	}
	old := target.Role
	updated, err := s.repo.UpdateUserRole(ctx, targetID, string(parsed))
	if err != nil {
		return db.User{}, err
	}
	actor := actorID
	if err := s.repo.WriteAuditLog(ctx, &actor, "user.role_changed", "user", targetID, map[string]any{
		"old_role": old,
		"new_role": string(parsed),
		"reason":   "admin_api",
	}); err != nil {
		return db.User{}, err
	}
	return updated, nil
}

func (s *UserService) IsRolePinned(email string) bool {
	return s.cfg.IsAdminEmail(email)
}
```

Widen `UserListRepository` (rename to `UserRepository` if cleaner) with the new methods. Update `NewUserService` call sites in `apps/api/cmd/api` (or wire file) to pass `cfg`.

- [ ] **Step 4: Run package tests**

Run: `go test ./apps/api/internal/service/ -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/service/user.go apps/api/internal/service/user_test.go apps/api/cmd/api/
git commit -m "feat: add UserService.UpdateUserRole with last-admin and env-pin guards"
```

---

### Task 6: Handler `PATCH /users/{id}` + `role_pinned` on list

**Files:**
- Modify: `apps/api/internal/handler/user.go`
- Modify: `apps/api/internal/handler/user_test.go`
- Modify: `apps/api/internal/service/auth.go` (`UserJSON` — add optional pinned)

**Interfaces:**
- Route: `PATCH /api/v1/users/:id` under existing admin group
- List items include `"role_pinned": true|false`

- [ ] **Step 1: Extend `UserJSON`**

```go
func UserJSON(user db.User, identities []db.UserIdentity) map[string]any {
	return UserJSONPinned(user, identities, false)
}

func UserJSONPinned(user db.User, identities []db.UserIdentity, rolePinned bool) map[string]any {
	out := /* existing fields */
	out["role_pinned"] = rolePinned
	return out
}
```

Prefer always including `role_pinned` on list/PATCH responses for UI simplicity. For `GET /auth/me`, either omit or set via `IsAdminEmail` — set correctly using `cfg` where available; if `/auth/me` cannot easily access cfg without churn, leave `role_pinned` only on `/users` responses (handler builds map without changing me).

**Decision for implementer:** include `role_pinned` only in `UserHandler` responses (list + patch), not in `/auth/me`, to avoid Authed UserJSON churn:

```go
func userAdminJSON(h *UserHandler, user db.User, identities []db.UserIdentity) map[string]any {
	out := UserJSON(user, identities)
	out["role_pinned"] = h.users.IsRolePinned(user.Email)
	return out
}
```

- [ ] **Step 2: Failing handler tests**

Cover: admin PATCH 200; member 403; last_admin 409; admin_emails_pinned 409; list includes `role_pinned`.

- [ ] **Step 3: Implement handler**

```go
func (h *UserHandler) Register(r gin.IRouter) {
	api := r.Group("/api/v1")
	api.Use(middleware.RequireSession(h.auth), middleware.RequireAdmin())
	api.GET("/users", h.listUsers)
	api.PATCH("/users/:id", h.updateUserRole)
}

type updateRoleBody struct {
	Role string `json:"role"`
}

func (h *UserHandler) updateUserRole(c *gin.Context) {
	actor, ok := middleware.UserFromContext(c)
	if !ok {
		WriteError(c, apperrors.Unauthorized("missing session"))
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		WriteError(c, apperrors.Validation("invalid user id", nil))
		return
	}
	var body updateRoleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		WriteError(c, apperrors.Validation("invalid body", nil))
		return
	}
	user, err := h.users.UpdateUserRole(c.Request.Context(), actor.ID, id, body.Role)
	if err != nil {
		WriteError(c, err)
		return
	}
	identities, _ := /* load identities if needed for response consistency */
	WriteJSON(c, http.StatusOK, userAdminJSON(h, user, identities))
}
```

For identities on PATCH: either return user fields only, or load via a small `ListUserIdentities` on an extended repo. Prefer loading identities the same way as list for one shape.

Update `listUsers` to use `userAdminJSON`.

- [ ] **Step 4: Run handler tests**

Run: `go test ./apps/api/internal/handler/ -run User -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/handler/user.go apps/api/internal/handler/user_test.go
git commit -m "feat: add PATCH /users/{id} role update and role_pinned"
```

---

### Task 7: Users page UI

**Files:**
- Create: `apps/web/src/pages/UsersPage.tsx`
- Create: `apps/web/src/pages/UsersPage.test.tsx`
- Create: `apps/web/src/pages/UsersPage.stories.tsx` (if page-level stories exist elsewhere; else Storybook for a `UserRoleSelect` shared control under `apps/web/src/components/users/`)
- Create: `apps/web/src/lib/usersApi.ts`
- Modify: `apps/web/src/App.tsx`
- Modify: `apps/web/src/components/layout/AppShell.tsx`
- Modify: `apps/web/src/locales/en/common.json`
- Modify: `apps/web/src/locales/ru/common.json`
- Modify: `apps/web/src/components/layout/AppShell.test.tsx`
- Modify: `docs/04-api-spec.md`

**UI pattern:** Follow `TeamsPage` — `PageHeader`, `DataTable`, `Select`, `Toast`, `Banner`. Admin-only route: if `user.role !== 'admin'`, show forbidden banner (or Navigate to `/shifts`).

- [ ] **Step 1: i18n keys (en + ru)**

```json
"nav.users": "Users",
"users.page_title": "Users",
"users.page_subtitle": "Manage access roles",
"users.col.name": "Name",
"users.col.email": "Email",
"users.col.role": "Role",
"users.role.admin": "Admin",
"users.role.member": "Member",
"users.role.viewer": "Viewer",
"users.pinned": "Pinned by config",
"users.role_updated": "Role updated",
"users.load_error": "Could not load users. Check your connection and try again.",
"users.forbidden": "Only admins can manage users."
```

Russian: faithful translations (e.g. `Пользователи`, `Роль обновлена`, `Закреплено конфигурацией`).

- [ ] **Step 2: `usersApi.ts`**

```ts
export type ListedUser = {
  id: string;
  email: string;
  display_name: string;
  role: 'admin' | 'member' | 'viewer';
  role_pinned?: boolean;
};

export async function fetchUsers(q = ''): Promise<{ items: ListedUser[] }> { /* GET /api/v1/users */ }
export async function patchUserRole(id: string, role: string): Promise<ListedUser> { /* PATCH */ }
```

Parse API error `message` for toasts.

- [ ] **Step 3: Failing Vitest**

`UsersPage.test.tsx`: mock fetch list; change role select; expect PATCH + success toast; pinned user demote shows API error message.

- [ ] **Step 4: Implement page + wire route/nav**

- `AppPage` union add `'users'`
- `pageFromPath`: `/users` → `users`
- Route: `/users` behind `ProtectedRoute`
- Nav: include Users **only when** `user?.role === 'admin'` (filter `navItems`)
- Disable role `<Select>` when `role_pinned`; show muted `users.pinned` text

- [ ] **Step 5: Run web tests**

Run: `cd apps/web && npm test -- --run UsersPage AppShell`

Expected: PASS.

- [ ] **Step 6: API spec**

In `docs/04-api-spec.md` users section:

```markdown
| PATCH | `/users/{id}` | session + admin | Body `{"role":"admin"|"member"|"viewer"}`. Conflicts: `last_admin`, `admin_emails_pinned`. |
```

Note list items include `role_pinned` boolean.

- [ ] **Step 7: Commit**

```bash
git add apps/web docs/04-api-spec.md
git commit -m "feat: add admin Users page for role management"
```

---

### Task 8: Phase 2 backlog + full gate + PR

**Files:**
- Modify: `backlog/epics/EPIC-09-teams-users-shifts.md`

- [ ] **Step 1: Append story** (e.g. `AEG-073`)

```markdown
### AEG-073 — Admin user role management

- **Status:** Ready
- **Depends on:** AEG-072, AEG-065
- **PRD:** REQ-AUTH-04, REQ-AUDIT-01
- **Acceptance:**
  - [ ] `PATCH /api/v1/users/{id}` with last-admin + ADMIN_EMAILS pin guards + audit
  - [ ] `GET /users` includes `role_pinned`
  - [ ] `/users` admin UI with en/ru copy + tests
  - [ ] API spec updated
```

- [ ] **Step 2: Full gate**

Run: `make lint type test`

Expected: green.

- [ ] **Step 3: Commit backlog + open PR 2**

```bash
git add backlog/epics/EPIC-09-teams-users-shifts.md
git commit -m "docs: add AEG-073 admin role management story"
```

Branch: `feat/users-AEG-073-role-management`

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `ADMIN_EMAILS` parse / fail invalid | Task 1 |
| OIDC promote + audit + no-op | Task 3 |
| DevAuth unchanged | Task 3 (explicit non-touch) |
| Setup + security + env examples | Tasks 1, 4 |
| Interim SQL | Task 4 |
| `UpdateUserRole` store | Task 2 |
| PATCH + last_admin + pin | Tasks 5–6 |
| `role_pinned` | Task 6–7 |
| Users UI + i18n | Task 7 |
| Two-PR rollout | Phase 1 / Phase 2 headers |
| Audit reasons `admin_emails_env` / `admin_api` | Tasks 3, 5 |

## Self-review notes

- No TBD/placeholder steps remain.
- `UserRepository` / constructor wiring must be updated wherever `NewUserService` is called — search `NewUserService(` during Task 5.
- Handler WriteError must already map custom `apperrors.Error.Code` through to JSON (verify existing `WriteError`; if it always emits generic CONFLICT, ensure `Code` field is preserved for `last_admin` / `admin_emails_pinned`).
