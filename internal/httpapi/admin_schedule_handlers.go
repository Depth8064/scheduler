package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"scheduler/internal/store"
)

type adminSchedulesHandler struct {
	schedules *store.ScheduleStore
}

func newAdminSchedulesHandler(schedules *store.ScheduleStore) *adminSchedulesHandler {
	return &adminSchedulesHandler{schedules: schedules}
}

func (h *adminSchedulesHandler) handlePOLines(w http.ResponseWriter, r *http.Request) {
	suffix := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/po-lines")
	suffix = strings.TrimPrefix(suffix, "/")

	if suffix == "" {
		switch r.Method {
		case http.MethodGet:
			h.handleInbox(w, r)
		default:
			methodNotAllowed(w)
		}
		return
	}

	if suffix == "import" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.handleImport(w, r)
		return
	}

	if strings.HasSuffix(suffix, "/promote") {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		poLineID := strings.TrimSuffix(suffix, "/promote")
		if poLineID == "" || strings.Contains(poLineID, "/") {
			http.NotFound(w, r)
			return
		}
		h.handlePromote(w, r, poLineID)
		return
	}

	http.NotFound(w, r)
}

func (h *adminSchedulesHandler) handleSchedules(w http.ResponseWriter, r *http.Request) {
	suffix := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/schedule-items")
	suffix = strings.TrimPrefix(suffix, "/")

	if suffix == "" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		default:
			methodNotAllowed(w)
		}
		return
	}

	if !strings.Contains(suffix, "/") {
		switch suffix {
		case "bulk-assign":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.handleBulkAssign(w, r)
			return
		case "reorder":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.handleReorder(w, r)
			return
		case "queue-version":
			if r.Method != http.MethodGet {
				methodNotAllowed(w)
				return
			}
			h.handleQueueVersion(w, r)
			return
		}
	}

	if strings.Contains(suffix, "/") {
		parts := strings.Split(suffix, "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		itemID := parts[0]
		action := parts[1]
		if itemID == "" {
			http.NotFound(w, r)
			return
		}

		switch action {
		case "release":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.handleRelease(w, r, itemID)
		case "assign":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.handleAssign(w, r, itemID)
		case "unassign":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.handleUnassign(w, r, itemID)
		default:
			http.NotFound(w, r)
		}
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.handleUpdate(w, r, suffix)
	default:
		methodNotAllowed(w)
	}
}

func (h *adminSchedulesHandler) handleQueueVersion(w http.ResponseWriter, r *http.Request) {
	workstationID := strings.TrimSpace(r.URL.Query().Get("workstation_id"))
	if workstationID == "" {
		jsonError(w, http.StatusBadRequest, "workstation_id is required")
		return
	}

	version, err := h.schedules.QueueVersion(r.Context(), workstationID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get queue version")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"workstation_id": workstationID,
		"queue_version":  version,
	})
}

func (h *adminSchedulesHandler) handleBulkAssign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkstationID string   `json:"workstation_id"`
		ItemIDs       []string `json:"item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.WorkstationID = strings.TrimSpace(req.WorkstationID)
	if req.WorkstationID == "" {
		jsonError(w, http.StatusBadRequest, "workstation_id is required")
		return
	}
	if len(req.ItemIDs) == 0 {
		jsonError(w, http.StatusBadRequest, "item_ids is required")
		return
	}

	type assignResult struct {
		ItemID string `json:"item_id"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]assignResult, 0, len(req.ItemIDs))
	successCount := 0
	for _, itemID := range req.ItemIDs {
		trimmed := strings.TrimSpace(itemID)
		if trimmed == "" {
			continue
		}

		_, err := h.schedules.Assign(r.Context(), trimmed, req.WorkstationID)
		if err != nil {
			message := "failed to assign"
			switch {
			case errors.Is(err, sql.ErrNoRows):
				message = "item not found"
			case errors.Is(err, store.ErrInvalidTransition):
				message = "item is not assignable"
			case errors.Is(err, store.ErrWorkstationInvalid):
				message = "workstation is invalid or inactive"
			}
			results = append(results, assignResult{ItemID: trimmed, Status: "failed", Error: message})
			continue
		}

		successCount++
		results = append(results, assignResult{ItemID: trimmed, Status: "assigned"})
	}

	statusCode := http.StatusOK
	if successCount == 0 {
		statusCode = http.StatusUnprocessableEntity
	}

	jsonResponse(w, statusCode, map[string]any{
		"workstation_id": req.WorkstationID,
		"requested":      len(req.ItemIDs),
		"assigned":       successCount,
		"results":        results,
	})
}

