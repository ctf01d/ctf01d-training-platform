package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	gamesvc "github.com/ctf01d/ctf01d-training-platform/internal/service/games"
)

func (h *Handler) HandleListGames(c *gin.Context) {
	page := 1
	perPage := 20
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("per_page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			perPage = p
		}
	}

	var q *string
	if v := c.Query("q"); v != "" {
		q = &v
	}

	result, err := h.games.List(c.Request.Context(), page, perPage, q)
	if err != nil {
		respondError(c, err)
		return
	}

	viewerRole, _ := middleware.CurrentRole(c)
	userID, hasUser := middleware.CurrentUserID(c)

	items := make([]httpserver.Game, len(result.Items))
	for i, g := range result.Items {
		items[i] = gameToHTTP(g, h.canAccessGameSecrets(c, g.ID, viewerRole, hasUser, userID))
	}

	c.JSON(http.StatusOK, httpserver.GameList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    result.Page,
			PerPage: result.PerPage,
			Total:   int(result.Total),
		},
	})
}

func (h *Handler) HandleCreateGame(c *gin.Context) {
	req, ok := bindJSON[httpserver.GameCreate](c)
	if !ok {
		return
	}

	params := gamesvc.CreateParams{
		Name:                 req.Name,
		Organizer:            req.Organizer,
		StartsAt:             req.StartsAt,
		EndsAt:               req.EndsAt,
		AvatarUrl:            req.AvatarUrl,
		SiteUrl:              req.SiteUrl,
		CtftimeUrl:           req.CtftimeUrl,
		RegistrationOpensAt:  req.RegistrationOpensAt,
		RegistrationClosesAt: req.RegistrationClosesAt,
		ScoreboardOpensAt:    req.ScoreboardOpensAt,
		ScoreboardClosesAt:   req.ScoreboardClosesAt,
		VpnUrl:               req.VpnUrl,
		VpnConfigUrl:         req.VpnConfigUrl,
		AccessInstructions:   req.AccessInstructions,
		AccessSecret:         req.AccessSecret,
	}

	game, err := h.games.Create(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}

	viewerRole, _ := middleware.CurrentRole(c)
	userID, hasUser := middleware.CurrentUserID(c)

	c.JSON(http.StatusCreated, gameToHTTP(*game, h.canAccessGameSecrets(c, game.ID, viewerRole, hasUser, userID)))
}

func (h *Handler) HandleGetGame(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	game, err := h.games.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	viewerRole, _ := middleware.CurrentRole(c)
	userID, hasUser := middleware.CurrentUserID(c)

	c.JSON(http.StatusOK, gameToHTTP(*game, h.canAccessGameSecrets(c, game.ID, viewerRole, hasUser, userID)))
}

func (h *Handler) HandleUpdateGame(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.GameUpdate](c)
	if !ok {
		return
	}

	params := gamesvc.UpdateParams{
		Name:                 req.Name,
		Organizer:            req.Organizer,
		StartsAt:             req.StartsAt,
		EndsAt:               req.EndsAt,
		AvatarUrl:            req.AvatarUrl,
		SiteUrl:              req.SiteUrl,
		CtftimeUrl:           req.CtftimeUrl,
		RegistrationOpensAt:  req.RegistrationOpensAt,
		RegistrationClosesAt: req.RegistrationClosesAt,
		ScoreboardOpensAt:    req.ScoreboardOpensAt,
		ScoreboardClosesAt:   req.ScoreboardClosesAt,
		VpnUrl:               req.VpnUrl,
		VpnConfigUrl:         req.VpnConfigUrl,
		AccessInstructions:   req.AccessInstructions,
		AccessSecret:         req.AccessSecret,
	}

	game, err := h.games.Update(c.Request.Context(), id, params)
	if err != nil {
		respondError(c, err)
		return
	}

	viewerRole, _ := middleware.CurrentRole(c)
	userID, hasUser := middleware.CurrentUserID(c)

	c.JSON(http.StatusOK, gameToHTTP(*game, h.canAccessGameSecrets(c, game.ID, viewerRole, hasUser, userID)))
}

func (h *Handler) HandleDeleteGame(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.games.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleFinalizeGame(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	game, err := h.games.Finalize(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	viewerRole, _ := middleware.CurrentRole(c)
	userID, hasUser := middleware.CurrentUserID(c)

	c.JSON(http.StatusOK, gameToHTTP(*game, h.canAccessGameSecrets(c, game.ID, viewerRole, hasUser, userID)))
}

func (h *Handler) HandleUnfinalizeGame(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	game, err := h.games.Unfinalize(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	viewerRole, _ := middleware.CurrentRole(c)
	userID, hasUser := middleware.CurrentUserID(c)

	c.JSON(http.StatusOK, gameToHTTP(*game, h.canAccessGameSecrets(c, game.ID, viewerRole, hasUser, userID)))
}

func (h *Handler) HandleListGameServices(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	serviceIDs, err := h.games.ListServices(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	ids := make([]int64, len(serviceIDs))
	copy(ids, serviceIDs)
	c.JSON(http.StatusOK, ids)
}

func (h *Handler) HandleAddGameService(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		ServiceId int64 `json:"service_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "invalid JSON"})
		return
	}

	if err := h.games.AddService(c.Request.Context(), id, req.ServiceId); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) HandleRemoveGameService(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	serviceID, ok := parseIDParam(c, "service_id")
	if !ok {
		return
	}

	if err := h.games.RemoveService(c.Request.Context(), id, serviceID); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func gameToHTTP(g gamesvc.Game, canAccessSecrets bool) httpserver.Game {
	result := httpserver.Game{
		Id:                   g.ID,
		Name:                 g.Name,
		Organizer:            g.Organizer,
		StartsAt:             g.StartsAt,
		EndsAt:               g.EndsAt,
		CreatedAt:            &g.CreatedAt,
		UpdatedAt:            &g.UpdatedAt,
		AvatarUrl:            g.AvatarUrl,
		SiteUrl:              g.SiteUrl,
		CtftimeUrl:           g.CtftimeUrl,
		Finalized:            g.Finalized,
		FinalizedAt:          g.FinalizedAt,
		RegistrationOpensAt:  g.RegistrationOpensAt,
		RegistrationClosesAt: g.RegistrationClosesAt,
		ScoreboardOpensAt:    g.ScoreboardOpensAt,
		ScoreboardClosesAt:   g.ScoreboardClosesAt,
		Status:               (*httpserver.GameStatus)(&g.Status),
		RegistrationStatus:   (*httpserver.GameRegistrationStatus)(&g.RegistrationStatus),
		ScoreboardStatus:     (*httpserver.GameScoreboardStatus)(&g.ScoreboardStatusVal),
	}
	if canAccessSecrets {
		result.AccessInstructions = g.AccessInstructions
		result.AccessSecret = g.AccessSecret
		result.VpnUrl = g.VpnUrl
		result.VpnConfigUrl = g.VpnConfigUrl
	}
	return result
}

func (h *Handler) canAccessGameSecrets(c *gin.Context, gameID int64, role string, hasUser bool, userID int64) bool {
	if role == roleAdmin {
		return true
	}
	if !hasUser {
		return false
	}
	approved, err := h.gameTeamsQ.IsUserApprovedInGameTeams(c.Request.Context(), db.IsUserApprovedInGameTeamsParams{
		GameID: gameID,
		UserID: userID,
	})
	if err != nil {
		return false
	}
	return approved
}
