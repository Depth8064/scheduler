package httpapi

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "regexp"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"

    "scheduler/internal/store"
)

func TestCreateWorkstationAndAssignUsers_E2E(t *testing.T) {
    t.Parallel()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("sqlmock new: %v", err)
    }
    defer db.Close()

    wsStore := store.NewWorkstationStore(db)
    access := store.NewUserWorkstationAccessStore(db)
    h := newAdminWorkstationsHandler(wsStore, access)

    // Expect create workstation insert
    mock.ExpectExec(regexp.QuoteMeta("INSERT INTO workstations (id, name, station_type, is_active, created_at, updated_at)\n\t\tVALUES ($1, $2, $3, $4, $5, $6)")).
        WithArgs(sqlmock.AnyArg(), "Burner 1", "Burner", true, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnResult(sqlmock.NewResult(1, 1))

    body := map[string]any{"name": "Burner 1", "station_type": "Burner"}
    b, _ := json.Marshal(body)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/workstations", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()

    h.handleWorkstations(rr, req)
    if rr.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d", rr.Code)
    }

    // Expect replace users (delete + insert)
    mock.ExpectBegin()
    mock.ExpectExec(regexp.QuoteMeta("DELETE FROM user_workstation_access WHERE workstation_id = $1")).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
    mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_workstation_access (user_id, workstation_id) VALUES ($1, $2),($3, $4)")).
        WithArgs("user_1", sqlmock.AnyArg(), "user_2", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(2, 2))
    mock.ExpectCommit()

    // Use the workstation id returned earlier
    var resp map[string]any
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("unmarshal create response: %v", err)
    }
    id, _ := resp["id"].(string)

    replaceBody := map[string]any{"user_ids": []string{"user_1", "user_2"}}
    rb, _ := json.Marshal(replaceBody)
    req2 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/workstations/"+id+"/users", bytes.NewReader(rb))
    req2.Header.Set("Content-Type", "application/json")
    rr2 := httptest.NewRecorder()

    h.handleWorkstation(rr2, req2)
    if rr2.Code != http.StatusOK {
        t.Fatalf("expected 200 replace users, got %d", rr2.Code)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
