package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"scheduler/internal/auth"
	"scheduler/internal/store"
)

type authHandler struct {
	authManager *auth.Manager
	access      *store.UserWorkstationAccessStore
}

func newAuthHandler(authManager *auth.Manager, access *store.UserWorkstationAccessStore) *authHandler {
	return &authHandler{authManager: authManager, access: access}
}

func (h *authHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.authManager.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound), errors.Is(err, auth.ErrInvalidPassword), errors.Is(err, auth.ErrUnauthorized):
			jsonError(w, http.StatusUnauthorized, "invalid username or password")
		default:
			jsonError(w, http.StatusInternalServerError, "login failed")
		}
		return
	}

	setSessionCookies(w, h.authManager, session)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if sessionID, err := h.authManager.ParseSessionCookieValue(cookie.Value); err == nil {
			h.authManager.InvalidateSession(r.Context(), sessionID)
		}
	}

	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// handleMe returns the currently authenticated user's identity.
// GET /api/v1/auth/me → 200 {user_id, username, role, assigned_workstation_ids}
// Returns 401 if not authenticated.
func (h *authHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	sessionID, err := h.authManager.ParseSessionCookieValue(cookie.Value)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	session, err := h.authManager.ValidateSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// assigned_workstation_ids is populated for workstation-role users via
	// user_workstation_access; admin users have access to all workstations.
	assignedIDs := []string{}
	if session.Role == store.RoleWorkstation {
		ids, err := h.access.GetWorkstationIDsByUser(r.Context(), session.UserID)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to load assigned workstations")
			return
		}
		assignedIDs = ids
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"user_id":                  session.UserID,
		"username":                 session.Username,
		"role":                     session.Role,
		"assigned_workstation_ids": assignedIDs,
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}
