package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"scheduler/internal/store"
)

func TestHandleListUsers_HTTP(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	access := store.NewUserWorkstationAccessStore(db)
	h := newAdminWorkstationsHandler(nil, access)

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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/workstations/workstation_1/users", nil)
	rr := httptest.NewRecorder()

	h.handleListUsers(rr, req, "workstation_1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 user, got %d", len(resp))
	}
	if resp[0]["id"] != "user_1" || resp[0]["username"] != "alice" {
		t.Fatalf("unexpected user payload: %+v", resp[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestHandleReplaceUsers_HTTP(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	access := store.NewUserWorkstationAccessStore(db)
	h := newAdminWorkstationsHandler(nil, access)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM user_workstation_access WHERE workstation_id = $1")).WithArgs("workstation_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_workstation_access (user_id, workstation_id) VALUES ($1, $2),($3, $4)")).WithArgs("user_1", "workstation_1", "user_2", "workstation_1").WillReturnResult(sqlmock.NewResult(2, 2))
	mock.ExpectCommit()

	body := map[string]any{"user_ids": []string{"user_1", "user_2"}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/workstations/workstation_1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.handleReplaceUsers(rr, req, "workstation_1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if updated, ok := resp["updated"].(bool); !ok || !updated {
		t.Fatalf("unexpected response body: %v", resp)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
