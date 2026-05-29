package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrScheduleItemExists = errors.New("schedule item already exists for po line")
	ErrInvalidTransition  = errors.New("invalid schedule transition")
	ErrStaleVersion       = errors.New("stale schedule version")
	ErrWorkstationInvalid = errors.New("invalid workstation")
	ErrQueueVersion       = errors.New("queue version conflict")
	ErrInvalidQueue       = errors.New("invalid queue reorder payload")
)

type ScheduleListFilter struct {
	Status        *ScheduleItemStatus
	WorkstationID *string
}

// ScheduleStore manages schedule_items persistence.
type ScheduleStore struct {
	db DBTX
}

// NewScheduleStore constructs a ScheduleStore.
func NewScheduleStore(db DBTX) *ScheduleStore {
	return &ScheduleStore{db: db}
}

// List returns a page of schedule items.
func (s *ScheduleStore) List(ctx context.Context, limit, offset int) ([]ScheduleItem, error) {
	return s.ListFiltered(ctx, limit, offset, ScheduleListFilter{})
}

func (s *ScheduleStore) ListFiltered(ctx context.Context, limit, offset int, filter ScheduleListFilter) ([]ScheduleItem, error) {
	query := `
		SELECT id, po_line_id, workstation_id, sort_order, status, job_type, job_group,
			part_number, part_description, material_spec, external_job_number, receiving_reference,
			planned_start_date, required_date, target_date, priority, planner_notes, qty_required,
			version, created_at, updated_at
		FROM schedule_items
		WHERE ($1::text IS NULL OR status = $1)
		  AND ($2::uuid IS NULL OR workstation_id = $2)
		ORDER BY COALESCE(sort_order, 2147483647), created_at DESC
		LIMIT $3 OFFSET $4
	`

	var statusArg *string
	if filter.Status != nil {
		v := string(*filter.Status)
		statusArg = &v
	}

	rows, err := s.db.QueryContext(ctx, query, statusArg, filter.WorkstationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list schedule items: %w", err)
	}
	defer rows.Close()

	items := make([]ScheduleItem, 0)
	for rows.Next() {
		item, scanErr := scanScheduleItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list schedule items: %w", err)
	}

	return items, nil
}

// GetByID loads a schedule item by id.
func (s *ScheduleStore) GetByID(ctx context.Context, id string) (*ScheduleItem, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, po_line_id, workstation_id, sort_order, status, job_type, job_group,
			part_number, part_description, material_spec, external_job_number, receiving_reference,
			planned_start_date, required_date, target_date, priority, planner_notes, qty_required,
			version, created_at, updated_at
		FROM schedule_items
		WHERE id = $1
	`, id)

	item, err := scanScheduleItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get schedule item %s: %w", id, err)
	}

	return item, nil
}

// Create inserts a schedule item.
func (s *ScheduleStore) Create(ctx context.Context, item *ScheduleItem) error {
	if item == nil {
		return fmt.Errorf("nil schedule item")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO schedule_items (
			id, po_line_id, workstation_id, sort_order, status, job_type, job_group,
			part_number, part_description, material_spec, external_job_number, receiving_reference,
			planned_start_date, required_date, target_date, priority, planner_notes, qty_required,
			version, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18,
			$19, $20, $21
		)
	`, item.ID, item.POLineID, item.WorkstationID, item.SortOrder, item.Status, item.JobType, item.JobGroup,
		item.PartNumber, item.PartDescription, item.MaterialSpec, item.ExternalJobNumber, item.ReceivingReference,
		item.PlannedStartDate, item.RequiredDate, item.TargetDate, item.Priority, item.PlannerNotes, item.QtyRequired,
		item.Version, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create schedule item %s: %w", item.ID, err)
	}

	return nil
}

func (s *ScheduleStore) ListUnpromotedPOLines(ctx context.Context, limit, offset int) ([]POLine, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.external_po_number, p.external_part_number, p.external_item_number,
			p.status, p.required_date, p.qty_required, p.raw_payload, p.external_last_mod_date, p.synced_at
		FROM jobboss_po_lines p
		LEFT JOIN schedule_items s ON s.po_line_id = p.id
		WHERE s.id IS NULL
		ORDER BY p.synced_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list po inbox: %w", err)
	}
	defer rows.Close()

	items := make([]POLine, 0)
	for rows.Next() {
		line, scanErr := scanPOLine(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list po inbox: %w", err)
	}

	return items, nil
}

