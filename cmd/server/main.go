package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/analysis"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/auth"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/dashboard"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/config"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/database"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/dataset"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/llm"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/seed"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/server"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		slog.Error("load .env", "error", err)
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	logger := newLogger(cfg)
	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func newLogger(cfg config.Config) *slog.Logger {
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func run(cfg config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	meterRepo := meter.NewSQLiteRepository(db)
	readingRepo := reading.NewSQLiteRepository(db)
	eventRepo := event.NewSQLiteRepository(db)
	anomalyRepo := anomaly.NewSQLiteRepository(db)
	runRepo := analysis.NewSQLiteRunRepository(db)

	if err := seedDatabase(ctx, cfg, logger, meterRepo, readingRepo, eventRepo); err != nil {
		return err
	}

	anomalySvc := anomaly.NewService(anomalyRepo)
	meterSvc := meter.NewService(meterRepo, readingRepo, anomalyRepo, eventRepo)
	analysisSvc := analysis.NewService(ctx, analysis.Dependencies{
		Runs:      runRepo,
		Meters:    meterRepo,
		Readings:  readingRepo,
		Events:    eventRepo,
		Anomalies: anomalyRepo,
		Explainer: newExplainer(ctx, cfg, logger),
		StepDelay: cfg.AnalysisStepDelay,
		Logger:    logger,
	})
	dashboardSvc := dashboard.NewService(meterSvc, readingRepo, anomalySvc, analysisSvc)
	authSvc := auth.NewService(auth.Config{
		Email:    cfg.DemoEmail,
		Password: cfg.DemoPassword,
		Name:     cfg.DemoName,
		Secret:   []byte(cfg.JWTSecret),
		TTL:      cfg.TokenTTL,
	})
	authHandler := auth.NewHandler(authSvc)

	router := server.NewRouter(server.RouterConfig{
		Public: []server.Registrar{server.RegistrarFunc(authHandler.RegisterPublic)},
		Protected: []server.Registrar{
			server.RegistrarFunc(authHandler.RegisterProtected),
			meter.NewHandler(meterSvc),
			event.NewHandler(eventRepo),
			anomaly.NewHandler(anomalySvc),
			analysis.NewHandler(analysisSvc),
			dashboard.NewHandler(dashboardSvc),
		},
		Auth:           auth.Middleware(authSvc),
		StaticDir:      cfg.StaticDir,
		AllowedOrigins: cfg.AllowedOrigins,
		Logger:         logger,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	analysisSvc.Wait()
	return nil
}

func seedDatabase(ctx context.Context, cfg config.Config, logger *slog.Logger, meters *meter.SQLiteRepository, readings *reading.SQLiteRepository, events *event.SQLiteRepository) error {
	count, err := meters.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	data, err := dataset.LoadDir(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load dataset from %s: %w", cfg.DataDir, err)
	}
	if _, err := seed.New(meters, readings, events).Run(ctx, data); err != nil {
		return err
	}
	logger.Info("database seeded", "meters", len(data.Meters), "readings", len(data.Readings), "events", len(data.Events))
	return nil
}

func newExplainer(ctx context.Context, cfg config.Config, logger *slog.Logger) analysis.Explainer {
	gen, err := llm.NewGenaiGenerator(ctx, cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		logger.Warn("gemini disabled, using rule-based explanations", "reason", err)
		return nil
	}
	logger.Info("gemini enabled", "model", cfg.GeminiModel)
	return llm.NewExplainer(gen, llm.Options{
		Model:      cfg.GeminiModel,
		Timeout:    cfg.GeminiTimeout,
		MaxRetries: cfg.GeminiMaxRetries,
		Backoff:    time.Second,
	})
}
