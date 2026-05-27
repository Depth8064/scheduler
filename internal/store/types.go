package store

import (
	"context"
	"database/sql"
	"time"
)

// DBTX is the minimal database interface used by repositories.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type UserRole string

const (
	RoleAdmin       UserRole = "admin"
	RoleWorkstation UserRole = "workstation"
)

// ScheduleItemStatus represents the possible states of a schedule item.
type ScheduleItemStatus string

const (
	StatusUnreleased ScheduleItemStatus = "unreleased"
	StatusReleased   ScheduleItemStatus = "released"
	StatusQueued     ScheduleItemStatus = "queued"
	StatusInProgress ScheduleItemStatus = "in_progress"
	StatusPaused     ScheduleItemStatus = "paused"
	StatusBlocked    ScheduleItemStatus = "blocked"
	StatusCompleted  ScheduleItemStatus = "completed"
)

// JobType represents the type of a schedule item.
type JobType string

const (
	JobTypeOpen   JobType = "OPEN"
	JobTypeKanBan JobType = "KAN_BAN"
)

// User mirrors the users table.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	Role         UserRole
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Workstation mirrors the workstations table.
// Name is the specific machine name (e.g. "Burner 1").
// StationType is the department-level grouping (e.g. "Burner", "Laser", "Saw").
type Workstation struct {
	ID          string
	Name        string
	StationType string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Session mirrors the sessions table.
type Session struct {
	ID        string
	UserID    string
	Username  string
	Role      UserRole
	CSRFToken string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// POLine mirrors the jobboss_po_lines table (read-only mirrored data from JobBOSS).
type POLine struct {
	ID                  string
	ExternalPONumber    string
	ExternalPartNumber  string
	ExternalItemNumber  string
	Status              *string
	RequiredDate        *time.Time
	QtyRequired         float64
	RawPayload          []byte
	ExternalLastModDate *time.Time
	SyncedAt            time.Time
}

// ScheduleItem mirrors the schedule_items table.
// POLineID is nil for KAN_BAN items (no corresponding PO line).
// WorkstationID is nil when unreleased or released-but-unassigned.
// SortOrder is nil when the item is not in any workstation queue.
type ScheduleItem struct {
	ID                 string
	POLineID           *string
	WorkstationID      *string
	SortOrder          *int
	Status             ScheduleItemStatus
	JobType            JobType
	JobGroup           *string
	PartNumber         string
	PartDescription    *string
	MaterialSpec       *string
	ExternalJobNumber  *string
	ReceivingReference *string
	PlannedStartDate   *time.Time
	RequiredDate       *time.Time
	TargetDate         *time.Time
	Priority           int
	PlannerNotes       *string
	QtyRequired        float64
	Version            int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ExecutionProgress mirrors the execution_progress table.
// There is at most one row per schedule item (the current live snapshot).
type ExecutionProgress struct {
	ID             string
	ScheduleItemID string
	QtyComplete    float64
	QtyRemaining   float64
	ScrapDelta     float64
	UpdatedBy      *string
	UpdatedAt      time.Time
}

// ExecutionEvent mirrors the execution_events table (immutable timeline entries).
type ExecutionEvent struct {
	ID             string
	ScheduleItemID string
	EventType      string
	ActorID        *string
	OccurredAt     time.Time
	Notes          *string
	EventPayload   []byte
}

// KanbanItem mirrors the kanban_items table.
type KanbanItem struct {
	PartNumber              string
	PartDescription         *string
	UnitsPerAssembly        int
	MonthlyQty              int
	WeeklyQty               int
	BinQty                  int
	TotalBins               int
	TotalKanbanQty          int
	BinTypeNotes            *string
	DedicatedWorkstationID  *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// SyncCheckpoint mirrors the sync_checkpoints table.
type SyncCheckpoint struct {
	ResourceName    string
	LastSyncAt      *time.Time
	LastModifiedMax *time.Time
	Status          string
	ErrorMessage    *string
	RecordCount     int
}

// SyncRun mirrors the sync_runs table.
type SyncRun struct {
	ID           string
	ResourceName string
	Status       string
	StartedAt    time.Time
	EndedAt      *time.Time
	RecordCount  int
	ErrorSummary *string
}
