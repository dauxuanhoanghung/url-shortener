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
	"github.com/dauxuanhoanghung/url-shortener/internal/event"
	eventhandler "github.com/dauxuanhoanghung/url-shortener/internal/event/handler"
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

	// Transport selected by MAIL_TRANSPORT env var; probed at startup with
	// automatic fallback to console. See docs/29-mailer-transports.md.
	appMailer := mailer.New(cfg.Mailer, log)

	userRepo     := repository.NewUserRepository(pgPool)
	userPlanRepo := repository.NewUserPlanRepository(pgPool)
	urlRepo      := repository.NewURLRepository(pgPool)
	metaRepo     := repository.NewURLMetadataRepository(pgPool)
	tagRepo      := repository.NewTagRepository(pgPool)
	tokenRepo    := repository.NewTokenRepository(pgPool)
	planRepo     := repository.NewPlanRepository(pgPool)
	auditRepo    := repository.NewAdminAuditRepository(pgPool)

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

	bus := event.NewBus(log)
	bus.Subscribe(event.TypeOf[event.UserRegistered](),
		eventhandler.SendVerificationEmail(tokenRepo, appMailer, cfg.Server.FrontendBaseURL),
		event.Async,
	)
	bus.Subscribe(event.TypeOf[event.PasswordResetRequested](),
		eventhandler.SendPasswordResetEmail(tokenRepo, appMailer, cfg.Server.FrontendBaseURL),
		event.Async,
	)
	bus.Subscribe(event.TypeOf[event.URLCreated](),
		eventhandler.EnqueueMetadataFetch(metaWorker),
		event.Async,
	)

	authService     := service.NewAuthService(userRepo, userPlanRepo, tokenRepo, bus, cfg.JWT.Secret)
	tagService      := service.NewTagService(tagRepo, urlRepo)
	urlService      := service.NewURLService(urlRepo, metaRepo, tagService, userPlanRepo, planRepo, bus)
	planService     := service.NewPlanService(planRepo)
	redirectService := service.NewRedirectService(urlRepo, appCache)
	adminService    := service.NewAdminService(userRepo, userPlanRepo, planRepo, auditRepo)

	authHandler     := handler.NewAuthHandler(authService)
	urlHandler      := handler.NewURLHandler(urlService, cfg.Server.BaseURL)
	tagHandler      := handler.NewTagHandler(tagService)
	planHandler     := handler.NewPlanHandler(planService)
	redirectHandler := handler.NewRedirectHandler(redirectService)
	sseHandler      := handler.NewSSEHandler(sseHub)
	adminHandler    := handler.NewAdminHandler(adminService)

	r := router.Setup(cfg.Server.Mode, cfg.JWT.Secret, userRepo, appCache, router.Handlers{
		Auth:     authHandler,
		URL:      urlHandler,
		Tag:      tagHandler,
		Plan:     planHandler,
		Redirect: redirectHandler,
		SSE:      sseHandler,
		Admin:    adminHandler,
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
