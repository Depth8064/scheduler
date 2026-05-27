# Workstation Flows

## Workstation Dashboard

1. User logs in and is presented with the queue for their authorized workstation(s).
2. If a user is authorized for multiple workstations, they select which one to view.
3. Queue is ordered by sort_order, with priority as a secondary visual indicator.
4. Items in unreleased status are not shown to workstation users.

## Execution Actions

1. **Start item** (queued → in_progress): marks the job as actively being worked. Cannot start if item is blocked or not queued.

2. **Pause item** (in_progress → paused): temporarily stop work while keeping the job in an active state. Can be resumed later. Useful when waiting for material, tooling, or handling interruptions.

3. **Resume item** (paused → in_progress): restart a paused job. Only valid from paused status.

4. **Mark blocked** (in_progress or paused → blocked): flag the job as unable to proceed. Reason (free-text) is recorded in the event. Examples: "Waiting for burner maintenance", "Material not received", "Tool broken". Admin must unblock via the admin API before work can continue.

5. **Update progress** (any status → same status): record partial qty updates without closing the item. Call this endpoint multiple times as work progresses. Each call appends a progress_updated event, preserving the full time-series.
   - Request: {qty_complete, qty_remaining, scrap_delta (optional)}
   - Must satisfy: qty_complete + qty_remaining + scrap_delta ≤ qty_required
   - Does NOT change the item status; item remains in_progress or paused
   - Does NOT require acknowledgment

6. **Complete item** (in_progress or paused → completed): finalize the job with final qty values. Item enters completed status.
   - Request: {qty_complete, qty_remaining (optional), scrap_delta (optional), partial_acknowledged (boolean)}
   - If qty_remaining > 0: indicates partial completion (not all planned qty was completed). Caller MUST set partial_acknowledged=true to confirm this is intentional. Rejects with 422 if qty_remaining > 0 but partial_acknowledged is false.
   - If qty_remaining == 0 (or omitted): full completion; partial_acknowledged not required.
   - Do NOT call /complete with qty_remaining > 0 unless truly partial. Use /progress for intermediate updates.

7. **Send to next operation** (metadata event): record that the item was sent to the next workstation/process (e.g. Burner → Laser → Finishing). Optional destination note (e.g. "Laser station", "Press area"). Does not close or change status; item remains in its current status (typically in_progress or paused). Use this before or after completion depending on workflow.

8. **Record metal finishing date** (metadata event): flag that the item has been sent to metal finishing. Similar to send-next; does not change item status. The timestamp of this event serves as the "Metal Finishing" date from the legacy spreadsheet.

## Validation

1. State transitions must follow the allowed transition map in the data model.
2. qty_complete on a progress update cannot exceed qty_required.
3. Completing an item with qty_remaining > 0 requires explicit acknowledgment (partial completion).
4. Unauthorized workstation access returns HTTP 403.
5. Actions on items not assigned to the user's workstation return HTTP 403.
