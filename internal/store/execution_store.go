package store

import (
    "context"
    "fmt"
)

// ExecutionStore manages execution progress and events.
type ExecutionStore struct{
    db DBTX
}

func NewExecutionStore(db DBTX) *ExecutionStore {
    return &ExecutionStore{db: db}
}

// RecordProgress stores a minimal progress update (stub).
func (s *ExecutionStore) RecordProgress(ctx context.Context, executionID string, progress int) error {
    if executionID == "" {
        return fmt.Errorf("missing execution id")
    }
    return nil
}

// AppendEvent appends an execution event (stub).
func (s *ExecutionStore) AppendEvent(ctx context.Context, executionID, event string) error {
    if executionID == "" {
        return fmt.Errorf("missing execution id")
    }
    return nil
}