func (s *ScheduleStore) UpsertPOLines(ctx context.Context, lines []POLine) (int, error) {
	if len(lines) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin po import: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	imported := 0
	for _, line := range lines {
		rawPayload := line.RawPayload
		if len(rawPayload) == 0 {
			rawPayload = []byte("{}")
		}
		now := time.Now().UTC()
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO jobboss_po_lines (
				id, external_po_number, external_part_number, external_item_number,
				status, required_date, qty_required, raw_payload, external_last_mod_date, synced_at
			) VALUES (
				$1, $2, $3, $4,
				$5, $6, $7, $8::jsonb, $9, $10
			)
			ON CONFLICT (external_po_number, external_part_number, external_item_number)
			DO UPDATE SET
				status = EXCLUDED.status,
				required_date = EXCLUDED.required_date,
				qty_required = EXCLUDED.qty_required,
				raw_payload = EXCLUDED.raw_payload,
				external_last_mod_date = EXCLUDED.external_last_mod_date,
				synced_at = EXCLUDED.synced_at
		`, line.ID, line.ExternalPONumber, line.ExternalPartNumber, line.ExternalItemNumber,
			line.Status, line.RequiredDate, line.QtyRequired, string(rawPayload), line.ExternalLastModDate, now); err != nil {
			return 0, fmt.Errorf("upsert po line: %w", err)
		}
		imported++
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit po import: %w", err)
	}

	return imported, nil
}

func (s *ScheduleStore) PromotePOLine(ctx context.Context, poLineID string, workstationID *string) (*ScheduleItem, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin promote po line: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existingID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM schedule_items WHERE po_line_id = $1`, poLineID).Scan(&existingID)
	if err == nil {
		return nil, ErrScheduleItemExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check schedule by po line: %w", err)
	}

	poLine, err := s.getPOLineTx(ctx, tx, poLineID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	status := StatusUnreleased
	var sortOrder *int
	if workstationID != nil {
		if err := s.validateWorkstationTx(ctx, tx, *workstationID); err != nil {
			return nil, err
		}
		status = StatusQueued
		next, seqErr := s.nextSortOrderTx(ctx, tx, *workstationID)
		if seqErr != nil {
			return nil, seqErr
		}
		sortOrder = &next
	}

	now := time.Now().UTC()
	item := &ScheduleItem{
		ID:                generateID(),
		POLineID:          &poLine.ID,
		WorkstationID:     workstationID,
		SortOrder:         sortOrder,
		Status:            status,
		JobType:           JobTypeOpen,
		PartNumber:        poLine.ExternalPartNumber,
		ExternalJobNumber: &poLine.ExternalPONumber,
		RequiredDate:      poLine.RequiredDate,
		Priority:          0,
		QtyRequired:       poLine.QtyRequired,
		Version:           1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err = s.createTx(ctx, tx, item); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit promote po line: %w", err)
	}

	return item, nil
}

func (s *ScheduleStore) Release(ctx context.Context, id string) (*ScheduleItem, error) {
	item, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status != StatusUnreleased {
		return nil, ErrInvalidTransition
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE schedule_items
		SET status = $2, workstation_id = NULL, sort_order = NULL, version = version + 1, updated_at = NOW()
		WHERE id = $1
	`, id, StatusReleased)
	if err != nil {
		return nil, fmt.Errorf("release schedule item %s: %w", id, err)
	}

	return s.GetByID(ctx, id)
}

func (s *ScheduleStore) Assign(ctx context.Context, id, workstationID string) (*ScheduleItem, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin assign schedule item: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item, err := s.getByIDTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if item.Status != StatusReleased && item.Status != StatusUnreleased {
		return nil, ErrInvalidTransition
	}

	if err := s.validateWorkstationTx(ctx, tx, workstationID); err != nil {
		return nil, err
	}

	next, err := s.nextSortOrderTx(ctx, tx, workstationID)
	if err != nil {
		return nil, err
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE schedule_items
		SET status = $2, workstation_id = $3, sort_order = $4, version = version + 1, updated_at = NOW()
		WHERE id = $1
	`, id, StatusQueued, workstationID, next); err != nil {
		return nil, fmt.Errorf("assign schedule item %s: %w", id, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit assign schedule item: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *ScheduleStore) Unassign(ctx context.Context, id string) (*ScheduleItem, error) {
	item, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status != StatusQueued {
		return nil, ErrInvalidTransition
	}

	if _, err = s.db.ExecContext(ctx, `
		UPDATE schedule_items
		SET status = $2, workstation_id = NULL, sort_order = NULL, version = version + 1, updated_at = NOW()
		WHERE id = $1
	`, id, StatusReleased); err != nil {
		return nil, fmt.Errorf("unassign schedule item %s: %w", id, err)
	}

	return s.GetByID(ctx, id)
}

func (s *ScheduleStore) UpdateMetadata(ctx context.Context, id string, expectedVersion int64, targetDate *time.Time, priority *int, jobGroup *string, plannerNotes *string) (*ScheduleItem, error) {
	if expectedVersion <= 0 {
		return nil, ErrStaleVersion
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE schedule_items
		SET target_date = COALESCE($3, target_date),
			priority = COALESCE($4, priority),
			job_group = COALESCE($5, job_group),
			planner_notes = COALESCE($6, planner_notes),
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND version = $2
	`, id, expectedVersion, targetDate, priority, jobGroup, plannerNotes)
	if err != nil {
		return nil, fmt.Errorf("update schedule item metadata %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("inspect schedule metadata update: %w", err)
	}
	if rowsAffected == 0 {
		return nil, ErrStaleVersion
	}

	return s.GetByID(ctx, id)
}

func (s *ScheduleStore) QueueVersion(ctx context.Context, workstationID string) (int64, error) {
	if workstationID == "" {
		return 0, ErrInvalidQueue
	}

	var version int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0)
		FROM schedule_items
		WHERE workstation_id = $1 AND status = 'queued'
	`, workstationID).Scan(&version); err != nil {
		return 0, fmt.Errorf("queue version for workstation %s: %w", workstationID, err)
	}

	return version, nil
}

func (s *ScheduleStore) ReorderQueue(ctx context.Context, workstationID string, expectedVersion int64, orderedItemIDs []string) error {
	if workstationID == "" || len(orderedItemIDs) == 0 {
		return ErrInvalidQueue
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin queue reorder: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = s.validateWorkstationTx(ctx, tx, workstationID); err != nil {
		return err
	}

	var currentVersion int64
	if err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0)
		FROM schedule_items
		WHERE workstation_id = $1 AND status = 'queued'
	`, workstationID).Scan(&currentVersion); err != nil {
		return fmt.Errorf("read current queue version: %w", err)
	}
	if expectedVersion > 0 && currentVersion != expectedVersion {
		return ErrQueueVersion
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM schedule_items
		WHERE workstation_id = $1 AND status = 'queued'
	`, workstationID)
	if err != nil {
		return fmt.Errorf("load queued items for reorder: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return fmt.Errorf("scan queued item id: %w", scanErr)
		}
		existing[id] = struct{}{}
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("iterate queued items: %w", err)
	}

	if len(existing) != len(orderedItemIDs) {
		return ErrInvalidQueue
	}

	seen := make(map[string]struct{}, len(orderedItemIDs))
	for _, id := range orderedItemIDs {
		if _, ok := existing[id]; !ok {
			return ErrInvalidQueue
		}
		if _, dup := seen[id]; dup {
			return ErrInvalidQueue
		}
		seen[id] = struct{}{}
	}

	for idx, id := range orderedItemIDs {
		result, execErr := tx.ExecContext(ctx, `
			UPDATE schedule_items
			SET sort_order = $3, version = version + 1, updated_at = NOW()
			WHERE id = $1 AND workstation_id = $2 AND status = 'queued'
		`, id, workstationID, idx+1)
		if execErr != nil {
			return fmt.Errorf("update queue order for %s: %w", id, execErr)
		}

		affected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("inspect queue update rows for %s: %w", id, rowsErr)
		}
		if affected == 0 {
			return ErrInvalidQueue
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit queue reorder: %w", err)
	}

	return nil
}

func (s *ScheduleStore) getPOLineTx(ctx context.Context, tx *sql.Tx, id string) (*POLine, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, external_po_number, external_part_number, external_item_number,
			status, required_date, qty_required, raw_payload, external_last_mod_date, synced_at
		FROM jobboss_po_lines
		WHERE id = $1
	`, id)
	return scanPOLine(row)
}

func (s *ScheduleStore) validateWorkstationTx(ctx context.Context, tx *sql.Tx, workstationID string) error {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM workstations WHERE id = $1 AND is_active = TRUE`, workstationID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrWorkstationInvalid
		}
		return fmt.Errorf("validate workstation %s: %w", workstationID, err)
	}

	return nil
}

