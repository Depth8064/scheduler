package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserWorkstationAccessStore_GetWorkstationIDsByUser(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewUserWorkstationAccessStore(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT workstation_id
		FROM user_workstation_access
		WHERE user_id = $1
		ORDER BY workstation_id
	`)).WithArgs("user_1").WillReturnRows(sqlmock.NewRows([]string{"workstation_id"}).AddRow("workstation_1"))

	ids, err := store.GetWorkstationIDsByUser(context.Background(), "user_1")
	if err != nil {
		t.Fatalf("GetWorkstationIDsByUser: %v", err)
	}

	if len(ids) != 1 || ids[0] != "workstation_1" {
		t.Fatalf("unexpected ids: %v", ids)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserWorkstationAccessStore_GetUsersByWorkstation(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewUserWorkstationAccessStore(db)

	createdAt := time.Now().UTC().Truncate(time.Second)
	updatedAt := createdAt

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT u.id, u.username, u.password_hash, u.role, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN user_workstation_access a ON a.user_id = u.id
		WHERE a.workstation_id = $1
		ORDER BY u.username
	`)).WithArgs("workstation_1").WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "role", "is_active", "created_at", "updated_at"}).AddRow(
		"user_1",
		"alice",
		"hash",
		"admin",
		true,
		createdAt,
		updatedAt,
	))

	users, err := store.GetUsersByWorkstation(context.Background(), "workstation_1")
	if err != nil {
		t.Fatalf("GetUsersByWorkstation: %v", err)
	}

	if len(users) != 1 || users[0].ID != "user_1" || users[0].Username != "alice" {
		t.Fatalf("unexpected users: %+v", users)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUserWorkstationAccessStore_ReplaceUsersForWorkstation(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewUserWorkstationAccessStore(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM user_workstation_access WHERE workstation_id = $1")).WithArgs("workstation_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_workstation_access (user_id, workstation_id) VALUES ($1, $2),($3, $4)")).WithArgs("user_1", "workstation_1", "user_2", "workstation_1").WillReturnResult(sqlmock.NewResult(2, 2))
	mock.ExpectCommit()

	if err := store.ReplaceUsersForWorkstation(context.Background(), "workstation_1", []string{"user_1", "user_2"}); err != nil {
		t.Fatalf("ReplaceUsersForWorkstation: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
