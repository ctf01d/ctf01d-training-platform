package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
)

func (h *Handler) HandleGetGameScoreboard(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	viewerRole, _ := middleware.CurrentRole(c)

	sb, err := h.scoreboard.ForGame(c.Request.Context(), gameID, viewerRole)
	if err != nil {
		respondError(c, err)
		return
	}

	entries := make([]httpserver.ScoreboardEntry, len(sb.Entries))
	for i, e := range sb.Entries {
		entries[i] = httpserver.ScoreboardEntry{
			TeamId:   e.TeamID,
			TeamName: e.TeamName,
			Score:    e.Score,
			Position: e.Position,
		}
	}

	c.JSON(http.StatusOK, httpserver.Scoreboard{
		GameId:  sb.GameID,
		Status:  httpserver.ScoreboardStatus(sb.Status),
		Entries: entries,
	})
}

func (h *Handler) HandleGetGlobalScoreboard(c *gin.Context) {
	sb, err := h.scoreboard.Global(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	entries := make([]struct {
		TeamId     int64  `json:"team_id"`
		TeamName   string `json:"team_name"`
		TotalScore int    `json:"total_score"`
	}, len(sb.Entries))
	for i, e := range sb.Entries {
		entries[i] = struct {
			TeamId     int64  `json:"team_id"`
			TeamName   string `json:"team_name"`
			TotalScore int    `json:"total_score"`
		}{
			TeamId:     e.TeamID,
			TeamName:   e.TeamName,
			TotalScore: e.TotalScore,
		}
	}

	c.JSON(http.StatusOK, httpserver.GlobalScoreboard{Entries: entries})
}
