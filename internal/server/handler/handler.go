package handler

import (
	"net/http"
	"strconv"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	authsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/auth"
	membersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/memberships"
	teamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/teams"
	unisvc "github.com/ctf01d/ctf01d-training-platform/internal/service/universities"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	users        *usersvc.Service
	auth         *authsvc.Service
	jwtMgr       *auth.Manager
	universities *unisvc.Service
	teams        *teamsvc.Service
	memberships  *membersvc.Service
}

func New(users *usersvc.Service, authSvc *authsvc.Service, jwtMgr *auth.Manager, universities *unisvc.Service, teams *teamsvc.Service, memberships *membersvc.Service) *Handler {
	return &Handler{
		users:        users,
		auth:         authSvc,
		jwtMgr:       jwtMgr,
		universities: universities,
		teams:        teams,
		memberships:  memberships,
	}
}

func (h *Handler) JWTMgr() *auth.Manager {
	return h.jwtMgr
}

func (h *Handler) Login(c *gin.Context) {
	req, ok := bindJSON[httpserver.LoginRequest](c)
	if !ok {
		return
	}

	token, user, err := h.auth.Login(c.Request.Context(), req.UserName, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, httpserver.LoginResponse{
		Token: token,
		User:  userToHTTP(*user),
	})
}

func (h *Handler) Logout(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "not authenticated"})
		return
	}

	user, err := h.auth.Me(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, userToHTTP(*user))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "not authenticated"})
		return
	}

	req, ok := bindJSON[httpserver.UserUpdate](c)
	if !ok {
		return
	}

	params := usersvc.UpdateParams{
		DisplayName: req.DisplayName,
		AvatarUrl:   req.AvatarUrl,
		Password:    req.Password,
	}

	user, err := h.users.Update(c.Request.Context(), userID, params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, userToHTTP(*user))
}

func (h *Handler) HandleListUsers(c *gin.Context) {
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

	result, err := h.users.List(c.Request.Context(), page, perPage)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.User, len(result.Items))
	for i, u := range result.Items {
		items[i] = userToHTTP(u)
	}

	c.JSON(http.StatusOK, httpserver.UserList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    result.Page,
			PerPage: result.PerPage,
			Total:   int(result.Total),
		},
	})
}

func (h *Handler) HandleCreateUser(c *gin.Context) {
	req, ok := bindJSON[httpserver.UserCreate](c)
	if !ok {
		return
	}

	role := "guest"
	if req.Role != nil {
		role = string(*req.Role)
	}

	params := usersvc.CreateParams{
		UserName:    req.UserName,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		Role:        role,
		AvatarUrl:   req.AvatarUrl,
	}

	user, err := h.users.Create(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, userToHTTP(*user))
}

func (h *Handler) HandleGetUser(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, userToHTTP(*user))
}

func (h *Handler) HandleUpdateUser(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.UserUpdate](c)
	if !ok {
		return
	}

	params := usersvc.UpdateParams{
		DisplayName: req.DisplayName,
		AvatarUrl:   req.AvatarUrl,
		Password:    req.Password,
	}

	user, err := h.users.Update(c.Request.Context(), id, params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, userToHTTP(*user))
}

func (h *Handler) HandleDeleteUser(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.users.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ServerInterface implementation (used for compile-time check)
func (h *Handler) ListUsers(c *gin.Context, params httpserver.ListUsersParams) {
	h.HandleListUsers(c)
}

func (h *Handler) CreateUser(c *gin.Context) {
	h.HandleCreateUser(c)
}

func (h *Handler) GetUser(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetUser(c)
}

func (h *Handler) UpdateUser(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateUser(c)
}

func (h *Handler) DeleteUser(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteUser(c)
}

func userToHTTP(u usersvc.User) httpserver.User {
	return httpserver.User{
		Id:          u.ID,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		Role:        httpserver.UserRole(u.Role),
		Rating:      u.Rating,
		AvatarUrl:   u.AvatarUrl,
		CreatedAt:   &u.CreatedAt,
		UpdatedAt:   &u.UpdatedAt,
	}
}

func parseIDParam(c *gin.Context, param string) (int64, bool) {
	s := c.Param(param)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "bad_request", "message": "invalid id parameter"})
		return 0, false
	}
	return id, true
}

