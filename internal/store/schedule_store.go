package store

import (
	"context"
	"database/sql"
	"fmt"
)

// ScheduleStore manages schedule_items persistence.
type ScheduleStore struct {
	db DBTX
}

// NewScheduleStore constructs a ScheduleStore.
func NewScheduleStore(db DBTX) *ScheduleStore {
	return &ScheduleStore{db: db}
}

// List returns a page of schedule items. For now it returns an empty slice.
func (s *ScheduleStore) List(ctx context.Context, limit, offset int) ([]ScheduleItem, error) {
	return []ScheduleItem{}, nil
}

// GetByID loads a schedule item by id.
func (s *ScheduleStore) GetByID(ctx context.Context, id string) (*ScheduleItem, error) {
	return nil, sql.ErrNoRows
}

// Create inserts a schedule item. This is a minimal stub for scaffolding.
func (s *ScheduleStore) Create(ctx context.Context, item *ScheduleItem) error {
	if item == nil {
		return fmt.Errorf("nil schedule item")
	}
	return nil
}