func (s *ScheduleStore) nextSortOrderTx(ctx context.Context, tx *sql.Tx, workstationID string) (int, error) {
	var next int
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(sort_order), 0) + 1
		FROM schedule_items
		WHERE workstation_id = $1 AND status = 'queued'
	`, workstationID).Scan(&next); err != nil {
		return 0, fmt.Errorf("next sort order for workstation %s: %w", workstationID, err)
	}
	return next, nil
}

func (s *ScheduleStore) createTx(ctx context.Context, tx *sql.Tx, item *ScheduleItem) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO schedule_items (
			id, po_line_id, workstation_id, sort_order, status, job_type, job_group,
			part_number, part_description, material_spec, external_job_number, receiving_reference,
			planned_start_date, required_date, target_date, priority, planner_notes, qty_required,
			version, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18,
			$19, $20, $21
		)
	`, item.ID, item.POLineID, item.WorkstationID, item.SortOrder, item.Status, item.JobType, item.JobGroup,
		item.PartNumber, item.PartDescription, item.MaterialSpec, item.ExternalJobNumber, item.ReceivingReference,
		item.PlannedStartDate, item.RequiredDate, item.TargetDate, item.Priority, item.PlannerNotes, item.QtyRequired,
		item.Version, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create schedule item %s: %w", item.ID, err)
	}

	return nil
}

