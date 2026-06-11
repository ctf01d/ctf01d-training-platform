package main

import (
	"context"
	"database/sql"
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
	ctf01dsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/ctf01d"
	gameteamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/gameteams"
	gamesvc "github.com/ctf01d/ctf01d-training-platform/internal/service/games"
	membersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/memberships"
	resultsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/results"
	scoreboardsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/scoreboard"
	svcsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/services"
	teamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/teams"
	unisvc "github.com/ctf01d/ctf01d-training-platform/internal/service/universities"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
	"github.com/ctf01d/ctf01d-training-platform/internal/server"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/handler"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
	"github.com/ctf01d/ctf01d-training-platform/pkg/logger"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("RUN_MIGRATIONS") == "true" {
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL is required when RUN_MIGRATIONS=true")
		}
		if err := goose.SetDialect("postgres"); err != nil {
			return fmt.Errorf("setting goose dialect: %w", err)
		}
		db, err := sql.Open("pgx", dbURL)
		if err != nil {
			return fmt.Errorf("opening DB for migrations: %w", err)
		}
		defer db.Close()
		if err := goose.Up(db, "migrations"); err != nil {
			return fmt.Errorf("running migrations: %w", err)
		}
		fmt.Println("migrations applied successfully")
	}

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

	fileStorage, err := storage.NewLocalStorage(cfg.Storage.Dir)
	if err != nil {
		return fmt.Errorf("creating file storage: %w", err)
	}

	jwtMgr := auth.NewManager(cfg.JWT.Secret, cfg.JWT.TTLHours)
	userService := usersvc.NewService(store.Queries)
	authService := authsvc.NewService(store.Queries, jwtMgr, &auth.PasswordCheckerImpl{})
	universityService := unisvc.NewService(store.Queries)
	teamService := teamsvc.NewService(store.Queries, store.Queries, store.Queries, store)
	membershipService := membersvc.NewService(store.Queries, store.Queries, store.Queries, store)
	gameService := gamesvc.NewService(store.Queries, store.Queries, store.Queries, store.Queries, store)
	gameTeamService := gameteamsvc.NewService(store.Queries, store)
	resultService := resultsvc.NewService(store.Queries, store.Queries)
	scoreboardService := scoreboardsvc.NewService(store.Queries, store.Queries, store.Queries, store.Queries)
	svcService := svcsvc.NewService(store.Queries)
	svcArchives := svcsvc.NewArchiveService(store.Queries, fileStorage, cfg.Storage.MaxUploadBytes)
	svcChecker := svcsvc.NewCheckerService(store.Queries, fileStorage)
	svcImport := svcsvc.NewImportService(store.Queries, fileStorage, cfg.Storage.MaxUploadBytes)
	ctf01dBuilder := ctf01dsvc.NewBuilder(store.Queries)
	h := handler.New(userService, authService, jwtMgr, universityService, teamService, membershipService, gameService, gameTeamService, resultService, scoreboardService, store.Queries, svcService, svcArchives, svcChecker, svcImport, ctf01dBuilder, cfg.Storage.MaxUploadBytes, cfg.Storage.Dir)

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
