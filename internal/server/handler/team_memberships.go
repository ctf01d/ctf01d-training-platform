package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	membersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/memberships"
)

func (h *Handler) HandleListTeamMemberships(c *gin.Context) {
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

	result, err := h.memberships.List(c.Request.Context(), page, perPage)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.TeamMembership, len(result.Items))
	for i, m := range result.Items {
		items[i] = membershipToHTTP(m)
	}

	c.JSON(http.StatusOK, httpserver.TeamMembershipList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    result.Page,
			PerPage: result.PerPage,
			Total:   int(result.Total),
		},
	})
}

func (h *Handler) HandleCreateTeamMembership(c *gin.Context) {
	req, ok := bindJSON[httpserver.TeamMembershipCreate](c)
	if !ok {
		return
	}

	role := roleGuest
	if req.Role != nil {
		role = string(*req.Role)
	}
	status := "pending"
	if req.Status != nil {
		status = string(*req.Status)
	}

	mem, err := h.memberships.CreateDirect(c.Request.Context(), db.CreateTeamMembershipParams{
		TeamID: req.TeamId,
		UserID: req.UserId,
		Role:   &role,
		Status: &status,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, membershipToHTTP(*mem))
}

func (h *Handler) HandleGetTeamMembership(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	mem, err := h.memberships.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, membershipToHTTP(*mem))
}

func (h *Handler) HandleUpdateTeamMembership(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.TeamMembershipUpdate](c)
	if !ok {
		return
	}

	role := ""
	if req.Role != nil {
		role = string(*req.Role)
	}
	status := ""
	if req.Status != nil {
		status = string(*req.Status)
	}

	mem, err := h.memberships.Update(c.Request.Context(), id, role, status)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, membershipToHTTP(*mem))
}

func (h *Handler) HandleDeleteTeamMembership(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.memberships.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleApproveTeamMembership(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}
	role, _ := middleware.CurrentRole(c)

	if err := h.memberships.Approve(c.Request.Context(), id, actorID, role); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleRejectTeamMembership(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}
	role, _ := middleware.CurrentRole(c)

	if err := h.memberships.Reject(c.Request.Context(), id, actorID, role); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleAcceptTeamMembership(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	if err := h.memberships.Accept(c.Request.Context(), id, userID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleDeclineTeamMembership(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	if err := h.memberships.Decline(c.Request.Context(), id, userID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleSetTeamMembershipRole(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	actorID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}
	role, _ := middleware.CurrentRole(c)

	req, ok := bindJSON[httpserver.SetRoleRequest](c)
	if !ok {
		return
	}

	if err := h.memberships.SetRole(c.Request.Context(), id, string(req.Role), actorID, role); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func membershipToHTTP(m membersvc.Membership) httpserver.TeamMembership {
	role := roleGuest
	if m.Role != nil {
		role = *m.Role
	}
	status := "pending"
	if m.Status != nil {
		status = *m.Status
	}
	return httpserver.TeamMembership{
		Id:        m.ID,
		TeamId:    m.TeamID,
		UserId:    m.UserID,
		Role:      httpserver.TeamMembershipRole(role),
		Status:    httpserver.TeamMembershipStatus(status),
		CreatedAt: &m.CreatedAt,
		UpdatedAt: &m.UpdatedAt,
	}
}

func eventToHTTP(e membersvc.MembershipEvent) httpserver.TeamMembershipEvent {
	var actorID *int64
	if e.ActorID != nil {
		v := int64(*e.ActorID)
		actorID = &v
	}
	return httpserver.TeamMembershipEvent{
		Id:         e.ID,
		TeamId:     e.TeamID,
		UserId:     e.UserID,
		ActorId:    actorID,
		Action:     e.Action,
		FromRole:   e.FromRole,
		ToRole:     e.ToRole,
		FromStatus: e.FromStatus,
		ToStatus:   e.ToStatus,
		CreatedAt:  &e.CreatedAt,
		UpdatedAt:  &e.UpdatedAt,
	}
}
