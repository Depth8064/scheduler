package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"scheduler/internal/auth"
	"scheduler/internal/logging"
	"scheduler/internal/store"
	"time"
)

// NewRouter creates the top-level HTTP router.
func NewRouter(logger *logging.Manager, authManager *auth.Manager, repositories *store.Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	authHandlers := newAuthHandler(authManager, repositories.Access)
	mux.HandleFunc("POST /api/v1/auth/login", authHandlers.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandlers.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/me", authHandlers.handleMe)

	adminUsers := newAdminUsersHandler(repositories.Users, authManager)
	adminWorkstations := newAdminWorkstationsHandler(repositories.Workstations, repositories.Access)
	mux.Handle("GET /api/v1/admin/users", requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminUsers.handleUsers)))
	mux.Handle("POST /api/v1/admin/users", requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminUsers.handleUsers))))
	mux.Handle("GET /api/v1/admin/users/", requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminUsers.handleUser)))
	mux.Handle("PATCH /api/v1/admin/users/", requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminUsers.handleUser))))
	mux.Handle("DELETE /api/v1/admin/users/", requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminUsers.handleUser))))
	mux.Handle("GET /api/v1/admin/workstations", requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminWorkstations.handleWorkstations)))
	mux.Handle("POST /api/v1/admin/workstations", requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstations))))
	mux.Handle("GET /api/v1/admin/workstations/", requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminWorkstations.handleWorkstation)))
	mux.Handle("PATCH /api/v1/admin/workstations/", requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstation))))
	mux.Handle("DELETE /api/v1/admin/workstations/", requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstation))))
	mux.Handle("PUT /api/v1/admin/workstations/", requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstation))))

	mux.Handle("/", staticHandler())

	return requestLogger(logger, mux)
}

func staticHandler() http.Handler {
	staticDir := filepath.Join("web", "static")
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		return http.FileServer(http.Dir(staticDir))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "static assets unavailable", http.StatusServiceUnavailable)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func requestLogger(logger *logging.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		logger.LogRequest(
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			r.UserAgent(),
			r.RemoteAddr,
			rec.status,
			time.Since(start),
		)
	})
}

func jsonResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]any{"error": message})
}
