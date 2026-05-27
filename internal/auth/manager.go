package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"scheduler/internal/store"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrSessionExpired  = errors.New("session expired")
)

// Config controls local authentication behavior.
type Config struct {
	SessionLifetime time.Duration
}

// Session represents an authenticated browser session.
type Session struct {
	ID        string
	UserID    string
	Username  string
	Role      store.UserRole
	CSRFToken string
	ExpiresAt time.Time
}

// Manager handles local account authentication and session lifecycle.
type Manager struct {
	mu       sync.RWMutex
	users    *store.UserStore
	sessions *store.AuthStore
	secret   []byte
	config   Config
}

// NewManager creates an auth manager backed by the shared store.
func NewManager(users *store.UserStore, sessions *store.AuthStore, secret string) *Manager {
	return &Manager{
		users:    users,
		sessions: sessions,
		secret:   []byte(secret),
		config:   Config{SessionLifetime: 8 * time.Hour},
	}
}

// SetSessionLifetime overrides the default session lifetime.
func (m *Manager) SetSessionLifetime(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.SessionLifetime = d
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword verifies a password against a bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Authenticate validates credentials and creates a new session.
func (m *Manager) Authenticate(ctx context.Context, username, password string) (*Session, error) {
	user, err := m.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if !user.IsActive {
		return nil, ErrUnauthorized
	}
	if !CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidPassword
	}

	session, err := m.newSession(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}

	if err := m.sessions.CreateSession(ctx, &store.Session{
		ID:        session.ID,
		UserID:    session.UserID,
		Username:  session.Username,
		Role:      session.Role,
		CSRFToken: session.CSRFToken,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, err
	}

	return session, nil
}

// ValidateSession returns a current session if it exists and has not expired.
func (m *Manager) ValidateSession(ctx context.Context, sessionID string) (*Session, error) {
	row, err := m.sessions.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if time.Now().After(row.ExpiresAt) {
		_ = m.sessions.DeleteSession(ctx, sessionID)
		return nil, ErrSessionExpired
	}

	return &Session{
		ID:        row.ID,
		UserID:    row.UserID,
		Username:  row.Username,
		Role:      row.Role,
		CSRFToken: row.CSRFToken,
		ExpiresAt: row.ExpiresAt,
	}, nil
}

// InvalidateSession removes a session.
func (m *Manager) InvalidateSession(ctx context.Context, sessionID string) {
	_ = m.sessions.DeleteSession(ctx, sessionID)
}

// NewSessionCookieValue creates a signed session token payload.
func (m *Manager) NewSessionCookieValue(sessionID string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(sessionID))
	signature := mac.Sum(nil)
	return sessionID + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// ParseSessionCookieValue validates and extracts the session ID from a cookie value.
func (m *Manager) ParseSessionCookieValue(cookieValue string) (string, error) {
	parts := splitCookieValue(cookieValue)
	if len(parts) != 2 {
		return "", ErrUnauthorized
	}
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrUnauthorized
	}
	if !hmac.Equal(expected, actual) {
		return "", ErrUnauthorized
	}
	return parts[0], nil
}

func (m *Manager) newSession(userID, username string, role store.UserRole) (*Session, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}
	sessionID := base64.RawURLEncoding.EncodeToString(raw[:])
	csrfRaw := sha256.Sum256(raw[:])
	csrfToken := base64.RawURLEncoding.EncodeToString(csrfRaw[:])

	m.mu.RLock()
	lifetime := m.config.SessionLifetime
	m.mu.RUnlock()

	return &Session{
		ID:        sessionID,
		UserID:    userID,
		Username:  username,
		Role:      role,
		CSRFToken: csrfToken,
		ExpiresAt: time.Now().Add(lifetime),
	}, nil
}

func splitCookieValue(value string) []string {
	for i := 0; i < len(value); i++ {
		if value[i] == '.' {
			return []string{value[:i], value[i+1:]}
		}
	}
	return []string{value}
}
