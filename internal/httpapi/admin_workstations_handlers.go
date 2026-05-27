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

type adminWorkstationsHandler struct {
	workstations *store.WorkstationStore
}

func newAdminWorkstationsHandler(workstations *store.WorkstationStore) *adminWorkstationsHandler {
	return &adminWorkstationsHandler{workstations: workstations}
}

func (h *adminWorkstationsHandler) handleWorkstations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (h *adminWorkstationsHandler) handleWorkstation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/workstations/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetByID(w, r, id)
	case http.MethodPatch:
		h.handleUpdate(w, r, id)
	case http.MethodDelete:
		h.handleDelete(w, r, id)
	default:
		methodNotAllowed(w)
	}
}

func (h *adminWorkstationsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := h.workstations.List(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list workstations")
		return
	}

	response := make([]map[string]any, 0, len(items))
	for _, item := range items {
		response = append(response, h.workstationResponse(item))
	}

	jsonResponse(w, http.StatusOK, response)
}

func (h *adminWorkstationsHandler) handleGetByID(w http.ResponseWriter, r *http.Request, id string) {
	workstation, err := h.workstations.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "workstation not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to get workstation")
		return
	}

	jsonResponse(w, http.StatusOK, h.workstationResponse(*workstation))
}

func (h *adminWorkstationsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		StationType string `json:"station_type"`
		Active      *bool  `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.StationType = strings.TrimSpace(req.StationType)
	if req.Name == "" || req.StationType == "" {
		jsonError(w, http.StatusBadRequest, "name and station_type are required")
		return
	}

	isActive := true
	if req.Active != nil {
		isActive = *req.Active
	}

	now := time.Now().UTC()
	workstation := &store.Workstation{
		ID:          generateID("workstation"),
		Name:        req.Name,
		StationType: req.StationType,
		IsActive:    isActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.workstations.Create(r.Context(), workstation); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create workstation")
		return
	}

	jsonResponse(w, http.StatusCreated, h.workstationResponse(*workstation))
}

func (h *adminWorkstationsHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Name        *string `json:"name"`
		StationType *string `json:"station_type"`
		Active      *bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	workstation, err := h.workstations.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "workstation not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to get workstation")
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			jsonError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
		workstation.Name = name
	}

	if req.StationType != nil {
		st := strings.TrimSpace(*req.StationType)
		if st == "" {
			jsonError(w, http.StatusBadRequest, "station_type cannot be empty")
			return
		}
		workstation.StationType = st
	}

	if req.Active != nil {
		workstation.IsActive = *req.Active
	}

	if err := h.workstations.Update(r.Context(), workstation); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update workstation")
		return
	}

	updated, err := h.workstations.GetByID(r.Context(), workstation.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to load updated workstation")
		return
	}

	jsonResponse(w, http.StatusOK, h.workstationResponse(*updated))
}

func (h *adminWorkstationsHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.workstations.Delete(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "workstation not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to delete workstation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *adminWorkstationsHandler) workstationResponse(workstation store.Workstation) map[string]any {
	return map[string]any{
		"id":           workstation.ID,
		"name":         workstation.Name,
		"station_type": workstation.StationType,
		"active":       workstation.IsActive,
		"created_at":   workstation.CreatedAt,
		"updated_at":   workstation.UpdatedAt,
	}
}

