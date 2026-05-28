package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"scheduler/internal/auth"
	"scheduler/internal/config"
	"scheduler/internal/db"
	"scheduler/internal/httpapi"
	"scheduler/internal/logging"
	"scheduler/internal/store"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	fmt.Println("Loading configuration...")
	cfg, err := config.FromEnv()
	if err != nil {
		panic(err)
	}

	fmt.Println("Starting scheduler...")
	logManager := logging.GetManager()
	logCfg := logging.DefaultConfig()
	logCfg.DebugEnabled = cfg.Env == "development"
	logCfg.VerboseEnabled = cfg.Env == "development"
	logCfg.LogToStdout = true
	logCfg.LogToFile = true
	logCfg.SyslogEnabled = false
	logCfg.SyslogTag = "scheduler"
	logCfg.FilePath = "./data/scheduler.log"

	fmt.Println("Configuring logger...")
	if err := logManager.Configure(logCfg); err != nil {
		fmt.Println("Failed to configure logger")
		fmt.Printf("Error: %v\n", err)
		panic(fmt.Errorf("configure logger: %w", err))
	}
	defer logManager.Close()

	fmt.Println("Connecting to database...")
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		fmt.Println("Failed to open database")
		fmt.Printf("Error: %v\n", err)
		logManager.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	fmt.Println("Checking database connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		fmt.Println("Failed to connect to database")
		fmt.Printf("Error: %v\n", err)
		logManager.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	fmt.Println("Running migrations...")
	if err := db.RunMigrations(ctx, sqlDB); err != nil {
		fmt.Println("Failed to run migrations")
		fmt.Printf("Error: %v\n", err)
		logManager.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	repositories := store.New(sqlDB)
	if err := ensureDefaultAdmin(ctx, repositories.Users, logManager); err != nil {
		fmt.Println("Failed to seed default admin user")
		fmt.Printf("Error: %v\n", err)
		logManager.Error("failed to seed default admin user", "error", err)
		os.Exit(1)
	}
	authManager := auth.NewManager(repositories.Users, repositories.Sessions, cfg.SessionSecret)
	authManager.SetSessionLifetime(8 * time.Hour)
	handler := httpapi.NewRouter(logManager, authManager, repositories)
	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logManager.Info("starting server", "addr", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logManager.Error("server crashed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	logManager.Info("shutting down server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logManager.Error("graceful shutdown failed", "error", err)
	}
}

func ensureDefaultAdmin(ctx context.Context, users *store.UserStore, logger *logging.Manager) error {
	exists, err := users.Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	username := getEnvOrDefault("SCHEDULER_DEFAULT_ADMIN_USERNAME", "admin")
	password := os.Getenv("SCHEDULER_DEFAULT_ADMIN_PASSWORD")
	if password == "" {
		password = "admin123"
	}

	logger.Info("seeding default admin user", "username", username)

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash default admin password: %w", err)
	}

	userID, err := newUUID()
	if err != nil {
		return fmt.Errorf("generate default admin id: %w", err)
	}

	now := time.Now().UTC()
	if err := users.Create(ctx, &store.User{
		ID:           userID,
		Username:     username,
		PasswordHash: hash,
		Role:         store.RoleAdmin,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		return fmt.Errorf("create default admin user: %w", err)
	}

	logger.Info("default admin user seeded", "username", username)
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
