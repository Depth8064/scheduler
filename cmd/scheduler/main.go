package main

import (
	"context"
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
	cfg, err := config.FromEnv()
	if err != nil {
		panic(err)
	}

	logManager := logging.GetManager()
	logCfg := logging.DefaultConfig()
	logCfg.DebugEnabled = cfg.Env == "development"
	logCfg.VerboseEnabled = cfg.Env == "development"
	logCfg.LogToStdout = true
	logCfg.LogToFile = false
	logCfg.SyslogEnabled = false
	logCfg.SyslogTag = "scheduler"
	logCfg.FilePath = "./data/scheduler.log"

	if err := logManager.Configure(logCfg); err != nil {
		panic(fmt.Errorf("configure logger: %w", err))
	}
	defer logManager.Close()

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logManager.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		logManager.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := db.RunMigrations(ctx, sqlDB); err != nil {
		logManager.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	repositories := store.New(sqlDB)
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
