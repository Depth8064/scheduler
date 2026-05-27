# Data Model

## Core Entities

1. users: app accounts and role metadata.
2. workstations: specific machines or execution lanes. Fields: id, name (e.g. "Burner 1"), station_type (e.g. "Burner", "Laser", "Saw"), active. station_type is the dept-level grouping; name is the specific machine.
3. user_workstation_access: maps users to one or more workstations they are authorized to view and act on.
4. jobboss_po_lines: read-only mirrored PO line items from JobBOSS.
5. schedule_items: schedulable work units. Each row links to a jobboss_po_lines row and optionally to a specific workstation. Fields include: workstation_id (nullable = unassigned), sort_order (integer position within workstation queue), status, job_type, job_group, part_number, part_description, material_spec, external_job_number, receiving_reference, planned_start_date, required_date, target_date, priority, planner_notes.
6. execution_progress: live qty state per schedule_item. Fields: schedule_item_id, qty_required, qty_complete, qty_remaining, scrap_delta, updated_by (user_id), updated_at.
7. execution_events: immutable timeline entries for all state transitions and operator actions. Fields: schedule_item_id, event_type, actor_id, occurred_at, notes.
8. kanban_items: reference parameters for recurring Kanban parts. Fields: part_number, part_description, units_per_assembly, monthly_qty, weekly_qty, bin_qty, total_bins, total_kanban_qty, bin_type_notes, dedicated_workstation_id.
9. sync_checkpoints: per-resource delta sync cursor.
10. sync_runs: audit log for each ingestion run.

## Key Decisions

1. Primary schedulable unit is a PO line item from JobBOSS. One schedule_item per PO line.
2. External mirror records (jobboss_po_lines) are read-only. Schedule state is owned locally and does not write back to JobBOSS.
3. Workstations are specific machines (Burner 1, Burner 2, Burner 3), not departments. station_type on the workstation record provides the dept-level grouping for admin views.
4. schedule_items.workstation_id is nullable. Null means the item is either unreleased or released-but-unassigned. The status field always reflects the correct state — workstation_id alone is not sufficient to determine visibility.
5. schedule_items.sort_order is an integer position within the assigned workstation queue. Reordering updates sort_order values. When an item is assigned to a new workstation, it appends to the end of that queue (highest sort_order + 1). Unassigned items have sort_order = null.
6. job_type is KAN_BAN or OPEN. KAN_BAN jobs have a string job number ("KAN BAN") and link to a kanban_items row. OPEN jobs have a numeric external_job_number. KAN_BAN schedule_items are created manually by the admin, not via PO line sync — they have no corresponding jobboss_po_lines row (foreign key is nullable for this case).
7. job_group is a free-text display grouping set by the admin (e.g. "Stock Jobs", "Uprights & Guards", "Open Jobs"). Used for visual section headers on the board; does not affect logic.
8. material_spec is free-text. Burner/Laser use stock codes (e.g. 205-020); Saw uses raw stock dimensions (e.g. 1/2" X 4-1/2"). No normalization enforced.
9. target_date is the station-agnostic field for the due/burn/cut date. UI can render a station-specific label based on the workstation's station_type.
10. qty_remaining on execution_progress is stored explicitly, not computed. It may differ from qty_required - qty_complete due to mid-run adjustments.
11. Every execution_progress write records updated_by (user_id). execution_events captures all state transitions with actor_id and occurred_at. Each progress_updated event row is kept permanently — execution_progress is only the current snapshot, events are the full history.
12. priority is an integer on schedule_items: 0=normal, 1=overdue, 2=machine issue delay, 3=less than lead time, 4=stock issue. Derived from spreadsheet color-code conventions but set explicitly here.
13. All schedule_items rows include created_at and updated_at timestamps. The version column is used for optimistic concurrency on all mutating operations including progress updates.
14. kanban_items is populated by admin via manual import (initially from xlsx reference sheets). No automated sync source exists for this data.

## Schedule State Model

Allowed states:

1. unreleased: synced from JobBOSS but not yet released to the floor. Not visible to workstation users.
2. released: admin has released the item but it is not yet assigned to a specific workstation. Visible in the admin unassigned queue; not visible to workstation users.
3. queued: released and assigned to a specific workstation. Appears in that workstation's queue, ordered by sort_order.
4. in_progress: operator has started the job.
5. paused: operator paused mid-run.
6. blocked: cannot proceed; reason recorded on the blocking event.
7. completed: all work finished; closed by operator or admin.

Allowed transitions:
- unreleased -> released (admin: release action)
- released -> queued (admin: assign to workstation)
- unreleased -> queued (admin: release + assign in one action)
- queued -> released (admin: unassign from workstation)
- queued -> in_progress (operator: start)
- in_progress -> paused (operator: pause)
- in_progress -> blocked (operator: block with reason)
- in_progress -> completed (operator: complete)
- paused -> in_progress (operator: resume)
- paused -> blocked (operator: block while paused)
- blocked -> queued (admin: unblock and reassign or return to queue)
- completed -> queued (admin: re-queue, e.g. partial completion discovered after close)
- any -> unreleased is not allowed after release.

Note: `released` replaces the ambiguity of `queued` with null workstation_id. An item is always in a well-defined state that reflects both its release status and its assignment status.

## Execution Event Types

1. released: admin released item to floor (status: unreleased -> released).
2. assigned: admin assigned or reassigned to a workstation (status: released -> queued, or queued -> queued).
3. unassigned: admin removed workstation assignment (status: queued -> released).
4. reordered: admin changed sort_order in queue.
5. started: operator began work (status: queued -> in_progress).
6. paused: operator paused (status: in_progress -> paused).
7. resumed: operator resumed (status: paused -> in_progress).
8. blocked: operator marked blocked with reason (status: in_progress or paused -> blocked).
9. unblocked: admin cleared block (status: blocked -> queued).
10. progress_updated: operator recorded qty_complete and qty_remaining. Does not change status. Each update is a separate event row so the time-series is preserved for throughput analysis.
11. metal_finishing: item sent to metal finishing; date recorded.
12. sent_next: item forwarded to next operation/press area; destination note recorded.
13. completed: item marked done (status: in_progress -> completed). Final qty values recorded on the event.
14. requeued: admin re-opened a completed item (status: completed -> queued).
15. notes_updated: planner updated planner_notes.

Note on `execution_progress` vs events: execution_progress holds the current live qty snapshot (latest state). execution_events row with type `progress_updated` or `completed` stores each individual qty submission so history is never lost. V2 throughput analysis reads the event rows, not the progress snapshot.

## Concurrency Model

1. schedule_items includes version column for optimistic concurrency.
2. Update APIs require expected version; conflicts return HTTP 409.
