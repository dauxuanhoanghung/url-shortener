package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/internal/config"
	"github.com/dauxuanhoanghung/url-shortener/internal/database"
	"github.com/dauxuanhoanghung/url-shortener/internal/handler"
	"github.com/dauxuanhoanghung/url-shortener/internal/mailer"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/internal/router"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/dauxuanhoanghung/url-shortener/internal/sse"
	"github.com/dauxuanhoanghung/url-shortener/internal/worker"
	"github.com/dauxuanhoanghung/url-shortener/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Server.Mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect to PostgreSQL
	pgPool, err := database.NewPostgresPool(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer pgPool.Close()
	log.Info("connected to PostgreSQL")

	// Build cache (Redis primary + in-memory fallback). See docs/23-backend-architecture.md.
	appCache, err := cache.New(ctx, cache.Config{
		Driver:   "redis",
		Addr:     cfg.Redis.Addr(),
		Fallback: true,
	})
	if err != nil {
		log.Fatal("failed to init cache", zap.Error(err))
	}
	defer appCache.Close()
	log.Info("cache ready", zap.String("driver", "redis"), zap.Bool("fallback", true))

	// Dev-only console mailer. Swap to a real provider in prod. See docs/24-user-account-lifecycle.md §5.
	appMailer := mailer.NewConsoleMailer(log)

	userRepo     := repository.NewUserRepository(pgPool)
	userPlanRepo := repository.NewUserPlanRepository(pgPool)
	urlRepo      := repository.NewURLRepository(pgPool)
	metaRepo     := repository.NewURLMetadataRepository(pgPool)
	tokenRepo    := repository.NewTokenRepository(pgPool)
	planRepo     := repository.NewPlanRepository(pgPool)

	sseHub := sse.NewHub()

	metaWorker := worker.NewMetadataWorker(worker.MetadataWorkerConfig{
		MetaRepo: metaRepo,
		URLRepo:  urlRepo,
		Cache:    appCache,
		Notifier: sseHub,
		Logger:   log,
		Workers:  4,
	})
	metaWorker.Start(ctx)
	defer metaWorker.Stop()

	authService     := service.NewAuthService(userRepo, userPlanRepo, tokenRepo, appMailer, cfg.JWT.Secret, cfg.Server.FrontendBaseURL)
	urlService      := service.NewURLService(urlRepo, metaRepo, userPlanRepo, planRepo, metaWorker)
	planService     := service.NewPlanService(planRepo)
	redirectService := service.NewRedirectService(urlRepo, appCache)

	authHandler     := handler.NewAuthHandler(authService)
	urlHandler      := handler.NewURLHandler(urlService, cfg.Server.BaseURL)
	planHandler     := handler.NewPlanHandler(planService)
	redirectHandler := handler.NewRedirectHandler(redirectService)
	sseHandler      := handler.NewSSEHandler(sseHub)

	r := router.Setup(cfg.Server.Mode, cfg.JWT.Secret, userRepo, router.Handlers{
		Auth:     authHandler,
		URL:      urlHandler,
		Plan:     planHandler,
		Redirect: redirectHandler,
		SSE:      sseHandler,
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: r,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("starting server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatal("server failed", zap.Error(err))
	case <-ctx.Done():
		log.Info("shutting down server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}
	log.Info("server stopped")
}
