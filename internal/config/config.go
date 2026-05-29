package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
)

// Config holds all application runtime configuration.
type Config struct {
	ListenAddr      string
	DatabaseURL     string
	SessionSecret   string
	JobBossClientID string
	JobBossSecret   string
	JobBossBaseURL  string
	JobBossTokenURL string
	Env             string
}

// FromEnv loads and validates config from environment variables.
func FromEnv() (Config, error) {
	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ListenAddr:      getOrDefault("SCHEDULER_LISTEN_ADDR", ":8080"),
		DatabaseURL:     databaseURL,
		SessionSecret:   os.Getenv("SCHEDULER_SESSION_SECRET"),
		JobBossClientID: os.Getenv("JB2_CLIENT_ID"),
		JobBossSecret:   os.Getenv("JB2_CLIENT_SECRET"),
		JobBossBaseURL:  getOrDefault("JB2_BASE_URL", "https://api-jb2.integrations.ecimanufacturing.com"),
		JobBossTokenURL: getOrDefault("JB2_TOKEN_URL", "https://api-user.integrations.ecimanufacturing.com/oauth2/api-user/token"),
		Env:             getOrDefault("SCHEDULER_ENV", "development"),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.DatabaseURL == "" {
		return errors.New("database config is required: set SCHEDULER_DATABASE_URL or discrete SCHEDULER_DB_* variables")
	}
	if c.SessionSecret == "" {
		return errors.New("SCHEDULER_SESSION_SECRET is required")
	}
	if c.JobBossClientID == "" {
		return errors.New("JB2_CLIENT_ID is required")
	}
	if c.JobBossSecret == "" {
		return errors.New("JB2_CLIENT_SECRET is required")
	}
	if len(c.SessionSecret) < 32 {
		return fmt.Errorf("SCHEDULER_SESSION_SECRET must be at least 32 characters")
	}
	return nil
}

func getOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func resolveDatabaseURL() (string, error) {
	if value := os.Getenv("SCHEDULER_DATABASE_URL"); value != "" {
		return value, nil
	}

	user := os.Getenv("SCHEDULER_DB_USER")
	password := os.Getenv("SCHEDULER_DB_PASSWORD")
	host := os.Getenv("SCHEDULER_DB_HOST")
	port := getOrDefault("SCHEDULER_DB_PORT", "5432")
	dbName := firstNonEmpty(os.Getenv("SCHEDULER_DB_NAME"), os.Getenv("SCHEDULER_DB_TABLE"))
	sslMode := getOrDefault("SCHEDULER_DB_SSLMODE", "disable")

	missing := make([]string, 0, 4)
	if user == "" {
		missing = append(missing, "SCHEDULER_DB_USER")
	}
	if password == "" {
		missing = append(missing, "SCHEDULER_DB_PASSWORD")
	}
	if host == "" {
		missing = append(missing, "SCHEDULER_DB_HOST")
	}
	if dbName == "" {
		missing = append(missing, "SCHEDULER_DB_NAME (or SCHEDULER_DB_TABLE)")
	}

	if len(missing) > 0 {
		return "", fmt.Errorf("database config is incomplete: set SCHEDULER_DATABASE_URL or provide %v", missing)
	}

	hostPort := net.JoinHostPort(host, port)
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   hostPort,
		Path:   "/" + dbName,
	}

	q := dsn.Query()
	q.Set("sslmode", sslMode)
	dsn.RawQuery = q.Encode()

	return dsn.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
