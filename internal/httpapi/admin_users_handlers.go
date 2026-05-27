package httpapi

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scheduler/internal/auth"
	"scheduler/internal/store"
)

type adminUsersHandler struct {
	users       *store.UserStore
	authManager *auth.Manager
}

func newAdminUsersHandler(users *store.UserStore, authManager *auth.Manager) *adminUsersHandler {
	return &adminUsersHandler{users: users, authManager: authManager}
}

func (h *adminUsersHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (h *adminUsersHandler) handleUser(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users/")
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

func (h *adminUsersHandler) handleGetByID(w http.ResponseWriter, r *http.Request, id string) {
	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	jsonResponse(w, http.StatusOK, h.userResponse(*user))
}

func (h *adminUsersHandler) handleList(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r)

	users, err := h.users.List(r.Context(), limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	response := make([]map[string]any, 0, len(users))
	for _, user := range users {
		response = append(response, h.userResponse(user))
	}

	jsonResponse(w, http.StatusOK, response)
}

func (h *adminUsersHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Active   *bool  `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		jsonError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	role := store.UserRole(strings.TrimSpace(req.Role))
	if role != store.RoleAdmin && role != store.RoleWorkstation {
		jsonError(w, http.StatusBadRequest, "invalid role")
		return
	}

	exists, err := h.users.UsernameExists(r.Context(), req.Username)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to validate username")
		return
	}
	if exists {
		jsonError(w, http.StatusUnprocessableEntity, "username already exists")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	isActive := true
	if req.Active != nil {
		isActive = *req.Active
	}

	user := &store.User{
		ID:           generateID("user"),
		Username:     req.Username,
		PasswordHash: hash,
		Role:         role,
		IsActive:     isActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := h.users.Create(r.Context(), user); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	jsonResponse(w, http.StatusCreated, h.userResponse(*user))
}

func (h *adminUsersHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Role   *string `json:"role"`
		Active *bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}

	if req.Role != nil {
		role := store.UserRole(strings.TrimSpace(*req.Role))
		if role != store.RoleAdmin && role != store.RoleWorkstation {
			jsonError(w, http.StatusBadRequest, "invalid role")
			return
		}
		user.Role = role
	}

	if req.Active != nil {
		user.IsActive = *req.Active
	}

	if err := h.users.Update(r.Context(), user); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	updated, err := h.users.GetByID(r.Context(), user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to load updated user")
		return
	}

	jsonResponse(w, http.StatusOK, h.userResponse(*updated))
}

func (h *adminUsersHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.users.Delete(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *adminUsersHandler) userResponse(user store.User) map[string]any {
	return map[string]any{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"active":     user.IsActive,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}
}

func generateID(prefix string) string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(raw[:])
}

func parseLimitOffset(r *http.Request) (int, int) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 500 {
		limit = 500
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}
