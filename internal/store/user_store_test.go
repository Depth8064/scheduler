package store

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserStore_Exists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	store := NewUserStore(db)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users\)`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := store.Exists(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatal("expected exists to be true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled mock expectations: %v", err)
	}
}

func TestUserStore_ExistsFalse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	store := NewUserStore(db)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users\)`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := store.Exists(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatal("expected exists to be false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled mock expectations: %v", err)
	}
}
