package handler

import (
	"net/http"
	"strconv"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	resultsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/results"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func parseIDQuery(c *gin.Context, key string) (int64, bool) {
	s := c.Query(key)
	if s == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		respondError(c, errs.NewValidationError(map[string]string{key: "must be a valid integer"}))
		return 0, false
	}
	return id, true
}

func (h *Handler) HandleListResults(c *gin.Context) {
	gameIDStr := c.Query("game_id")
	teamIDStr := c.Query("team_id")

	var results []resultsvc.Result
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
		results, err = h.results.ListByGameAndTeam(c.Request.Context(), gameID, teamID)
	case gameIDStr != "":
		gameID, ok := parseIDQuery(c, "game_id")
		if !ok {
			return
		}
		results, err = h.results.ListByGame(c.Request.Context(), gameID)
	case teamIDStr != "":
		teamID, ok := parseIDQuery(c, "team_id")
		if !ok {
			return
		}
		results, err = h.results.ListByTeam(c.Request.Context(), teamID)
	default:
		results, err = h.results.ListAll(c.Request.Context())
	}

	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.Result, len(results))
	for i, r := range results {
		items[i] = resultToHTTP(r)
	}

	c.JSON(http.StatusOK, httpserver.ResultList{Items: items})
}

func (h *Handler) HandleCreateResult(c *gin.Context) {
	req, ok := bindJSON[httpserver.ResultCreate](c)
	if !ok {
		return
	}

	role, _ := middleware.CurrentRole(c)

	params := resultsvc.CreateParams{
		GameID: req.GameId,
		TeamID: req.TeamId,
		Score:  int32PtrFromIntPtr(req.Score),
	}

	result, err := h.results.Create(c.Request.Context(), params, role)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resultToHTTP(*result))
}

func (h *Handler) HandleGetResult(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	result, err := h.results.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resultToHTTP(*result))
}

func (h *Handler) HandleUpdateResult(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.ResultUpdate](c)
	if !ok {
		return
	}

	role, _ := middleware.CurrentRole(c)

	params := resultsvc.UpdateParams{
		Score: int32PtrFromIntPtr(req.Score),
	}

	result, err := h.results.Update(c.Request.Context(), id, params, role)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resultToHTTP(*result))
}

func (h *Handler) HandleDeleteResult(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	role, _ := middleware.CurrentRole(c)

	if err := h.results.Delete(c.Request.Context(), id, role); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func resultToHTTP(r resultsvc.Result) httpserver.Result {
	return httpserver.Result{
		Id:        r.ID,
		GameId:    r.GameID,
		TeamId:    r.TeamID,
		Score:     intPtrFromInt32Ptr(r.Score),
		CreatedAt: &r.CreatedAt,
		UpdatedAt: &r.UpdatedAt,
	}
}

func int32PtrFromIntPtr(v *int) *int32 {
	if v == nil {
		return nil
	}
	r := int32(*v)
	return &r
}

func intPtrFromInt32Ptr(v *int32) *int {
	if v == nil {
		return nil
	}
	r := int(*v)
	return &r
}
