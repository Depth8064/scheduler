# UX Section Flows (V1)

This document defines complete section flows for the current Scheduler web UI. The goal is to finish each section end-to-end before moving to the next section.

## UX Build Rules

1. Do not ship placeholder actions in production paths.
2. Each section must have full state coverage before marked complete.
3. API integration, validation, loading, empty, and error states are part of "done".
4. Role boundaries must be explicit in navigation, route protection, and action controls.
5. Every mutating action must have optimistic feedback and conflict handling.

## Section Inventory

1. Login (`/login`)
2. Dashboard (`/dashboard`, admin)
3. Users (`/users`, admin)
4. Workstations (`/workstations`, admin)
5. Admin Operations (`/admin`, admin)
6. Operator Console (`/operator`, workstation)
7. Shared Shell (header, sidebar auth card, health chip)

---

## 1) Login Flow

### User Goal

Authenticate quickly and land on the correct home page for role.

### Entry

1. User opens `/login`.
2. If session exists, redirect immediately:
   - admin -> `/dashboard`
   - workstation -> `/operator`

### Primary Flow

1. User enters username/password.
2. Submit button enters loading state (disabled, text changes to "Signing in...").
3. API call: `POST /api/v1/auth/login`.
4. On success, fetch `GET /api/v1/auth/me`.
5. Redirect by role.

### State Coverage

1. Loading: form locked while request is in flight.
2. Invalid credentials: inline error near form, preserve username.
3. System error (500/network): retry guidance + non-destructive message.
4. Session mismatch: if `/me` fails after successful login, show session recovery message with retry action.

### Done Criteria

1. Keyboard-only sign-in works.
2. Enter key submits once.
3. No silent failures.
4. Clear success/failure feedback under 1 second after response.

---

## 2) Dashboard Flow (Admin Home)

### User Goal

Understand system status and quickly jump to pending operational actions.

### Entry

1. Admin lands on `/dashboard` after login.
2. Shared shell validates session and role.

### Primary Flow

1. See top-level status cards:
   - Service health
   - Pending PO lines awaiting promotion
   - Released items awaiting assignment
   - Blocked in-progress items
2. Click any card to deep-link to filtered section in Admin Operations.
3. See recent activity feed (latest execution/admin events).

### State Coverage

1. Loading skeleton cards.
2. Empty activity history state.
3. Partial data available (health ok, one widget failed): render partial results and isolate error to failed widget.
4. Role drift (session changed): redirect to role home.

### Done Criteria

1. Every card has a meaningful action.
2. No dead-end informational panels.
3. Status refresh pattern is clear (manual refresh now, auto-refresh optional next).

---

## 3) Users Flow (Admin)

### User Goal

Create, update, deactivate, and remove users safely.

### Entry

1. Admin opens `/users`.
2. User table loads immediately with filter/search bar.

### Primary Flows

1. Create user
   - Fill username/password/role/active.
   - Submit -> optimistic pending row or loading state.
   - Success toast + row appears in sorted list.
2. Edit role/active
   - Inline action opens compact confirm state.
   - Submit patch with concurrency version (when backend supports).
3. Delete user
   - Two-step confirmation with username text.
   - Prevent self-delete.

### State Coverage

1. Empty list: CTA to create first user.
2. Validation errors (username exists, weak password, invalid role).
3. Authorization error (403): page-level guard message and action to return dashboard.
4. Conflict (409): prompt reload with "This user changed since you loaded the page."

### Done Criteria

1. All destructive actions require explicit confirmation.
2. Mutations preserve list scroll/filter context.
3. User receives clear status for each mutation.

---

## 4) Workstations Flow (Admin)

### User Goal

Manage workstation lifecycle and assignment access with confidence.

### Entry

1. Admin opens `/workstations`.
2. Grouped list by `station_type` with counts (active/inactive).

### Primary Flows

1. Create workstation
   - Name + station type + active state.
2. Update workstation
   - Enable/disable toggle.
   - Rename inline or in edit drawer.
3. Manage access
   - Open assignment panel for selected workstation.
   - Show all users with role badge and active status.
   - Save full replacement list.
4. Delete workstation
   - Block delete when schedule items are attached; show dependency guidance.

### State Coverage

1. No workstations yet: guided empty state.
2. No assignable users: assignment panel explains why.
3. Save assignment success/failure state in-panel without losing selection.
4. Disabled workstation assignment attempt returns actionable error.

### Done Criteria

