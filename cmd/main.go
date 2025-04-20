package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ctf01d/internal/config"
	"ctf01d/internal/handler"
	"ctf01d/internal/httpserver"
	"ctf01d/internal/middleware/auth"
	migration "ctf01d/internal/migrations/psql"
	"ctf01d/internal/repository"
	"ctf01d/pkg/ginmiddleware"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	_ "go.uber.org/automaxprocs"
)

func main() {
	path := "./configs/config.development.yml"
	if envPath, exists := os.LookupEnv("CONFIG_PATH"); exists {
		path = envPath
	}

	cfg, err := config.New(path)
	if err != nil {
		slog.Error("Config error: " + err.Error())
		os.Exit(1)
	}
	cfgLog := cfg.Log
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.Level(cfg.ParseLogLevel(cfgLog.Level)),
	}))
	slog.SetDefault(logger)
	slog.Info("Config path is - " + path)

	// Подключение к БД
	db, err := migration.SetupDatabase(cfg)
	if err != nil {
		slog.Error("Database connection error: " + err.Error())
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Database connection established successfully")
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	// Добавление CORS middleware
	router.Use(cors.Default())

	// Загрузка спецификации OpenAPI
	swagger, err := openapi3.NewLoader().LoadFromFile("api/openapi.yaml")
	if err != nil {
		log.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	// OpenAPI middleware валидации запросов
	requestValidator := ginmiddleware.OapiRequestValidatorWithOptions(swagger, &ginmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: func(_ context.Context, _ *openapi3filter.AuthenticationInput) error {
				return nil
			},
		},
	})

	// OpenAPI middleware валидации ответов
	responseValidator := ginmiddleware.OapiResponseValidator(swagger)
	hndlr := &handler.Handler{DB: db}

	// API-группа, к которой применяются валидаторы
	apiGroup := router.Group("/", requestValidator, responseValidator)
	sessionRepo := repository.NewSessionRepository(db)
	options := httpserver.GinServerOptions{
		Middlewares: []httpserver.MiddlewareFunc{
			auth.AuthenticationMiddleware(sessionRepo),
		},
	}
	httpserver.RegisterHandlersWithOptions(apiGroup, hndlr, options)

	// HTML маршрутизатор для корня
	router.GET("/", httpserver.NewSSRRouter())
	router.GET("/games", httpserver.NewSSRRouter())
	router.GET("/games/", httpserver.NewSSRRouter())
	router.GET("/teams", httpserver.NewSSRRouter())
	router.GET("/teams/", httpserver.NewSSRRouter())
	router.GET("/users", httpserver.NewSSRRouter())
	router.GET("/users/", httpserver.NewSSRRouter())

	// Для всего остального (404)
	router.NoRoute(httpserver.NewSSRRouter())

	// Для статики (если нужно)
	router.Static("/static", "./static")

	// Запуск сервера
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	slog.Info("Server starting on", slog.String("address", addr))

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	slog.Info("Server is running", slog.String("address", addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Server exited gracefully")
}
