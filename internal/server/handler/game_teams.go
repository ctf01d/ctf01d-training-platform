package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	gameteamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/gameteams"
)

func (h *Handler) HandleListGameTeams(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	items, err := h.gameTeams.ListByGame(c.Request.Context(), gameID)
	if err != nil {
		respondError(c, err)
		return
	}

	result := make([]httpserver.GameTeam, len(items))
	for i, gt := range items {
		result[i] = gameTeamToHTTP(gt)
	}

	c.JSON(http.StatusOK, httpserver.GameTeamList{Items: result})
}

func (h *Handler) HandleReorderGameTeams(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.ReorderRequest](c)
	if !ok {
		return
	}

	items := make([]gameteamsvc.ReorderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = gameteamsvc.ReorderItem{ID: item.Id, Order: item.Order}
	}

	if err := h.gameTeams.Reorder(c.Request.Context(), gameID, items); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleCreateGameTeam(c *gin.Context) {
	req, ok := bindJSON[httpserver.GameTeamCreate](c)
	if !ok {
		return
	}

	order := int32(0)
	if req.Order != nil {
		v, ok := int32FromInt(*req.Order)
		if !ok {
			respondError(c, errs.NewValidationError(map[string]string{"order": "must fit int32"}))
			return
		}
		order = v
	}

	var overrides json.RawMessage
	if req.Ctf01dOverrides != nil {
		b, _ := json.Marshal(req.Ctf01dOverrides)
		overrides = b
	} else {
		overrides = json.RawMessage("{}")
	}

	params := gameteamsvc.CreateParams{
		GameID:          req.GameId,
		TeamID:          req.TeamId,
		IpAddress:       req.IpAddress,
		Ctf01dID:        req.Ctf01dId,
		Ctf01dOverrides: overrides,
		TeamType:        req.TeamType,
		Order:           order,
	}

	gt, err := h.gameTeams.Create(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gameTeamToHTTP(*gt))
}

func (h *Handler) HandleUpdateGameTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.GameTeamUpdate](c)
	if !ok {
		return
	}

	var order *int32
	if req.Order != nil {
		v, ok := int32FromInt(*req.Order)
		if !ok {
			respondError(c, errs.NewValidationError(map[string]string{"order": "must fit int32"}))
			return
		}
		order = &v
	}

	var overrides *json.RawMessage
	if req.Ctf01dOverrides != nil {
		b, _ := json.Marshal(req.Ctf01dOverrides)
		raw := json.RawMessage(b)
		overrides = &raw
	}

	params := gameteamsvc.UpdateParams{
		IpAddress:       req.IpAddress,
		Ctf01dID:        req.Ctf01dId,
		Ctf01dOverrides: overrides,
		TeamType:        req.TeamType,
		Order:           order,
	}

	gt, err := h.gameTeams.Update(c.Request.Context(), id, params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gameTeamToHTTP(*gt))
}

func (h *Handler) HandleDeleteGameTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.gameTeams.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func gameTeamToHTTP(gt gameteamsvc.GameTeam) httpserver.GameTeam {
	var overrides *map[string]interface{}
	if gt.Ctf01dOverrides != nil {
		var m map[string]interface{}
		if err := json.Unmarshal(gt.Ctf01dOverrides, &m); err != nil {
			slog.Warn("invalid ctf01d_overrides JSON in game_team", "id", gt.ID, "error", err)
		} else {
			overrides = &m
		}
	}
	return httpserver.GameTeam{
		Id:              gt.ID,
		GameId:          gt.GameID,
		TeamId:          gt.TeamID,
		IpAddress:       gt.IpAddress,
		Ctf01dId:        gt.Ctf01dID,
		Ctf01dOverrides: overrides,
		TeamType:        gt.TeamType,
		Order:           int(gt.Order),
		CreatedAt:       &gt.CreatedAt,
		UpdatedAt:       &gt.UpdatedAt,
	}
}
