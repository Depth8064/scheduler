package httpapi

import (
    "net/http"
    "scheduler/internal/store"
)

type adminSchedulesHandler struct{
    schedules *store.ScheduleStore
}

func newAdminSchedulesHandler(schedules *store.ScheduleStore) *adminSchedulesHandler {
    return &adminSchedulesHandler{schedules: schedules}
}

func (h *adminSchedulesHandler) handleSchedules(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        h.handleList(w, r)
    default:
        methodNotAllowed(w)
    }
}

func (h *adminSchedulesHandler) handleList(w http.ResponseWriter, r *http.Request) {
    limit, offset := parseLimitOffset(r)
    items, err := h.schedules.List(r.Context(), limit, offset)
    if err != nil {
        jsonError(w, http.StatusInternalServerError, "failed to list schedule items")
        return
    }

    response := make([]map[string]any, 0, len(items))
    for _, it := range items {
        response = append(response, map[string]any{"id": it.ID})
    }
    jsonResponse(w, http.StatusOK, response)
}