1. Access assignment is understandable at a glance.
2. Station grouping improves scan speed.
3. Dangerous operations are explicit and reversible where possible.

---

## 5) Admin Operations Flow (Scheduling + Ingestion)

### User Goal

Run all schedule operations from one operational workspace without context switching.

### Section Layout

1. PO Line Inbox
2. Schedule Board
3. Kanban Management
4. Sync Operations
5. Execution Timeline Drawer

### Primary Flows

1. PO Inbox
   - Load unpromoted lines with filters.
   - Promote to `unreleased`.
   - Promote + assign in one action.
2. Schedule Board
   - Filter by status, station type, workstation, job type, job group.
   - Release item.
   - Assign/unassign/reassign.
   - Reorder queue (drag or explicit move up/down).
   - Bulk release+assign.
   - Edit metadata (target date, priority, group, notes).
3. Kanban
   - CSV import.
   - Edit reference row.
   - Create manual KAN_BAN schedule item.
4. Sync
   - Trigger manual sync.
   - View run history.
   - Retry failed runs.
5. Timeline
   - Open item detail drawer with full execution/admin event feed.

### State Coverage

1. Queue-level reorder conflict (`409`) with reload + replay guidance.
2. Stale version mutation conflict (`409`) with row-level refresh.
3. Bulk action partial failure: show per-item success/failure count.
4. Empty inbox/board states with next actions.
5. Sync failure state includes reason and retry CTA.

### Done Criteria

1. All admin flows from `docs/api/admin-flows.md` are represented in UI.
2. Every mutation exposes pending/success/error feedback.
3. No operation requires manual page refresh.

---

## 6) Operator Console Flow

### User Goal

Run assigned work quickly with minimal clicks and clear status.

### Entry

1. Workstation user opens `/operator`.
2. If multiple assigned workstations, select active station via station picker.
3. Queue loads ordered by `sort_order`.

### Primary Flows

1. Start item (`queued -> in_progress`)
2. Pause/resume (`in_progress <-> paused`)
3. Block with reason (`in_progress|paused -> blocked`)
4. Progress update (qty complete/remaining/scrap)
5. Complete item (with partial acknowledgment support)
6. Send to next operation (metadata event)
7. Record metal finishing date (metadata event)

### State Coverage

1. No assigned stations.
2. No queued items for selected station.
3. Forbidden action when item is not on assigned station (`403`).
4. Invalid transition (`422`) with guidance on valid next actions.
5. Partial completion without acknowledgment rejected with explicit correction path.

### Done Criteria

1. Most common operation (start -> progress -> complete) can be done without page navigation.
2. Current item state is always visually obvious.
3. Error copy tells operator exactly what to do next.

---

## 7) Shared Shell Flow

### User Goal

Always know auth state, app health, and available navigation.

### Primary Behavior

1. Header nav only shows links allowed by role.
2. Sidebar auth card shows username, role, station scope, sign out.
3. Health chip updates every 30 seconds.

### State Coverage

1. Health endpoint unavailable (show degraded state, do not break page).
2. Auth expired during use (sign-out flow and redirect to login).
3. CSRF token missing on mutating actions (friendly retry guidance).

### Done Criteria

1. Shared shell behavior is consistent across all protected pages.
2. Role confusion is impossible from navigation.

---

## Vertical Slice Implementation Order

Implement in this order to avoid partial UX drift:

1. Login + Shared Shell completeness
2. Users section end-to-end
3. Workstations section end-to-end
4. Operator console end-to-end (using currently available execution APIs as they land)
5. Admin Operations end-to-end
6. Dashboard finalization as command center over completed sections

Each slice includes:

1. UI layout + interaction states
2. API wiring
3. Error/empty/loading handling
4. Tests (handler + UI integration where available)
5. Documentation updates

---

## Current Gap Snapshot (as of 2026-05-28)

1. Implemented now:
   - Auth login/logout/me
   - User CRUD page baseline
   - Workstation CRUD + assignment baseline
   - Role-aware navigation shell
2. Partially implemented:
   - Admin page (buttons only, no operational workspace)
   - Operator page (assignment display only, progress form is placeholder)
   - Dashboard (status info only, no operational widgets)
3. Not yet implemented in UI:
   - PO inbox, kanban tools, schedule board, reorder/bulk flows, sync history, timeline drawer

Use this gap snapshot as the planning baseline; do not add new UI sections until one existing section reaches done criteria.