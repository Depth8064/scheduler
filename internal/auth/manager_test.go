package auth

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"scheduler/internal/store"
)

func TestManagerAuthenticateValidateAndInvalidateSession(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	users := store.NewUserStore(db)
	sessions := store.NewAuthStore(db)
	manager := NewManager(users, sessions, "01234567890123456789012345678901")
	manager.SetSessionLifetime(2 * time.Hour)

	passwordHash, err := HashPassword("secret-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	createdAt := time.Now().UTC().Truncate(time.Second)
	updatedAt := createdAt

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, username, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE username = $1
	`)).WithArgs("alice").WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "role", "is_active", "created_at", "updated_at"}).AddRow(
		"user_1",
		"alice",
		passwordHash,
		store.RoleAdmin,
		true,
		createdAt,
		updatedAt,
	))

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO sessions (id, user_id, username, role, csrf_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)).WithArgs(sqlmock.AnyArg(), "user_1", "alice", store.RoleAdmin, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

	session, err := manager.Authenticate(context.Background(), "alice", "secret-password")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if session.Username != "alice" || session.Role != store.RoleAdmin {
		t.Fatalf("unexpected session: %+v", session)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, username, role, csrf_token, expires_at, created_at
		FROM sessions
		WHERE id = $1
	`)).WithArgs(session.ID).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "username", "role", "csrf_token", "expires_at", "created_at"}).AddRow(
		session.ID,
		"user_1",
		"alice",
		store.RoleAdmin,
		session.CSRFToken,
		session.ExpiresAt,
		createdAt,
	))

	validated, err := manager.ValidateSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if validated.ID != session.ID || validated.CSRFToken != session.CSRFToken {
		t.Fatalf("unexpected validated session: %+v", validated)
	}

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM sessions WHERE id = $1`)).WithArgs(session.ID).WillReturnResult(sqlmock.NewResult(1, 1))
	manager.InvalidateSession(context.Background(), session.ID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestManagerAuthenticateRejectsBadPassword(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	users := store.NewUserStore(db)
	sessions := store.NewAuthStore(db)
	manager := NewManager(users, sessions, "01234567890123456789012345678901")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, username, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE username = $1
	`)).WithArgs("alice").WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "role", "is_active", "created_at", "updated_at"}).AddRow(
		"user_1",
		"alice",
		"$2a$10$abcdefghijklmnopqrstuv",
		store.RoleAdmin,
		true,
		time.Now(),
		time.Now(),
	))

	if _, err := manager.Authenticate(context.Background(), "alice", "wrong-password"); err == nil {
		t.Fatalf("expected error for bad password")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
