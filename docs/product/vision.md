# Product Vision

## Goal

Replace spreadsheet-based workstation scheduling with a secure web application that:

1. Pulls pending work from JobBOSS in read-only mode.
2. Lets admins assign and reorder schedule items across workstations.
3. Lets workstation users see only their queue and update progress in real time.

## Personas

1. Admin/Planner: manages all schedules, imports new work, handles exceptions.
2. Workstation User: executes assigned work and updates job progress.

## Scope for V1

1. Local account authentication with role-based access (admin, workstation).
2. User management: admin creates/edits accounts and assigns workstations to users.
3. Read-only JobBOSS sync for PO line items; admin promotes PO lines to schedule items.
4. KAN_BAN schedule items created manually by admin (not synced).
5. Scheduling board: release, assign to specific workstation, reorder queue, bulk release+assign.
6. Workstation execution actions: start, pause, resume, block, unblock, partial progress updates, complete, send-next, metal-finishing.
7. Full execution event timeline per schedule item (audit trail, readable via API and UI).
8. Concurrency control via version field on all mutable schedule state.

## Non-Goals for V1

1. Writing back updates to JobBOSS.
2. Automated optimization solver.
3. Native mobile app.
4. Offline-first client sync.

## Future Direction: Throughput Intelligence (V2+)

Because the app owns the full execution event timeline, it will accumulate real throughput data over time — how long each job actually took per workstation, actual qty rates per operator, how often jobs are blocked and why. This opens the door to a planning assistance layer:

1. **Throughput baselines**: derive average and p90 cycle times per part type and workstation from historical execution_events. Show planners how long a queue will realistically take given current load.
2. **Capacity visibility**: given a workstation's current queue and its historical rate, surface an estimated completion date per item. Flag items where the target_date is likely to be missed before work begins.
3. **Queue load warnings**: alert the admin when a workstation's queue is overloaded relative to available time before the nearest target_date.
4. **Pattern recognition**: identify recurring bottlenecks (e.g. a specific part type that consistently blocks, or a workstation that underperforms on certain material specs). Surface as a simple insight report.
5. **AI-assisted scheduling (exploratory)**: once sufficient history exists, use it to suggest optimal assignment of new items — which workstation is best suited given current load, historical throughput for that part type, and upcoming target dates.

### Data Prerequisites

All of the above depend on V1 execution_events being recorded accurately and completely. V1 must capture: start/end timestamps, operator id, qty per update, and block reasons. No additional schema changes are anticipated for the V2 layer — the data will already be there.

### Implementation Approach

The intelligence layer is read-only and additive. It reads execution_events and execution_progress, computes metrics, and exposes them through read-only API endpoints and dashboard widgets. It does not modify schedule state. An AI/LLM integration (if added) would consume the same computed metrics as context and return advisory suggestions — never automatic writes.