func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": "not_implemented", "message": "not implemented"})
}

func (h *Handler) ListUniversities(c *gin.Context, params httpserver.ListUniversitiesParams) {
	h.HandleListUniversities(c)
}

func (h *Handler) CreateUniversity(c *gin.Context) {
	h.HandleCreateUniversity(c)
}

func (h *Handler) GetUniversity(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetUniversity(c)
}

func (h *Handler) UpdateUniversity(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateUniversity(c)
}

func (h *Handler) DeleteUniversity(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteUniversity(c)
}

func (h *Handler) ListTeams(c *gin.Context, params httpserver.ListTeamsParams) {
	h.HandleListTeams(c)
}

func (h *Handler) CreateTeam(c *gin.Context) {
	h.HandleCreateTeam(c)
}

func (h *Handler) GetTeam(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetTeam(c)
}

func (h *Handler) UpdateTeam(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateTeam(c)
}

func (h *Handler) DeleteTeam(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteTeam(c)
}

func (h *Handler) ListTeamEvents(c *gin.Context, id int64, params httpserver.ListTeamEventsParams) {
	c.Set("id", id)
	h.HandleListTeamEvents(c)
}

func (h *Handler) InviteToTeam(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleInviteToTeam(c)
}

func (h *Handler) RequestJoinTeam(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleRequestJoinTeam(c)
}

func (h *Handler) ListTeamMembers(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleListTeamMembers(c)
}

func (h *Handler) ListTeamMemberships(c *gin.Context, params httpserver.ListTeamMembershipsParams) {
	h.HandleListTeamMemberships(c)
}

func (h *Handler) CreateTeamMembership(c *gin.Context) {
	h.HandleCreateTeamMembership(c)
}

func (h *Handler) DeleteTeamMembership(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteTeamMembership(c)
}

func (h *Handler) GetTeamMembership(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetTeamMembership(c)
}

func (h *Handler) UpdateTeamMembership(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateTeamMembership(c)
}

func (h *Handler) AcceptTeamMembership(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleAcceptTeamMembership(c)
}

func (h *Handler) ApproveTeamMembership(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleApproveTeamMembership(c)
}

func (h *Handler) DeclineTeamMembership(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeclineTeamMembership(c)
}

func (h *Handler) RejectTeamMembership(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleRejectTeamMembership(c)
}

func (h *Handler) SetTeamMembershipRole(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleSetTeamMembershipRole(c)
}

func (h *Handler) ListGames(c *gin.Context, params httpserver.ListGamesParams) {
	notImplemented(c)
}

func (h *Handler) CreateGame(c *gin.Context) {
	notImplemented(c)
}

func (h *Handler) GetGame(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) UpdateGame(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) DeleteGame(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) FinalizeGame(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) UnfinalizeGame(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) ListGameServices(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) AddGameService(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) RemoveGameService(c *gin.Context, id int64, serviceId int64) {
	notImplemented(c)
}

func (h *Handler) ListGameTeams(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) ReorderGameTeams(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) CreateGameTeam(c *gin.Context) {
	notImplemented(c)
}

func (h *Handler) UpdateGameTeam(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) DeleteGameTeam(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) ListResults(c *gin.Context, params httpserver.ListResultsParams) {
	notImplemented(c)
}

func (h *Handler) CreateResult(c *gin.Context) {
	notImplemented(c)
}

func (h *Handler) GetResult(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) UpdateResult(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) DeleteResult(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) GetGameScoreboard(c *gin.Context, id int64) {
	notImplemented(c)
}

func (h *Handler) GetGlobalScoreboard(c *gin.Context) {
	notImplemented(c)
}

var _ httpserver.ServerInterface = (*Handler)(nil)
