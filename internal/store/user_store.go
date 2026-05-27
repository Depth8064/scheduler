package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// UserStore provides user persistence operations.
type UserStore struct {
	db DBTX
}

// NewUserStore constructs a user repository.
func NewUserStore(db DBTX) *UserStore {
	return &UserStore{db: db}
}

// Create inserts a new user row.
func (s *UserStore) Create(ctx context.Context, user *User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.Username, user.PasswordHash, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user %q: %w", user.Username, err)
	}

	return nil
}

// GetByID returns a user by its primary key.
func (s *UserStore) GetByID(ctx context.Context, id string) (*User, error) {
	return s.getOne(ctx, `WHERE id = $1`, id)
}

// GetByUsername returns a user by username.
func (s *UserStore) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.getOne(ctx, `WHERE username = $1`, username)
}

// List returns users ordered by username with pagination.
func (s *UserStore) List(ctx context.Context, limit, offset int) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, password_hash, role, is_active, created_at, updated_at
		FROM users
		ORDER BY username
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
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
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

// ListActive returns all active users ordered by username.
func (s *UserStore) ListActive(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE is_active = TRUE
		ORDER BY username
	`)
	if err != nil {
		return nil, fmt.Errorf("list active users: %w", err)
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
		return nil, fmt.Errorf("list active users: %w", err)
	}

	return users, nil
}

// UpdateRole changes the role for an existing user.
func (s *UserStore) UpdateRole(ctx context.Context, id string, role UserRole) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET role = $2, updated_at = NOW()
		WHERE id = $1
	`, id, role)
	if err != nil {
		return fmt.Errorf("update user role %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect user role update %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// SetActive updates the active flag for a user.
func (s *UserStore) SetActive(ctx context.Context, id string, isActive bool) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`, id, isActive)
	if err != nil {
		return fmt.Errorf("update user active flag %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect user active update %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdatePasswordHash updates a user's password hash.
func (s *UserStore) UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`, id, passwordHash)
	if err != nil {
		return fmt.Errorf("update user password hash %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect user password update %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Update updates the user's username, role, and active status.
func (s *UserStore) Update(ctx context.Context, user *User) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET username = $2, role = $3, is_active = $4, updated_at = NOW()
		WHERE id = $1
	`, user.ID, user.Username, user.Role, user.IsActive)
	if err != nil {
		return fmt.Errorf("update user %s: %w", user.ID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect user update %s: %w", user.ID, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Delete removes a user row.
func (s *UserStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect user delete %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UsernameExists reports whether a username is already taken.
func (s *UserStore) UsernameExists(ctx context.Context, username string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, nil
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE username = $1`, username).Scan(&count); err != nil {
		return false, fmt.Errorf("check username %q: %w", username, err)
	}

	return count > 0, nil
}

func (s *UserStore) getOne(ctx context.Context, whereClause string, arg any) (*User, error) {
	query := fmt.Sprintf(`
		SELECT id, username, password_hash, role, is_active, created_at, updated_at
		FROM users
		%s
	`, whereClause)

	row := s.db.QueryRowContext(ctx, query, arg)
	user, err := scanUserRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return user, nil
}

func scanUserRow(row interface{ Scan(dest ...any) error }) (*User, error) {
	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	return &user, nil
}

func scanUser(rows interface{ Scan(dest ...any) error }) (*User, error) {
	var user User
	if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}

	return &user, nil
}
