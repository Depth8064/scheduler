package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// WorkstationStore provides workstation persistence operations.
type WorkstationStore struct {
	db DBTX
}

// NewWorkstationStore constructs a workstation repository.
func NewWorkstationStore(db DBTX) *WorkstationStore {
	return &WorkstationStore{db: db}
}

// Create inserts a new workstation row.
func (s *WorkstationStore) Create(ctx context.Context, workstation *Workstation) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workstations (id, name, station_type, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, workstation.ID, workstation.Name, workstation.StationType, workstation.IsActive, workstation.CreatedAt, workstation.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create workstation %q: %w", workstation.Name, err)
	}

	return nil
}

// GetByID returns a workstation by its primary key.
func (s *WorkstationStore) GetByID(ctx context.Context, id string) (*Workstation, error) {
	return s.getOne(ctx, `WHERE id = $1`, id)
}

// List returns all workstations ordered by station_type then name.
func (s *WorkstationStore) List(ctx context.Context) ([]Workstation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, station_type, is_active, created_at, updated_at
		FROM workstations
		ORDER BY station_type, name
	`)
	if err != nil {
		return nil, fmt.Errorf("list workstations: %w", err)
	}
	defer rows.Close()

	workstations := make([]Workstation, 0)
	for rows.Next() {
		workstation, scanErr := scanWorkstation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		workstations = append(workstations, *workstation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list workstations: %w", err)
	}

	return workstations, nil
}

// ListActive returns all active workstations ordered by station_type then name.
func (s *WorkstationStore) ListActive(ctx context.Context) ([]Workstation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, station_type, is_active, created_at, updated_at
		FROM workstations
		WHERE is_active = TRUE
		ORDER BY station_type, name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active workstations: %w", err)
	}
	defer rows.Close()

	workstations := make([]Workstation, 0)
	for rows.Next() {
		workstation, scanErr := scanWorkstation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		workstations = append(workstations, *workstation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list active workstations: %w", err)
	}

	return workstations, nil
}

// SetActive updates the active flag for a workstation.
func (s *WorkstationStore) SetActive(ctx context.Context, id string, isActive bool) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE workstations
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`, id, isActive)
	if err != nil {
		return fmt.Errorf("update workstation active flag %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect workstation active update %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Update updates the workstation name, station_type, and active state.
func (s *WorkstationStore) Update(ctx context.Context, workstation *Workstation) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE workstations
		SET name = $2, station_type = $3, is_active = $4, updated_at = NOW()
		WHERE id = $1
	`, workstation.ID, workstation.Name, workstation.StationType, workstation.IsActive)
	if err != nil {
		return fmt.Errorf("update workstation %s: %w", workstation.ID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect workstation update %s: %w", workstation.ID, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Delete removes a workstation row.
func (s *WorkstationStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM workstations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete workstation %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect workstation delete %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *WorkstationStore) getOne(ctx context.Context, whereClause string, arg any) (*Workstation, error) {
	query := fmt.Sprintf(`
		SELECT id, name, station_type, is_active, created_at, updated_at
		FROM workstations
		%s
	`, whereClause)

	row := s.db.QueryRowContext(ctx, query, arg)
	workstation, err := scanWorkstationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return workstation, nil
}

func scanWorkstationRow(row interface{ Scan(dest ...any) error }) (*Workstation, error) {
	var workstation Workstation
	if err := row.Scan(&workstation.ID, &workstation.Name, &workstation.StationType, &workstation.IsActive, &workstation.CreatedAt, &workstation.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan workstation: %w", err)
	}

	return &workstation, nil
}

func scanWorkstation(rows interface{ Scan(dest ...any) error }) (*Workstation, error) {
	var workstation Workstation
	if err := rows.Scan(&workstation.ID, &workstation.Name, &workstation.StationType, &workstation.IsActive, &workstation.CreatedAt, &workstation.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan workstation: %w", err)
	}

	return &workstation, nil
}
