package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	authsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/auth"
	membersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/memberships"
	teamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/teams"
	unisvc "github.com/ctf01d/ctf01d-training-platform/internal/service/universities"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
	"github.com/ctf01d/ctf01d-training-platform/internal/server"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/handler"
	"github.com/ctf01d/ctf01d-training-platform/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log, err := logger.New(cfg.Env, cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}
	defer logger.Sync(log)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := repository.NewStore(ctx, cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer store.Close()

	jwtMgr := auth.NewManager(cfg.JWT.Secret, cfg.JWT.TTLHours)
	userService := usersvc.NewService(store.Queries)
	authService := authsvc.NewService(store.Queries, jwtMgr, &auth.PasswordCheckerImpl{})
	universityService := unisvc.NewService(store.Queries)
	teamService := teamsvc.NewService(store.Queries, store.Queries, store.Queries, store)
	membershipService := membersvc.NewService(store.Queries, store.Queries, store.Queries, store)
	h := handler.New(userService, authService, jwtMgr, universityService, teamService, membershipService)

	engine := server.New(cfg, log, store, h)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("starting server", zap.String("addr", cfg.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	if err := srv.Close(); err != nil {
		log.Error("server close error", zap.Error(err))
	}

	log.Info("server stopped")
	return nil
}
