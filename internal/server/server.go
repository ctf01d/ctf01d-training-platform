package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/handler"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
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
	requireAdmin := []gin.HandlerFunc{requireAuth, middleware.RequireRole("admin")}

	api.POST("/session", h.Login)
	api.DELETE("/session", requireAuth, h.Logout)
	api.GET("/profile", requireAuth, h.GetProfile)
	api.PATCH("/profile", requireAuth, h.UpdateProfile)

	api.GET("/users", requireAuth, h.HandleListUsers)
	api.POST("/users", append(requireAdmin, h.HandleCreateUser)...)
	api.GET("/users/:id", requireAuth, h.HandleGetUser)
	api.PATCH("/users/:id", requireAuth, h.HandleUpdateUser)
	api.PATCH("/users/:id/role", append(requireAdmin, h.HandleUpdateUserRole)...)
	api.DELETE("/users/:id", append(requireAdmin, h.HandleDeleteUser)...)

	api.GET("/universities", requireAuth, h.HandleListUniversities)
	api.POST("/universities", append(requireAdmin, h.HandleCreateUniversity)...)
	api.GET("/universities/:id", requireAuth, h.HandleGetUniversity)
	api.PATCH("/universities/:id", append(requireAdmin, h.HandleUpdateUniversity)...)
	api.DELETE("/universities/:id", append(requireAdmin, h.HandleDeleteUniversity)...)

	api.GET("/teams", requireAuth, h.HandleListTeams)
	api.POST("/teams", requireAuth, h.HandleCreateTeam)
	api.GET("/teams/:id", requireAuth, h.HandleGetTeam)
	api.PATCH("/teams/:id", requireAuth, h.HandleUpdateTeam)
	api.DELETE("/teams/:id", requireAuth, h.HandleDeleteTeam)
	api.POST("/teams/:id/join-request", requireAuth, h.HandleRequestJoinTeam)
	api.POST("/teams/:id/invite", requireAuth, h.HandleInviteToTeam)
	api.GET("/teams/:id/members", requireAuth, h.HandleListTeamMembers)
	api.GET("/teams/:id/events", requireAuth, h.HandleListTeamEvents)

	api.GET("/team-memberships", requireAuth, h.HandleListTeamMemberships)
	api.POST("/team-memberships", append(requireAdmin, h.HandleCreateTeamMembership)...)
	api.GET("/team-memberships/:id", requireAuth, h.HandleGetTeamMembership)
	api.PATCH("/team-memberships/:id", append(requireAdmin, h.HandleUpdateTeamMembership)...)
	api.DELETE("/team-memberships/:id", append(requireAdmin, h.HandleDeleteTeamMembership)...)
	api.POST("/team-memberships/:id/approve", requireAuth, h.HandleApproveTeamMembership)
	api.POST("/team-memberships/:id/reject", requireAuth, h.HandleRejectTeamMembership)
	api.POST("/team-memberships/:id/accept", requireAuth, h.HandleAcceptTeamMembership)
	api.POST("/team-memberships/:id/decline", requireAuth, h.HandleDeclineTeamMembership)
	api.POST("/team-memberships/:id/set-role", requireAuth, h.HandleSetTeamMembershipRole)

	requirePlayer := []gin.HandlerFunc{requireAuth, middleware.RequireRole("player")}

	api.GET("/games", requireAuth, h.HandleListGames)
	api.POST("/games", append(requirePlayer, h.HandleCreateGame)...)
	api.GET("/games/:id", requireAuth, h.HandleGetGame)
	api.PATCH("/games/:id", append(requirePlayer, h.HandleUpdateGame)...)
	api.DELETE("/games/:id", append(requirePlayer, h.HandleDeleteGame)...)
	api.POST("/games/:id/finalize", append(requirePlayer, h.HandleFinalizeGame)...)
	api.POST("/games/:id/unfinalize", append(requirePlayer, h.HandleUnfinalizeGame)...)
	api.GET("/games/:id/services", requireAuth, h.HandleListGameServices)
	api.POST("/games/:id/services", append(requirePlayer, h.HandleAddGameService)...)
	api.DELETE("/games/:id/services/:service_id", append(requirePlayer, h.HandleRemoveGameService)...)
	api.GET("/games/:id/teams", requireAuth, h.HandleListGameTeams)
	api.POST("/games/:id/teams/reorder", append(requirePlayer, h.HandleReorderGameTeams)...)
	api.GET("/games/:id/scoreboard", middleware.OptionalAuth(h.JWTMgr()), h.HandleGetGameScoreboard)
	api.GET("/games/:id/export/ctf01d/options", append(requirePlayer, h.HandleGetCtf01dExportOptions)...)
	api.POST("/games/:id/export/ctf01d", append(requirePlayer, h.HandleExportCtf01d)...)

	api.POST("/game-teams", append(requirePlayer, h.HandleCreateGameTeam)...)
	api.PATCH("/game-teams/:id", append(requirePlayer, h.HandleUpdateGameTeam)...)
	api.DELETE("/game-teams/:id", append(requirePlayer, h.HandleDeleteGameTeam)...)

	api.GET("/results", requireAuth, h.HandleListResults)
	api.POST("/results", append(requirePlayer, h.HandleCreateResult)...)
	api.GET("/results/:id", requireAuth, h.HandleGetResult)
	api.PATCH("/results/:id", append(requirePlayer, h.HandleUpdateResult)...)
	api.DELETE("/results/:id", append(requirePlayer, h.HandleDeleteResult)...)

	api.GET("/writeups", requireAuth, h.HandleListWriteups)
	api.POST("/writeups", requireAuth, h.HandleCreateWriteup)
	api.GET("/writeups/:id", requireAuth, h.HandleGetWriteup)
	api.DELETE("/writeups/:id", requireAuth, h.HandleDeleteWriteup)

	api.GET("/scoreboard", requireAuth, h.HandleGetGlobalScoreboard)

	api.GET("/services", middleware.OptionalAuth(h.JWTMgr()), h.HandleListServices)
	api.POST("/services", append(requirePlayer, h.HandleCreateService)...)
	api.POST("/services/import/github", append(requirePlayer, h.HandleImportServiceFromGithub)...)
	api.POST("/services/import/zip", append(requirePlayer, h.HandleImportServiceFromZip)...)
	api.DELETE("/services/:id", append(requirePlayer, h.HandleDeleteService)...)
	api.GET("/services/:id", middleware.OptionalAuth(h.JWTMgr()), h.HandleGetService)
	api.PATCH("/services/:id", append(requirePlayer, h.HandleUpdateService)...)
	api.POST("/services/:id/check-checker", append(requirePlayer, h.HandleCheckServiceChecker)...)
	api.GET("/services/:id/download/:kind", requireAuth, h.HandleDownloadServiceArchive)
	api.POST("/services/:id/redownload", append(requirePlayer, h.HandleRedownloadServiceArchives)...)
	api.POST("/services/:id/toggle-public", append(requirePlayer, h.HandleToggleServicePublic)...)
	api.POST("/services/:id/upload-archives", append(requirePlayer, h.HandleUploadServiceArchives)...)

	return engine
}

var Version = "dev"
