# Legacy Schedule Reference

The spreadsheet screenshot is stored at [docs/assets/legacy-schedule.png](../assets/legacy-schedule.png).

## Purpose

This artifact captures the operational baseline and terminology used by the shop before migration to the web app.

## Column Mapping Baseline

Columns present across all station types (Burner 1/2/3, Laser, Saw):

1. Start Date -> schedule_items.planned_start_date
2. Ship Date / KB Rec Date -> schedule_items.required_date. "KB Rec Date" applies when job_type = KAN_BAN.
3. Material Thickness / Material Spec -> schedule_items.material_spec. Burner/Laser store stock codes (e.g. 205-020); Saw stores raw dimensions (e.g. 1/2" X 4-1/2"). Free-text field.
4. Job # -> schedule_items.external_job_number. Value is "KAN BAN" for Kanban recurring jobs; numeric string for open jobs.
5. REC# -> schedule_items.receiving_reference
6. Part # and Description -> schedule_items.part_number, schedule_items.part_description. Saw embeds operation spec in description (e.g. "SAW CUT TO: 3.75\" +/- 0.015\"").
7. Total Quantity Required -> execution_progress.qty_required
8. QTY Remaining / QTY Open / QTY TO COMPLETE -> execution_progress.qty_remaining (station naming varies; same concept)
9. QTY Completed -> execution_progress.qty_complete. Legacy format encodes operator initials and date (e.g. 400BH1/7).
10. Burn By Date / Cut By Date -> schedule_items.target_date. Label is workstation-specific; field is shared.
11. Scrap +/- -> execution_progress.scrap_delta. Present on Laser; implied on others.
12. Metal Finishing -> execution_events(event_type=metal_finishing, occurred_at=<date>)
13. Sent to Next Operation / Press Area -> execution_events(event_type=sent_next, occurred_at=<date>)
14. Burned On Date / Closed -> execution_events(event_type=closed, occurred_at=<date>)
15. ME Steel Planning -> schedule_items.planner_notes. Free-text field for material/engineering notes.
16. Priority -> schedule_items.priority. Explicit column on Saw; implied by color code on Burner/Laser.
17. Color Code -> schedule_items.priority. See Key Decisions in data-model.md for mapping.

## Additional Sheets

1. Weekly-Monthly (Burner 1) / PACER TANK (Burner 2): Kanban replenishment run list. Maps to a kanban_run_queue view, not individual schedule_items.
2. 1954 Subs and KB QTYs (Burner 1, 3): Reference table for Kanban part parameters -> kanban_items entity. Columns: part_number, description, units_per_assembly, monthly_qty, weekly_qty, single_bin_qty, kanban_bin_qty, total_bins, total_kanban_qty, bin_type_description, dedicated_workstation.
3. 1954 FORECAST (Burner 2): Monthly/weekly/daily ship forecast by product line. Out of scope for V1.
4. Historical snapshot sheets (Saw, e.g. "3.13 H-14 (157)"): Archived weekly snapshots. App replaces this with execution_events audit trail.

## Notes

1. Color coding in Excel represented priority bands and work buckets. The Burner 2 header row defines the legend explicitly: Red = Past Target Burn Date, Purple = Past Target Burn Date Due to Burner Issues, Yellow = Less Than Lead Time, Blue = Stock Issues. These map to priority levels 1-4 on schedule_items.
2. The web UI will preserve intent through explicit status and priority values instead of implicit cell color.
3. "Not Released to Floor yet" is used in Saw schedules to group jobs not yet active. This requires the unreleased state in the state machine.
4. Operator identity is embedded in QTY Completed values as initials (e.g. BH, SH, RJ, JG, MS, DB). The app must capture operator_id explicitly on execution_progress writes.
5. Kanban job numbers are the string "KAN BAN" rather than a numeric job ID. These need explicit job_type = KAN_BAN to avoid treating them as lookup keys in JobBOSS.
6. Each station file is revised weekly (filename includes rev date). The app replaces this versioning model with live record updates and execution_events history.
