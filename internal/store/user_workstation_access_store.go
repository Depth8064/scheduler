package store

import (
	"context"
	"fmt"
	"strings"
)

// UserWorkstationAccessStore manages user-workstation assignments.
type UserWorkstationAccessStore struct {
	db DBTX
}

// NewUserWorkstationAccessStore constructs a repository for access assignments.
func NewUserWorkstationAccessStore(db DBTX) *UserWorkstationAccessStore {
	return &UserWorkstationAccessStore{db: db}
}

// GetWorkstationIDsByUser returns the workstation IDs assigned to a user.
func (s *UserWorkstationAccessStore) GetWorkstationIDsByUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT workstation_id
		FROM user_workstation_access
		WHERE user_id = $1
		ORDER BY workstation_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list workstation access for user %s: %w", userID, err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan workstation access for user %s: %w", userID, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list workstation access for user %s: %w", userID, err)
	}

	return ids, nil
}

// GetUsersByWorkstation returns all users assigned to a workstation.
func (s *UserWorkstationAccessStore) GetUsersByWorkstation(ctx context.Context, workstationID string) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.password_hash, u.role, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN user_workstation_access a ON a.user_id = u.id
		WHERE a.workstation_id = $1
		ORDER BY u.username
	`, workstationID)
	if err != nil {
		return nil, fmt.Errorf("list users for workstation %s: %w", workstationID, err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users for workstation %s: %w", workstationID, err)
	}

	return users, nil
}

// ReplaceUsersForWorkstation replaces the full user list for a workstation.
func (s *UserWorkstationAccessStore) ReplaceUsersForWorkstation(ctx context.Context, workstationID string, userIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace workstation users %s: %w", workstationID, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM user_workstation_access WHERE workstation_id = $1`, workstationID); err != nil {
		return fmt.Errorf("clear workstation assignments %s: %w", workstationID, err)
	}

	if len(userIDs) > 0 {
		placeholders := make([]string, 0, len(userIDs))
		args := make([]any, 0, len(userIDs)*2)
		for i, userID := range userIDs {
			placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
			args = append(args, userID, workstationID)
		}

		query := fmt.Sprintf(`INSERT INTO user_workstation_access (user_id, workstation_id) VALUES %s`, strings.Join(placeholders, ","))
		if _, err = tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("insert workstation assignments for %s: %w", workstationID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit workstation assignments for %s: %w", workstationID, err)
	}

	return nil
}
