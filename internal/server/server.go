package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/handler"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Store interface {
	Ping() error
}

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
		MaxAge:           12 * time.Hour,
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

	api := engine.Group("/api/v1")
	requireAuth := middleware.RequireAuth(h.JWTMgr())
	requireAdmin := func(c *gin.Context) {
		requireAuth(c)
		if c.IsAborted() {
			return
		}
		middleware.RequireRole("admin")(c)
	}

	api.POST("/session", h.Login)
	api.DELETE("/session", requireAuth, h.Logout)
	api.GET("/profile", requireAuth, h.GetProfile)
	api.PATCH("/profile", requireAuth, h.UpdateProfile)

	api.GET("/users", requireAuth, h.HandleListUsers)
	api.POST("/users", requireAdmin, h.HandleCreateUser)
	api.GET("/users/:id", requireAuth, h.HandleGetUser)
	api.PATCH("/users/:id", requireAuth, h.HandleUpdateUser)
	api.DELETE("/users/:id", requireAdmin, h.HandleDeleteUser)

	return engine
}

var Version = "dev"
