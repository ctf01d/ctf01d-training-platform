package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/handler"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
)

type Store interface {
	Ping() error
}

const corsMaxAge = 12 * time.Hour

func New(cfg *config.Config, log *zap.Logger, store Store, h *handler.Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	if cfg.Env == "development" {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(requestid.New())
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(cfg.CORS.AllowedOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           corsMaxAge,
	}))
	engine.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()
		latency := time.Since(start)
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("request_id", requestid.Get(c)),
		)
	})

	engine.GET("/healthz", func(c *gin.Context) {
		if err := store.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	engine.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})

	httpserver.RegisterHandlersWithOptions(engine, h, httpserver.GinServerOptions{
		BaseURL: "/api/v1",
		Middlewares: []httpserver.MiddlewareFunc{
			middleware.OpenAPIAuth(h.JWTMgr(), h.SessionChecker()),
			middleware.OpenAPIRole(),
		},
		ErrorHandler: middleware.OpenAPIErrorHandler,
	})

	return engine
}

var Version = "dev"