func (s *ScheduleStore) getByIDTx(ctx context.Context, tx *sql.Tx, id string) (*ScheduleItem, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, po_line_id, workstation_id, sort_order, status, job_type, job_group,
			part_number, part_description, material_spec, external_job_number, receiving_reference,
			planned_start_date, required_date, target_date, priority, planner_notes, qty_required,
			version, created_at, updated_at
		FROM schedule_items
		WHERE id = $1
	`, id)
	item, err := scanScheduleItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get schedule item %s: %w", id, err)
	}

	return item, nil
}

func scanPOLine(row interface{ Scan(dest ...any) error }) (*POLine, error) {
	var item POLine
	var status sql.NullString
	var requiredDate sql.NullTime
	var externalLastMod sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.ExternalPONumber,
		&item.ExternalPartNumber,
		&item.ExternalItemNumber,
		&status,
		&requiredDate,
		&item.QtyRequired,
		&item.RawPayload,
		&externalLastMod,
		&item.SyncedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan po line: %w", err)
	}

	if status.Valid {
		v := status.String
		item.Status = &v
	}
	if requiredDate.Valid {
		v := requiredDate.Time
		item.RequiredDate = &v
	}
	if externalLastMod.Valid {
		v := externalLastMod.Time
		item.ExternalLastModDate = &v
	}
	if len(item.RawPayload) == 0 {
		item.RawPayload = json.RawMessage("{}")
	}

	return &item, nil
}

func scanScheduleItem(row interface{ Scan(dest ...any) error }) (*ScheduleItem, error) {
	var item ScheduleItem
	var poLineID sql.NullString
	var workstationID sql.NullString
	var sortOrder sql.NullInt64
	var jobGroup sql.NullString
	var partDescription sql.NullString
	var materialSpec sql.NullString
	var externalJobNumber sql.NullString
	var receivingReference sql.NullString
	var plannedStartDate sql.NullTime
	var requiredDate sql.NullTime
	var targetDate sql.NullTime
	var plannerNotes sql.NullString

	if err := row.Scan(
		&item.ID,
		&poLineID,
		&workstationID,
		&sortOrder,
		&item.Status,
		&item.JobType,
		&jobGroup,
		&item.PartNumber,
		&partDescription,
		&materialSpec,
		&externalJobNumber,
		&receivingReference,
		&plannedStartDate,
		&requiredDate,
		&targetDate,
		&item.Priority,
		&plannerNotes,
		&item.QtyRequired,
		&item.Version,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan schedule item: %w", err)
	}

	if poLineID.Valid {
		v := poLineID.String
		item.POLineID = &v
	}
	if workstationID.Valid {
		v := workstationID.String
		item.WorkstationID = &v
	}
	if sortOrder.Valid {
		v := int(sortOrder.Int64)
		item.SortOrder = &v
	}
	if jobGroup.Valid {
		v := jobGroup.String
		item.JobGroup = &v
	}
	if partDescription.Valid {
		v := partDescription.String
		item.PartDescription = &v
	}
	if materialSpec.Valid {
		v := materialSpec.String
		item.MaterialSpec = &v
	}
	if externalJobNumber.Valid {
		v := externalJobNumber.String
		item.ExternalJobNumber = &v
	}
	if receivingReference.Valid {
		v := receivingReference.String
		item.ReceivingReference = &v
	}
	if plannedStartDate.Valid {
		v := plannedStartDate.Time
		item.PlannedStartDate = &v
	}
	if requiredDate.Valid {
		v := requiredDate.Time
		item.RequiredDate = &v
	}
	if targetDate.Valid {
		v := targetDate.Time
		item.TargetDate = &v
	}
	if plannerNotes.Valid {
		v := plannerNotes.String
		item.PlannerNotes = &v
	}

	return &item, nil
}
