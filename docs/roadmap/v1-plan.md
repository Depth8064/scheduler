# V1 Delivery Plan

## Milestone 1

1. Canonical docs complete.
2. App skeleton compiles.
3. DB migrations bootstrap core schema (all 10 entities including the `released` state).
4. Shared logging manager integrated as the only logging path.
5. Startup config uses logging.DefaultConfig with explicit environment overrides.
6. Docker-first runtime scaffolding: production Dockerfile + local docker compose baseline.
7. CI skeleton established with lint, vet, and `go test ./...` required checks.

## Milestone 2

1. Local auth and role enforcement; user management endpoints (CRUD).
2. Workstation scoping API foundation.
3. Manual JobBOSS sync endpoint; PO line inbox (promote to schedule item).
4. Kanban items import endpoint (CSV, from xlsx reference sheets).
5. CSRF middleware wired to all mutating routes.
6. Integration test harness running against containerized Postgres in CI.

## Milestone 3

1. Admin scheduling board APIs (release, assign, reorder, bulk, requeue, unassign).
2. Workstation execution APIs (start, pause, resume, block, progress, complete, send-next, metal-finishing).
3. Execution event timeline APIs.
4. CI enforces migration-apply validation and auth/RBAC regression test suite.

## Milestone 4

1. Web UI workflows complete.
2. Integration tests and pilot validation.
3. Production readiness checklist.
4. Docker image release path complete (immutable tags, smoke-tested image artifact).
5. Release gate policy active: all required CI stages green before release.

## Reuse Targets and Landing Zones

Use this section as the implementation queue for pulling proven pieces from Aribra to JobBOSS Connector and PrintMaster.

### Milestone 2: Auth and Security Foundation

1. Local auth/session manager scaffold (adapt from Aribra).
	- Source: `Aribra to JobBOSS Connector/internal/auth/auth.go` (`Manager`, `Middleware`, `RequireRole`, session/context helpers).
	- Source: `Aribra to JobBOSS Connector/internal/api/auth_handlers.go` (`/auth/api/login`, `/auth/api/logout`, `/auth/api/session`, route registration pattern).
	- Destination: `scheduler/internal/auth/manager.go` (new), `scheduler/internal/httpapi/auth_handlers.go` (new), `scheduler/internal/httpapi/router.go` (wire routes + middleware).

2. OIDC login initiation/callback with proper token verification (adapt from PrintMaster, not Aribra's simplified parsing path).
	- Source: `printmaster/server/oidc_handlers.go` (`handleOIDCStart`, `handleOIDCCallback`, `buildOAuthConfig`, `cachedOIDCProvider`, `resolveOIDCUser`).
	- Source: `printmaster/server/main.go` (public OIDC route registration + web auth route protection patterns).
	- Destination: `scheduler/internal/auth/oidc.go` (new), `scheduler/internal/httpapi/oidc_handlers.go` (new), `scheduler/internal/httpapi/router.go` (public callback/start + protected API wiring).

3. Session cookie hardening and request auth guard.
	- Source: `printmaster/server/main.go` (`requireWebAuth`, `createSessionCookie`, `requestIsHTTPS`, cookie flags).
	- Source: `Aribra to JobBOSS Connector/internal/api/auth_handlers.go` (cookie lifecycle on login/logout).
	- Destination: `scheduler/internal/httpapi/auth_middleware.go` (new), `scheduler/internal/httpapi/auth_handlers.go` (new).

4. Auth/OIDC persistence schema additions.
	- Source: `printmaster/server/storage/migrations/0003_add_oidc_tables.sql` (`oidc_providers`, `oidc_sessions`, `oidc_links`).
	- Source: `Aribra to JobBOSS Connector/internal/storage/sqlite/sqlite.go` (user/session CRUD method shape).
	- Destination: `scheduler/internal/db/migrations/002_auth_oidc.up.sql` (new), `scheduler/internal/db/migrations/002_auth_oidc.down.sql` (new), `scheduler/internal/store/auth_store.go` (new).

### Milestone 3: Admin and Role Enforcement Integration

1. Role-based route protection and admin/user boundaries.
	- Source: `Aribra to JobBOSS Connector/internal/auth/auth.go` (`RequireRole`, `isPublicPath`, unauthorized handling semantics).
	- Source: `printmaster/server/main.go` (`requireWebAuth` usage on admin APIs).
	- Destination: `scheduler/internal/httpapi/router.go` (route groups), `scheduler/internal/httpapi/admin_handlers.go` (new), `scheduler/internal/httpapi/workstation_handlers.go` (new).

2. User management endpoint structure for admin flows.
	- Source: `Aribra to JobBOSS Connector/internal/api/auth_handlers.go` (`handleUsers`, `handleUser`, request/response contracts).
	- Destination: `scheduler/internal/httpapi/admin_users_handlers.go` (new), `scheduler/docs/api/openapi.yaml` (update to match behavior).

### Milestone 4: Web UI and Theme System

1. Theme toggle UI, behavior, and visual language.
	- Source: `printmaster/server/web/index.html` (toggle markup in header).
	- Source: `printmaster/server/web/app.js` (`initThemeToggle`, localStorage persistence, `body.light-mode` class switching).
	- Source: `printmaster/server/web/style.css` (`.theme-toggle*` component styles and animation).
	- Source: `printmaster/common/web/shared.css` (dark/light variable system and global token pattern).
	- Destination: `scheduler/web/static/index.html` (insert toggle), `scheduler/web/static/app.js` (new), `scheduler/web/static/style.css` (new).

2. Login UX and SSO entry patterns.
	- Source: `Aribra to JobBOSS Connector/static/login.html` (local + OIDC UX pattern and break-glass fallback concept).
	- Source: `printmaster/server/web/login.html` and `printmaster/server/web/login.js` (multi-provider options, tenant hint/lookup model).
	- Destination: `scheduler/web/static/login.html` (new), `scheduler/web/static/login.js` (new), `scheduler/internal/httpapi/login_page_handlers.go` (new).

### Pull Rules

1. Prefer Aribra for auth manager shape and endpoint ergonomics.
2. Prefer PrintMaster for OIDC verification correctness and cookie/session hardening.
3. Prefer PrintMaster for theme system and toggle polish.
4. Do not copy Aribra's simplified ID token parsing logic as the final verification strategy.

---

# V2+ Candidate Features

## Throughput Intelligence

Depends on V1 execution history being complete and accurate. No schema changes required — data is already captured in execution_events and execution_progress.

1. Per-workstation throughput baselines derived from historical cycle times.
2. Estimated completion dates per queue item based on actual historical rates.
3. Overload warnings: flag when a workstation queue cannot realistically meet nearest target_date.
4. Bottleneck reporting: recurring block reasons, underperforming part/material combinations.
5. AI-assisted queue suggestions: advisory-only assignment recommendations based on load, throughput history, and target dates. Never auto-assigns.

## Other V2 Candidates

1. Kanban replenishment dashboard: track bin consumption rates against kanban_items parameters and surface reorder signals.
2. Cross-workstation dependency view: when a Burner job feeds a Laser or Saw job, surface the chain.
3. JobBOSS write-back (selected fields, e.g. actual completion date).
