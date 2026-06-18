package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	teamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/teams"
)

func (h *Handler) HandleListTeams(c *gin.Context) {
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

	result, err := h.teams.List(c.Request.Context(), page, perPage, q)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.Team, len(result.Items))
	for i, t := range result.Items {
		items[i] = teamToHTTP(t)
	}

	c.JSON(http.StatusOK, httpserver.TeamList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    result.Page,
			PerPage: result.PerPage,
			Total:   int(result.Total),
		},
	})
}

func (h *Handler) HandleCreateTeam(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	req, ok := bindJSON[httpserver.TeamCreate](c)
	if !ok {
		return
	}

	params := teamsvc.CreateParams{
		Name:         req.Name,
		Description:  req.Description,
		Website:      req.Website,
		AvatarUrl:    req.AvatarUrl,
		UniversityID: req.UniversityId,
	}

	team, err := h.teams.Create(c.Request.Context(), userID, params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, teamToHTTP(*team))
}

func (h *Handler) HandleGetTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	team, err := h.teams.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, teamToHTTP(*team))
}

func (h *Handler) HandleUpdateTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}
	role, _ := middleware.CurrentRole(c)

	if err := h.teams.CanManage(c.Request.Context(), id, userID, role); err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[httpserver.TeamUpdate](c)
	if !ok {
		return
	}

	var name *string
	if req.Name != nil {
		name = req.Name
	}

	params := teamsvc.UpdateParams{
		Name:         name,
		Description:  req.Description,
		Website:      req.Website,
		AvatarUrl:    req.AvatarUrl,
		UniversityID: req.UniversityId,
	}

	team, err := h.teams.Update(c.Request.Context(), id, params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, teamToHTTP(*team))
}

func (h *Handler) HandleDeleteTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}
	role, _ := middleware.CurrentRole(c)

	if err := h.teams.CanManage(c.Request.Context(), id, userID, role); err != nil {
		respondError(c, err)
		return
	}

	if err := h.teams.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleRequestJoinTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	if err := h.teams.RequestJoin(c.Request.Context(), id, userID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleInviteToTeam(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	role, _ := middleware.CurrentRole(c)

	req, ok := bindJSON[httpserver.InviteRequest](c)
	if !ok {
		return
	}

	if err := h.teams.Invite(c.Request.Context(), id, userID, req.UserId, role); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleListTeamMembers(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	members, err := h.memberships.ListByTeam(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	// Unauthenticated viewers only see the approved roster; pending/rejected
	// join requests must not leak to the public team page.
	_, hasUser := middleware.CurrentUserID(c)

	items := make([]httpserver.TeamMembership, 0, len(members))
	for _, m := range members {
		if !hasUser && (m.Status == nil || *m.Status != memStatusApproved) {
			continue
		}
		items = append(items, membershipToHTTP(m))
	}

	c.JSON(http.StatusOK, httpserver.TeamMembershipList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    1,
			PerPage: len(items),
			Total:   len(items),
		},
	})
}

func (h *Handler) HandleListTeamEvents(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

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

	result, err := h.memberships.ListEvents(c.Request.Context(), id, page, perPage)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.TeamMembershipEvent, len(result.Items))
	for i, e := range result.Items {
		items[i] = eventToHTTP(e)
	}

	c.JSON(http.StatusOK, httpserver.TeamMembershipEventList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    result.Page,
			PerPage: result.PerPage,
			Total:   int(result.Total),
		},
	})
}

func teamToHTTP(t teamsvc.Team) httpserver.Team {
	var captainID *int64
	if t.CaptainID != nil {
		v := int64(*t.CaptainID)
		captainID = &v
	}
	return httpserver.Team{
		Id:           t.ID,
		Name:         t.Name,
		Description:  t.Description,
		Website:      t.Website,
		AvatarUrl:    t.AvatarUrl,
		CaptainId:    captainID,
		UniversityId: t.UniversityID,
		CreatedAt:    &t.CreatedAt,
		UpdatedAt:    &t.UpdatedAt,
	}
}
