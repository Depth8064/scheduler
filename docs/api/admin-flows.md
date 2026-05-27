# Admin Flows

## User Management

1. View all user accounts.
2. Create a user account (username, password, role: admin or workstation).
3. Edit a user's role or disable the account.
4. Delete a user account.

## Workstation Management

1. View all workstations grouped by station_type.
2. Create workstation (name, station_type).
3. Edit workstation name or disable it.
4. Assign users to one or more workstations (replaces the full list for a workstation).

## PO Line Inbox

1. View synced PO lines not yet promoted to schedule items, filterable by work center.
2. View inactive PO lines (soft-marked as potentially cancelled by sync).
3. Promote a PO line to a schedule item in `unreleased` state.
4. Promote and directly assign to a workstation in one action (creates item in `queued` state).

## KAN_BAN Item Management

1. Import kanban_items reference table from CSV (initial population from xlsx reference sheets).
2. View and edit kanban parameters (bin sizes, weekly qtys, dedicated workstation).
3. Create KAN_BAN schedule items manually (not from PO line sync).

## Schedule Board Operations

1. View all schedule items across all workstations, filterable by station_type, workstation, status, job_type, and job_group.
2. View `released` (unassigned) items separately from `queued` items.
3. Release an item from `unreleased` -> `released`. Item is visible to admin but not to workstation users.
4. Assign a released item to a specific workstation (status: `released` -> `queued`; appends to queue at end).
5. Release and assign in one action (shortcut for the common case).
6. Unassign from a workstation (status: `queued` -> `released`; sort_order cleared).
7. Bulk release and assign: select multiple items, assign to workstation in one action.
8. Reassign item from one workstation to another (remains `queued`; appends to new queue).
9. Reorder items within a workstation queue (updates sort_order for all items in queue).
10. Edit target_date, priority, job_group, and planner_notes on any item.
11. Unblock a blocked item (returns it to `queued`).
12. Re-queue a completed item with a reason (status: `completed` -> `queued`).
13. View the full execution event timeline for any schedule item.

## Ingestion Operations

1. Trigger manual JobBOSS sync.
2. View sync run history and errors.
3. Re-run failed sync after issue resolution.

## Validation

1. All mutating endpoints enforce optimistic concurrency via version field.
2. Reorder endpoint takes a queue-level version to prevent concurrent reorder conflicts.
3. Missing or stale version returns HTTP 409.
4. Non-admin access to admin routes returns HTTP 403.
5. Assigning to a disabled workstation returns HTTP 422.
6. Promoting a PO line that already has a schedule item returns HTTP 409.

## Implementation Targets (Planned Reuse)

Use this as a quick index for where admin-flow implementation pieces will be pulled from and where they will land.

1. Auth guard + role checks for admin endpoints.
	- Pull from: `Aribra to JobBOSS Connector/internal/auth/auth.go` (`Middleware`, `RequireRole`).
	- Pull from: `printmaster/server/main.go` (`requireWebAuth` protection pattern on admin APIs).
	- Land in: `scheduler/internal/httpapi/router.go`, `scheduler/internal/httpapi/auth_middleware.go`.

2. Admin user CRUD endpoint structure.
	- Pull from: `Aribra to JobBOSS Connector/internal/api/auth_handlers.go` (`handleUsers`, `handleUser`).
	- Land in: `scheduler/internal/httpapi/admin_users_handlers.go`, `scheduler/docs/api/openapi.yaml`.

3. OIDC login flow for admin UI access.
	- Pull from: `printmaster/server/oidc_handlers.go` (`handleOIDCStart`, `handleOIDCCallback`, `resolveOIDCUser`).
	- Pull from: `printmaster/server/main.go` (OIDC route wiring and session cookie behavior).
	- Land in: `scheduler/internal/httpapi/oidc_handlers.go`, `scheduler/internal/auth/oidc.go`, `scheduler/internal/db/migrations/002_auth_oidc.up.sql`.

4. Theme toggle and login UX used by admin pages.
	- Pull from: `printmaster/server/web/index.html`, `printmaster/server/web/app.js`, `printmaster/server/web/style.css`.
	- Pull from: `printmaster/server/web/login.html`, `printmaster/server/web/login.js`, `Aribra to JobBOSS Connector/static/login.html`.
	- Land in: `scheduler/web/static/index.html`, `scheduler/web/static/style.css`, `scheduler/web/static/app.js`, `scheduler/web/static/login.html`, `scheduler/web/static/login.js`.

See `docs/roadmap/v1-plan.md` for the milestone-level sequence.
