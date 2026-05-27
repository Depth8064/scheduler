# Current State

The shop currently uses an Excel workbook to plan and track workstation schedules.

## Current Process

1. Planner manually reads pending work and enters schedule rows in Excel.
2. Workstation operators reference shared spreadsheet views.
3. Progress and handoff notes are manually entered in cells.

## Current Pain Points

1. Concurrent edits and accidental overwrite risk.
2. No server-enforced access controls by workstation.
3. Limited change auditability and historical trace.
4. Manual import and mapping from JobBOSS into spreadsheet rows.
5. Job release state is implicit (rows grouped under "Not Released to Floor yet" text with no enforcement).
6. Priority and urgency communicated only through manual cell color coding; no enforcement or alerting.
7. Operator attribution embedded as initials in free-text cells; no structured accountability.
8. Kanban replenishment parameters (bin sizes, weekly qtys, dedicated burner) maintained in a separate sheet with no link enforcement to schedule rows.
9. Separate xlsx file per workstation per week creates version fragmentation with no cross-station visibility.

## Target State

1. System-managed schedule entities and status changes.
2. Role-based workstation-scoped access.
3. Reliable ingest from JobBOSS with idempotent sync.
4. Full timeline audit for schedule and execution updates.