func (h *adminSchedulesHandler) handleReorder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkstationID string   `json:"workstation_id"`
		QueueVersion  int64    `json:"queue_version"`
		ItemIDs       []string `json:"item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.WorkstationID = strings.TrimSpace(req.WorkstationID)
	if req.WorkstationID == "" {
		jsonError(w, http.StatusBadRequest, "workstation_id is required")
		return
	}
	if len(req.ItemIDs) == 0 {
		jsonError(w, http.StatusBadRequest, "item_ids is required")
		return
	}

	if err := h.schedules.ReorderQueue(r.Context(), req.WorkstationID, req.QueueVersion, req.ItemIDs); err != nil {
		switch {
		case errors.Is(err, store.ErrQueueVersion):
			jsonError(w, http.StatusConflict, "queue version conflict")
		case errors.Is(err, store.ErrInvalidQueue), errors.Is(err, store.ErrWorkstationInvalid):
			jsonError(w, http.StatusUnprocessableEntity, "invalid queue reorder request")
		default:
			jsonError(w, http.StatusInternalServerError, "failed to reorder queue")
		}
		return
	}

	newVersion, versionErr := h.schedules.QueueVersion(r.Context(), req.WorkstationID)
	if versionErr != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get reordered queue version")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"workstation_id": req.WorkstationID,
		"queue_version":  newVersion,
		"updated":        true,
	})
}

func (h *adminSchedulesHandler) handleInbox(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r)
	items, err := h.schedules.ListUnpromotedPOLines(r.Context(), limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to load po inbox")
		return
	}

	response := make([]map[string]any, 0, len(items))
	for _, it := range items {
		response = append(response, map[string]any{
			"id":                     it.ID,
			"external_po_number":     it.ExternalPONumber,
			"external_part_number":   it.ExternalPartNumber,
			"external_item_number":   it.ExternalItemNumber,
			"status":                 it.Status,
			"required_date":          it.RequiredDate,
			"qty_required":           it.QtyRequired,
			"external_last_mod_date": it.ExternalLastModDate,
			"synced_at":              it.SyncedAt,
		})
	}
	jsonResponse(w, http.StatusOK, response)
}

func (h *adminSchedulesHandler) handleImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Lines []struct {
			ExternalPONumber   string     `json:"external_po_number"`
			ExternalPartNumber string     `json:"external_part_number"`
			ExternalItemNumber string     `json:"external_item_number"`
			Status             *string    `json:"status"`
			RequiredDate       *time.Time `json:"required_date"`
			QtyRequired        float64    `json:"qty_required"`
			RawPayload         any        `json:"raw_payload"`
			ExternalLastMod    *time.Time `json:"external_last_mod_date"`
		} `json:"lines"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Lines) == 0 {
		jsonError(w, http.StatusBadRequest, "lines is required")
		return
	}

	lines := make([]store.POLine, 0, len(req.Lines))
	for _, line := range req.Lines {
		if strings.TrimSpace(line.ExternalPONumber) == "" || strings.TrimSpace(line.ExternalPartNumber) == "" || strings.TrimSpace(line.ExternalItemNumber) == "" {
			jsonError(w, http.StatusBadRequest, "external_po_number, external_part_number, and external_item_number are required")
			return
		}
		if line.QtyRequired < 0 {
			jsonError(w, http.StatusBadRequest, "qty_required must be non-negative")
			return
		}

		rawPayload, err := json.Marshal(line.RawPayload)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid raw_payload")
			return
		}
		if len(rawPayload) == 0 || string(rawPayload) == "null" {
			rawPayload = []byte("{}")
		}

		lines = append(lines, store.POLine{
			ID:                  generateID(),
			ExternalPONumber:    strings.TrimSpace(line.ExternalPONumber),
			ExternalPartNumber:  strings.TrimSpace(line.ExternalPartNumber),
			ExternalItemNumber:  strings.TrimSpace(line.ExternalItemNumber),
			Status:              line.Status,
			RequiredDate:        line.RequiredDate,
			QtyRequired:         line.QtyRequired,
			RawPayload:          rawPayload,
			ExternalLastModDate: line.ExternalLastMod,
			SyncedAt:            time.Now().UTC(),
		})
	}

	count, err := h.schedules.UpsertPOLines(r.Context(), lines)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to import po lines")
		return
	}

	jsonResponse(w, http.StatusAccepted, map[string]any{"imported": count})
}

