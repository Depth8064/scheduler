package config

import (
	"errors"
	"fmt"
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
	cfg := Config{
		ListenAddr:      getOrDefault("SCHEDULER_LISTEN_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("SCHEDULER_DATABASE_URL"),
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
		return errors.New("SCHEDULER_DATABASE_URL is required")
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
