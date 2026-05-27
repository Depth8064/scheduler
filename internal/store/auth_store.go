package store

import (
	"context"
	"database/sql"
	"fmt"
)

// AuthStore provides persistence for auth sessions.
type AuthStore struct {
	db DBTX
}

// NewAuthStore constructs a session repository.
func NewAuthStore(db DBTX) *AuthStore {
	return &AuthStore{db: db}
}

// CreateSession inserts or replaces a session row.
func (s *AuthStore) CreateSession(ctx context.Context, session *Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, username, role, csrf_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, session.ID, session.UserID, session.Username, session.Role, session.CSRFToken, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("create session %s: %w", session.ID, err)
	}

	return nil
}

// GetSession returns a session by id.
func (s *AuthStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, username, role, csrf_token, expires_at, created_at
		FROM sessions
		WHERE id = $1
	`, sessionID)

	var session Session
	if err := row.Scan(&session.ID, &session.UserID, &session.Username, &session.Role, &session.CSRFToken, &session.ExpiresAt, &session.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get session %s: %w", sessionID, err)
	}

	return &session, nil
}

// DeleteSession removes a single session.
func (s *AuthStore) DeleteSession(ctx context.Context, sessionID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID); err != nil {
		return fmt.Errorf("delete session %s: %w", sessionID, err)
	}

	return nil
}

// DeleteExpiredSessions removes expired rows.
func (s *AuthStore) DeleteExpiredSessions(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}
