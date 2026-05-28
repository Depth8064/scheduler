package httpapi

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"scheduler/internal/auth"
	"scheduler/internal/logging"
	"scheduler/internal/store"
	"time"
)

type pageData struct {
	Title  string
	Active string
}

type methodRouter struct {
	handlers map[string]http.Handler
}

func (m methodRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := m.handlers[r.Method]; ok {
		handler.ServeHTTP(w, r)
		return
	}

	if r.Method == http.MethodOptions {
		allowed := make([]string, 0, len(m.handlers))
		for method := range m.handlers {
			allowed = append(allowed, method)
		}
		w.Header().Set("Allow", fmt.Sprintf("%s", allowed))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Allow", allowedMethods(m.handlers))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func allowedMethods(handlers map[string]http.Handler) string {
	methods := make([]string, 0, len(handlers))
	for method := range handlers {
		methods = append(methods, method)
	}
	return fmt.Sprintf("%s", methods)
}

func NewRouter(logger *logging.Manager, authManager *auth.Manager, repositories *store.Store) http.Handler {
	templates, err := parseTemplates()
	if err != nil {
		panic(fmt.Errorf("parse templates: %w", err))
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	authHandlers := newAuthHandler(authManager, repositories.Access)
	mux.Handle("/api/v1/auth/login", methodRouter{handlers: map[string]http.Handler{
		http.MethodPost: http.HandlerFunc(authHandlers.handleLogin),
	}})
	mux.Handle("/api/v1/auth/logout", methodRouter{handlers: map[string]http.Handler{
		http.MethodPost: http.HandlerFunc(authHandlers.handleLogout),
	}})
	mux.Handle("/api/v1/auth/me", methodRouter{handlers: map[string]http.Handler{
		http.MethodGet: http.HandlerFunc(authHandlers.handleMe),
	}})

	adminUsers := newAdminUsersHandler(repositories.Users, authManager)
	adminWorkstations := newAdminWorkstationsHandler(repositories.Workstations, repositories.Access)
	mux.Handle("/api/v1/admin/users", methodRouter{handlers: map[string]http.Handler{
		http.MethodGet:  requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminUsers.handleUsers)),
		http.MethodPost: requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminUsers.handleUsers))),
	}})
	mux.Handle("/api/v1/admin/users/", methodRouter{handlers: map[string]http.Handler{
		http.MethodGet:    requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminUsers.handleUser)),
		http.MethodPatch:  requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminUsers.handleUser))),
		http.MethodDelete: requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminUsers.handleUser))),
	}})
	mux.Handle("/api/v1/admin/workstations", methodRouter{handlers: map[string]http.Handler{
		http.MethodGet:  requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminWorkstations.handleWorkstations)),
		http.MethodPost: requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstations))),
	}})
	mux.Handle("/api/v1/admin/workstations/", methodRouter{handlers: map[string]http.Handler{
		http.MethodGet:    requireRole(string(store.RoleAdmin), authManager, http.HandlerFunc(adminWorkstations.handleWorkstation)),
		http.MethodPatch:  requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstation))),
		http.MethodDelete: requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstation))),
		http.MethodPut:    requireRole(string(store.RoleAdmin), authManager, requireCSRF(authManager, http.HandlerFunc(adminWorkstations.handleWorkstation))),
	}})

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join("web", "static")))))
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("docs"))))
	mux.HandleFunc("/dashboard", pageHandler(templates, "dashboard", pageData{Title: "Scheduler Dashboard", Active: "dashboard"}))
	mux.HandleFunc("/admin", pageHandler(templates, "admin", pageData{Title: "Admin Configuration", Active: "admin"}))
	mux.HandleFunc("/users", pageHandler(templates, "users", pageData{Title: "User Management", Active: "users"}))
	mux.HandleFunc("/workstations", pageHandler(templates, "workstations", pageData{Title: "Workstation Management", Active: "workstations"}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		pageHandler(templates, "dashboard", pageData{Title: "Scheduler Dashboard", Active: "dashboard"})(w, r)
	})

	return requestLogger(logger, mux)
}

func parseTemplates() (map[string]*template.Template, error) {
	funcMap := template.FuncMap{
		"eq":     func(a, b string) bool { return a == b },
		"printf": fmt.Sprintf,
	}

	basePath := filepath.Join("web", "templates", "base.html")
	base, err := template.New("base").Funcs(funcMap).ParseFiles(basePath)
	if err != nil {
		return nil, err
	}

	pages := map[string]string{
		"dashboard":    "dashboard.html",
		"admin":        "admin.html",
		"users":        "users.html",
		"workstations": "workstations.html",
	}

	templates := make(map[string]*template.Template, len(pages))
	for name, pageFile := range pages {
		tpl, err := base.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := tpl.ParseFiles(filepath.Join("web", "templates", pageFile)); err != nil {
			return nil, err
		}
		templates[name] = tpl
	}

	return templates, nil
}

func pageHandler(templates map[string]*template.Template, templateName string, data pageData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tpl, ok := templates[templateName]
		if !ok {
			http.Error(w, "template not found", http.StatusInternalServerError)
			return
		}

		if err := tpl.ExecuteTemplate(w, templateName, data); err != nil {
			http.Error(w, "template render error", http.StatusInternalServerError)
			return
		}
	}
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
