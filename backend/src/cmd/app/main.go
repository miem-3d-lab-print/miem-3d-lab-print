// Package main — точка входа приложения MIEM 3D Lab Print Backend.
//
//	@title			MIEM 3D Lab Print API
//	@version		1.0.0
//	@description	REST API сервиса подачи заявок на 3D-печать МИЭМ НИУ ВШЭ.\nАутентификация — OTP через email + JWT (Bearer token).
//	@host			localhost:3000
//	@BasePath		/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Введите "Bearer <access_token>"
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

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/config"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/db"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/handlers"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/middleware"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/repository"
	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/services"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	appContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	database, err := db.Open(cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	sqlDB, err := db.Configure(appContext, database, cfg.DatabasePool)
	if err != nil {
		return fmt.Errorf("configure database: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Error("close database", "err", err)
		}
	}()

	storageService, err := services.NewStorageService(cfg.MinIO)
	if err != nil {
		return fmt.Errorf("initialize object storage: %w", err)
	}
	if err := storageService.EnsureBucket(appContext); err != nil {
		return fmt.Errorf("ensure object storage bucket: %w", err)
	}

	// Repositories
	txMgr := repository.NewTxManager(database)
	otpRepo := repository.NewGORMOTPRepository(database)
	rateLimitRepo := repository.NewInMemoryRateLimitRepository()
	userRepo := repository.NewGORMUserRepository(database)
	refreshTokenRepo := repository.NewGORMRefreshTokenRepository(database)
	materialRepo := repository.NewGORMMaterialRepository(database)
	colorRepo := repository.NewGORMColorRepository(database)
	appRepo := repository.NewGORMApplicationRepository(database)
	fileRepo := repository.NewGORMFileRepository(database)
	historyRepo := repository.NewGORMStatusHistoryRepository(database)
	statsRepo := repository.NewGORMStatsRepository(database)

	// Services
	jwtService := services.NewJWTService(cfg.JWT.Secret)
	emailService := services.NewEmailService(cfg.SMTP, cfg.SiteURL)
	authService := services.NewAuthService(
		logger, emailService, otpRepo, rateLimitRepo, userRepo, refreshTokenRepo, jwtService,
	)
	profileService := services.NewProfileService(userRepo)
	materialService := services.NewMaterialService(materialRepo, colorRepo)
	appService := services.NewApplicationService(
		logger, txMgr, appRepo, fileRepo, historyRepo, userRepo, materialService, storageService, emailService,
	)
	go func() {
		appService.CleanupExpiredPendingFiles(appContext)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				appService.CleanupExpiredPendingFiles(appContext)
			case <-appContext.Done():
				return
			}
		}
	}()
	adminService := services.NewAdminService(logger, txMgr, userRepo)
	statsService := services.NewStatsService(statsRepo)

	// Middleware
	authMW := middleware.Auth(jwtService)
	consentMW := middleware.RequireConsent(userRepo)
	adminMW := middleware.RequireAdmin
	corsMW := middleware.CORS(cfg.CORSOrigins)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, logger, cfg.TrustProxy)
	profileHandler := handlers.NewProfileHandler(profileService, logger)
	appHandler := handlers.NewApplicationHandler(appService, logger)
	materialHandler := handlers.NewMaterialHandler(materialService, logger)
	adminHandler := handlers.NewAdminHandler(adminService, statsService, logger)

	openapiYAML, err := os.ReadFile("docs/swagger.yaml")
	if err != nil {
		return fmt.Errorf("read OpenAPI specification: %w", err)
	}

	apiMux := http.NewServeMux()
	apiMux.Handle("GET /api/health/live", handlers.Live())
	apiMux.Handle("GET /api/health/ready", handlers.Ready(sqlDB))
	apiMux.Handle("GET /api/health", handlers.Ready(sqlDB))

	authHandler.Register(apiMux)
	profileHandler.Register(apiMux, authMW, consentMW)
	appHandler.Register(apiMux, authMW, consentMW, adminMW)
	materialHandler.Register(apiMux, authMW, consentMW, adminMW)
	adminHandler.Register(apiMux, authMW, adminMW)

	apiMux.HandleFunc("GET /api/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Header().Set("Content-Disposition", "inline; filename=openapi.yaml")
		_, _ = w.Write(openapiYAML)
	})

	server := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           corsMW(apiMux),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("starting server", "addr", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
	case <-appContext.Done():
		logger.Info("shutting down server")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}
