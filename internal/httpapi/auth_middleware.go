package httpapi

import (
	"context"
	"net/http"
	"strings"

	"scheduler/internal/auth"
)

type contextKey string

const (
	sessionCookieName   = "scheduler_session"
	csrfCookieName      = "scheduler_csrf"
	contextKeySession   contextKey = "session"
	csrfHeaderName                 = "X-CSRF-Token"
)

func requireAuth(authManager *auth.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := currentSession(r, authManager)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		ctx := context.WithValue(r.Context(), contextKeySession, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireRole(role string, authManager *auth.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := currentSession(r, authManager)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if string(session.Role) != role {
			jsonError(w, http.StatusForbidden, "forbidden")
			return
		}

		ctx := context.WithValue(r.Context(), contextKeySession, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireCSRF(authManager *auth.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		session, err := currentSession(r, authManager)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if strings.TrimSpace(r.Header.Get(csrfHeaderName)) != session.CSRFToken {
			jsonError(w, http.StatusForbidden, "invalid csrf token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func currentSession(r *http.Request, authManager *auth.Manager) (*auth.Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, auth.ErrUnauthorized
	}
	sessionID, err := authManager.ParseSessionCookieValue(cookie.Value)
	if err != nil {
		return nil, err
	}
	return authManager.ValidateSession(r.Context(), sessionID)
}

func sessionFromContext(ctx context.Context) (*auth.Session, bool) {
	session, ok := ctx.Value(contextKeySession).(*auth.Session)
	return session, ok
}

func setSessionCookies(w http.ResponseWriter, authManager *auth.Manager, session *auth.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    authManager.NewSessionCookieValue(session.ID),
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    session.CSRFToken,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})
}

func clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: "", Path: "/", MaxAge: -1})
}
