package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	writeupsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/writeups"
)

func (h *Handler) HandleListWriteups(c *gin.Context) {
	gameIDStr := c.Query("game_id")
	teamIDStr := c.Query("team_id")

	var writeups []writeupsvc.Writeup
	var err error

	switch {
	case gameIDStr != "" && teamIDStr != "":
		gameID, ok := parseIDQuery(c, "game_id")
		if !ok {
			return
		}
		teamID, ok := parseIDQuery(c, "team_id")
		if !ok {
			return
		}
		writeups, err = h.writeups.ListByGameAndTeam(c.Request.Context(), gameID, teamID)
	case gameIDStr != "":
		gameID, ok := parseIDQuery(c, "game_id")
		if !ok {
			return
		}
		writeups, err = h.writeups.ListByGame(c.Request.Context(), gameID)
	case teamIDStr != "":
		teamID, ok := parseIDQuery(c, "team_id")
		if !ok {
			return
		}
		writeups, err = h.writeups.ListByTeam(c.Request.Context(), teamID)
	default:
		writeups, err = h.writeups.ListAll(c.Request.Context())
	}

	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.Writeup, len(writeups))
	for i, w := range writeups {
		items[i] = writeupToHTTP(w)
	}

	c.JSON(http.StatusOK, httpserver.WriteupList{Items: items})
}

func (h *Handler) HandleCreateWriteup(c *gin.Context) {
	req, ok := bindJSON[httpserver.WriteupCreate](c)
	if !ok {
		return
	}

	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		respondError(c, errs.ErrUnauthorized)
		return
	}
	actorRole, _ := middleware.CurrentRole(c)

	writeup, err := h.writeups.Create(c.Request.Context(), actorID, actorRole, writeupsvc.CreateParams{
		GameID: req.GameId,
		TeamID: req.TeamId,
		Title:  req.Title,
		URL:    req.Url,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, writeupToHTTP(*writeup))
}

func (h *Handler) HandleGetWriteup(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	writeup, err := h.writeups.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, writeupToHTTP(*writeup))
}

func (h *Handler) HandleDeleteWriteup(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		respondError(c, errs.ErrUnauthorized)
		return
	}
	actorRole, _ := middleware.CurrentRole(c)

	if err := h.writeups.Delete(c.Request.Context(), actorID, actorRole, id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListWriteups(c *gin.Context, _ httpserver.ListWriteupsParams) {
	h.HandleListWriteups(c)
}

func (h *Handler) CreateWriteup(c *gin.Context) {
	h.HandleCreateWriteup(c)
}

func (h *Handler) GetWriteup(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetWriteup(c)
}

func (h *Handler) DeleteWriteup(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteWriteup(c)
}

func writeupToHTTP(w writeupsvc.Writeup) httpserver.Writeup {
	return httpserver.Writeup{
		Id:        w.ID,
		GameId:    w.GameID,
		TeamId:    w.TeamID,
		Title:     w.Title,
		Url:       w.URL,
		CreatedAt: &w.CreatedAt,
		UpdatedAt: &w.UpdatedAt,
	}
}
