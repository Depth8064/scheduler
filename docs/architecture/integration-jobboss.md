# JobBOSS Integration

## Policy

1. Integration is read-only for V1.
2. Scheduler does not create, update, or delete records in JobBOSS.

## Source Library

Use go-jobboss2-api as the integration client.

## Resources to Sync

1. Purchase orders and PO line items.
2. Related order metadata needed for planning context.
3. Work center metadata used for workstation mapping. The mapping from JobBOSS work center to scheduler workstation is configured as a static admin-managed table in the app, not auto-derived. Admin maps each JobBOSS work center ID to a workstation record.

## PO Line to Schedule Item Promotion

Synced PO lines do not automatically create schedule_items. The admin reviews synced PO lines and explicitly promotes selected lines to schedule_items. This keeps the scheduling board clean and allows the admin to filter out noise (e.g. lines for workcenters out of scope, already-complete work, etc.).

Promotion creates a schedule_item in the `unreleased` state linked to the jobboss_po_lines row by a stable composite key (purchase order number + line number).

## Cancellation Handling

Incremental sync uses LastModDate cursor and cannot detect hard-deleted or cancelled records. To handle this:

1. On each full sync run, compare the set of active jobboss_po_lines keys to the full set from JobBOSS. Records no longer returned are soft-marked as inactive.
2. If a jobboss_po_lines row is marked inactive and its linked schedule_item is in `unreleased` or `released` status, the admin is surfaced an alert to decide whether to cancel or keep the item.
3. schedule_items in `in_progress`, `paused`, or `blocked` status are never auto-cancelled; they require explicit admin action.

Full sync should run on a configurable schedule (e.g. nightly) in addition to incremental runs.

## Sync Mechanics

1. Paginate using take/skip with page size 200.
2. Support full sync and incremental sync.
3. Incremental sync uses LastModDate cursor per resource.
4. Upserts must be idempotent using stable external keys.
5. Record every sync run and checkpoint update.

## Failure Handling

1. Respect client retry/backoff behavior.
2. Persist run status as running, completed, or failed.
3. Capture error summaries and affected resource counts.
4. Keep prior checkpoint on failure.