func (h *adminSchedulesHandler) handlePromote(w http.ResponseWriter, r *http.Request, poLineID string) {
	var req struct {
		WorkstationID *string `json:"workstation_id"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.WorkstationID != nil {
		trimmed := strings.TrimSpace(*req.WorkstationID)
		if trimmed == "" {
			req.WorkstationID = nil
		} else {
			req.WorkstationID = &trimmed
		}
	}

	item, err := h.schedules.PromotePOLine(r.Context(), poLineID, req.WorkstationID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			jsonError(w, http.StatusNotFound, "po line not found")
		case errors.Is(err, store.ErrScheduleItemExists):
			jsonError(w, http.StatusConflict, "po line already promoted")
		case errors.Is(err, store.ErrWorkstationInvalid):
			jsonError(w, http.StatusUnprocessableEntity, "workstation is invalid or inactive")
		default:
			jsonError(w, http.StatusInternalServerError, "failed to promote po line")
		}
		return
	}

	jsonResponse(w, http.StatusCreated, h.scheduleResponse(*item))
}

func (h *adminSchedulesHandler) handleList(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r)
	filter := store.ScheduleListFilter{}
	if statusRaw := strings.TrimSpace(r.URL.Query().Get("status")); statusRaw != "" {
		status := store.ScheduleItemStatus(statusRaw)
		filter.Status = &status
	}
	if wsID := strings.TrimSpace(r.URL.Query().Get("workstation_id")); wsID != "" {
		filter.WorkstationID = &wsID
	}

	items, err := h.schedules.ListFiltered(r.Context(), limit, offset, filter)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list schedule items")
		return
	}

	response := make([]map[string]any, 0, len(items))
	for _, it := range items {
		response = append(response, h.scheduleResponse(it))
	}
	jsonResponse(w, http.StatusOK, response)
}

func (h *adminSchedulesHandler) handleRelease(w http.ResponseWriter, r *http.Request, itemID string) {
	updated, err := h.schedules.Release(r.Context(), itemID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			jsonError(w, http.StatusNotFound, "schedule item not found")
		case errors.Is(err, store.ErrInvalidTransition):
			jsonError(w, http.StatusUnprocessableEntity, "only unreleased items can be released")
		default:
			jsonError(w, http.StatusInternalServerError, "failed to release item")
		}
		return
	}

	jsonResponse(w, http.StatusOK, h.scheduleResponse(*updated))
}

func (h *adminSchedulesHandler) handleAssign(w http.ResponseWriter, r *http.Request, itemID string) {
	var req struct {
		WorkstationID string `json:"workstation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.WorkstationID = strings.TrimSpace(req.WorkstationID)
	if req.WorkstationID == "" {
		jsonError(w, http.StatusBadRequest, "workstation_id is required")
		return
	}

	updated, err := h.schedules.Assign(r.Context(), itemID, req.WorkstationID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			jsonError(w, http.StatusNotFound, "schedule item not found")
		case errors.Is(err, store.ErrInvalidTransition):
			jsonError(w, http.StatusUnprocessableEntity, "item must be unreleased or released before assignment")
		case errors.Is(err, store.ErrWorkstationInvalid):
			jsonError(w, http.StatusUnprocessableEntity, "workstation is invalid or inactive")
		default:
			jsonError(w, http.StatusInternalServerError, "failed to assign item")
		}
		return
	}

	jsonResponse(w, http.StatusOK, h.scheduleResponse(*updated))
}

func (h *adminSchedulesHandler) handleUnassign(w http.ResponseWriter, r *http.Request, itemID string) {
	updated, err := h.schedules.Unassign(r.Context(), itemID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			jsonError(w, http.StatusNotFound, "schedule item not found")
		case errors.Is(err, store.ErrInvalidTransition):
			jsonError(w, http.StatusUnprocessableEntity, "only queued items can be unassigned")
		default:
			jsonError(w, http.StatusInternalServerError, "failed to unassign item")
		}
		return
	}

	jsonResponse(w, http.StatusOK, h.scheduleResponse(*updated))
}

func (h *adminSchedulesHandler) handleUpdate(w http.ResponseWriter, r *http.Request, itemID string) {
	var req struct {
		Version     int64   `json:"version"`
		TargetDate  *string `json:"target_date"`
		Priority    *int    `json:"priority"`
		JobGroup    *string `json:"job_group"`
		PlannerNote *string `json:"planner_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Version <= 0 {
		jsonError(w, http.StatusBadRequest, "version is required")
		return
	}

	var targetDate *time.Time
	if req.TargetDate != nil {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.TargetDate))
		if err != nil {
			jsonError(w, http.StatusBadRequest, "target_date must be YYYY-MM-DD")
			return
		}
		targetDate = &t
	}

	updated, err := h.schedules.UpdateMetadata(r.Context(), itemID, req.Version, targetDate, req.Priority, req.JobGroup, req.PlannerNote)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrStaleVersion):
			jsonError(w, http.StatusConflict, "version conflict")
		case errors.Is(err, sql.ErrNoRows):
			jsonError(w, http.StatusNotFound, "schedule item not found")
		default:
			jsonError(w, http.StatusInternalServerError, "failed to update schedule item")
		}
		return
	}

	jsonResponse(w, http.StatusOK, h.scheduleResponse(*updated))
}

func (h *adminSchedulesHandler) scheduleResponse(it store.ScheduleItem) map[string]any {
	return map[string]any{
		"id":                  it.ID,
		"po_line_id":          it.POLineID,
		"workstation_id":      it.WorkstationID,
		"sort_order":          it.SortOrder,
		"status":              it.Status,
		"job_type":            it.JobType,
		"job_group":           it.JobGroup,
		"part_number":         it.PartNumber,
		"part_description":    it.PartDescription,
		"material_spec":       it.MaterialSpec,
		"external_job_number": it.ExternalJobNumber,
		"target_date":         it.TargetDate,
		"priority":            it.Priority,
		"planner_notes":       it.PlannerNotes,
		"qty_required":        it.QtyRequired,
		"version":             it.Version,
		"created_at":          it.CreatedAt,
		"updated_at":          it.UpdatedAt,
	}
}
