package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	authsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/auth"
	ctf01dsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/ctf01d"
	gamesvc "github.com/ctf01d/ctf01d-training-platform/internal/service/games"
	gameteamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/gameteams"
	membersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/memberships"
	resultsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/results"
	scoreboardsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/scoreboard"
	svcsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/services"
	teamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/teams"
	unisvc "github.com/ctf01d/ctf01d-training-platform/internal/service/universities"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
	writeupsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/writeups"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
)

type Handler struct {
	users          *usersvc.Service
	auth           *authsvc.Service
	jwtMgr         *auth.Manager
	universities   *unisvc.Service
	teams          *teamsvc.Service
	memberships    *membersvc.Service
	games          *gamesvc.Service
	gameTeams      *gameteamsvc.Service
	results        *resultsvc.Service
	writeups       *writeupsvc.Service
	scoreboard     *scoreboardsvc.Service
	gameTeamsQ     *db.Queries
	svcService     *svcsvc.Service
	svcArchives    *svcsvc.ArchiveService
	svcChecker     *svcsvc.CheckerService
	svcImport      *svcsvc.ImportService
	ctf01dBuilder  *ctf01dsvc.Builder
	maxUploadBytes int64
	storageDir     string
	fileStorage    storage.Storage
}

const (
	roleGuest  = "guest"
	rolePlayer = "player"
	roleAdmin  = "admin"

	maxBytesReaderOverhead = 1024

	minInt32 = -1 << 31
	maxInt32 = 1<<31 - 1
)

func New(
	users *usersvc.Service,
	authSvc *authsvc.Service,
	jwtMgr *auth.Manager,
	universities *unisvc.Service,
	teams *teamsvc.Service,
	memberships *membersvc.Service,
	games *gamesvc.Service,
	gameTeams *gameteamsvc.Service,
	results *resultsvc.Service,
	writeups *writeupsvc.Service,
	scoreboard *scoreboardsvc.Service,
	gameTeamsQ *db.Queries,
	svcService *svcsvc.Service,
	svcArchives *svcsvc.ArchiveService,
	svcChecker *svcsvc.CheckerService,
	svcImport *svcsvc.ImportService,
	ctf01dBuilder *ctf01dsvc.Builder,
	maxUploadBytes int64,
	storageDir string,
	fileStorage storage.Storage,
) *Handler {
	return &Handler{
		users:          users,
		auth:           authSvc,
		jwtMgr:         jwtMgr,
		universities:   universities,
		teams:          teams,
		memberships:    memberships,
		games:          games,
		gameTeams:      gameTeams,
		results:        results,
		writeups:       writeups,
		scoreboard:     scoreboard,
		gameTeamsQ:     gameTeamsQ,
		svcService:     svcService,
		svcArchives:    svcArchives,
		svcChecker:     svcChecker,
		svcImport:      svcImport,
		ctf01dBuilder:  ctf01dBuilder,
		maxUploadBytes: maxUploadBytes,
		storageDir:     storageDir,
		fileStorage:    fileStorage,
	}
}

func (h *Handler) JWTMgr() *auth.Manager {
	return h.jwtMgr
}

// SessionChecker exposes the auth service for session enforcement in
// middleware. It returns nil when no auth service is configured (tests).
func (h *Handler) SessionChecker() middleware.SessionChecker {
	if h.auth == nil {
		return nil
	}
	return h.auth
}

func (h *Handler) Login(c *gin.Context) {
	req, ok := bindJSON[httpserver.LoginRequest](c)
	if !ok {
		return
	}

	token, user, err := h.auth.Login(c.Request.Context(), req.UserName, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, httpserver.LoginResponse{
		Token: token,
		User:  userToHTTPPrivate(*user),
	})
}

func (h *Handler) Logout(c *gin.Context) {
	if jti, ok := middleware.CurrentSessionJTI(c); ok {
		if err := h.auth.Logout(c.Request.Context(), jti); err != nil {
			respondError(c, err)
			return
		}
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	user, err := h.auth.Me(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, userToHTTPPrivate(*user))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	req, ok := bindJSON[httpserver.UserProfileUpdate](c)
	if !ok {
		return
	}

	params := usersvc.ProfileUpdateParams{
		DisplayName: req.DisplayName,
		Password:    req.Password,
		Bio:         req.Bio,
		Telegram:    req.Telegram,
		Github:      req.Github,
		Email:       req.Email,
		Language:    enumStringPtr(req.Language),
		Theme:       enumStringPtr(req.Theme),
	}

	user, err := h.users.UpdateProfile(c.Request.Context(), userID, params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, userToHTTPPrivate(*user))
}

func (h *Handler) ChangeProfilePassword(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	req, ok := bindJSON[httpserver.PasswordUpdate](c)
	if !ok {
		return
	}

	if _, err := h.users.ChangePassword(c.Request.Context(), userID, req.Password); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
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

	var q *string
	if v := c.Query("q"); v != "" {
		q = &v
	}

	result, err := h.users.List(c.Request.Context(), page, perPage, q)
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

	role := roleGuest
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

	userID, _ := middleware.CurrentUserID(c)
	role, _ := middleware.CurrentRole(c)
	if role != roleAdmin && userID != id {
		respondError(c, errs.ErrForbidden)
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

	req, ok := bindJSON[httpserver.UserDeleteRequest](c)
	if !ok {
		return
	}

	adminID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}
	if !h.auth.VerifyUserPassword(c.Request.Context(), adminID, req.Password) {
		c.JSON(http.StatusForbidden, errorResponse{Code: codeForbidden, Message: "admin password confirmation failed"})
		return
	}

	if err := h.users.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleUpdateUserRole(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	adminID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}
	if adminID == id {
		c.JSON(http.StatusForbidden, errorResponse{Code: codeForbidden, Message: "you cannot change your own role"})
		return
	}

	req, ok := bindJSON[httpserver.UserRoleUpdate](c)
	if !ok {
		return
	}
	if !req.Role.Valid() {
		respondError(c, errs.NewValidationError(map[string]string{"role": "must be one of guest, player, admin"}))
		return
	}

	user, err := h.users.UpdateRole(c.Request.Context(), id, string(req.Role))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, userToHTTP(*user))
}

// ServerInterface implementation (used for compile-time check)
func (h *Handler) ListUsers(c *gin.Context, _ httpserver.ListUsersParams) {
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

func (h *Handler) UpdateUserRole(c *gin.Context, _ int64) {
	h.HandleUpdateUserRole(c)
}

func (h *Handler) DeleteUser(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteUser(c)
}

func userToHTTP(u usersvc.User) httpserver.User {
	return userToHTTPWithPrivate(u, false)
}

func userToHTTPPrivate(u usersvc.User) httpserver.User {
	return userToHTTPWithPrivate(u, true)
}

func userToHTTPWithPrivate(u usersvc.User, includePrivate bool) httpserver.User {
	result := httpserver.User{
		Id:          u.ID,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		Language:    httpserver.UserLanguage(u.Language),
		Theme:       httpserver.UserTheme(u.Theme),
		Role:        httpserver.UserRole(u.Role),
		Rating:      u.Rating,
		AvatarUrl:   u.AvatarUrl,
		Bio:         u.Bio,
		Telegram:    u.Telegram,
		Github:      u.Github,
		Email:       u.Email,
		IsBlocked:   u.IsBlocked,
		CreatedAt:   &u.CreatedAt,
		UpdatedAt:   &u.UpdatedAt,
	}
	if includePrivate {
		result.LastLoginIp = u.LastLoginIp
		result.LastLoginAt = u.LastLoginAt
	}
	return result
}

func enumStringPtr[T ~string](value *T) *string {
	if value == nil {
		return nil
	}
	s := string(*value)
	return &s
}

func parseIDParam(c *gin.Context, param string) (int64, bool) {
	s := c.Param(param)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: codeBadRequest, Message: "invalid id parameter"})
		return 0, false
	}
	return id, true
}

func int32FromInt(value int) (int32, bool) {
	if value < minInt32 || value > maxInt32 {
		return 0, false
	}
	return int32(value), true
}

func (h *Handler) ListUniversities(c *gin.Context, _ httpserver.ListUniversitiesParams) {
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

func (h *Handler) ListTeams(c *gin.Context, _ httpserver.ListTeamsParams) {
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

func (h *Handler) ListTeamEvents(c *gin.Context, id int64, _ httpserver.ListTeamEventsParams) {
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

func (h *Handler) ListTeamMemberships(c *gin.Context, _ httpserver.ListTeamMembershipsParams) {
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

func (h *Handler) ListGames(c *gin.Context, _ httpserver.ListGamesParams) {
	h.HandleListGames(c)
}

func (h *Handler) CreateGame(c *gin.Context) {
	h.HandleCreateGame(c)
}

func (h *Handler) GetGame(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetGame(c)
}

func (h *Handler) UpdateGame(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateGame(c)
}

func (h *Handler) DeleteGame(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteGame(c)
}

func (h *Handler) FinalizeGame(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleFinalizeGame(c)
}

func (h *Handler) UnfinalizeGame(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUnfinalizeGame(c)
}

func (h *Handler) ListGameServices(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleListGameServices(c)
}

func (h *Handler) AddGameService(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleAddGameService(c)
}

func (h *Handler) RemoveGameService(c *gin.Context, id int64, serviceId int64) {
	c.Set("id", id)
	c.Set("service_id", serviceId)
	h.HandleRemoveGameService(c)
}

func (h *Handler) SetGameServiceStatus(c *gin.Context, id int64, serviceId int64) {
	c.Set("id", id)
	c.Set("service_id", serviceId)
	h.HandleSetGameServiceStatus(c)
}

func (h *Handler) PublishGame(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandlePublishGame(c)
}

func (h *Handler) ListGameTeams(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleListGameTeams(c)
}

func (h *Handler) ReorderGameTeams(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleReorderGameTeams(c)
}

func (h *Handler) CreateGameTeam(c *gin.Context) {
	h.HandleCreateGameTeam(c)
}

func (h *Handler) UpdateGameTeam(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateGameTeam(c)
}

func (h *Handler) DeleteGameTeam(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteGameTeam(c)
}

func (h *Handler) ListResults(c *gin.Context, _ httpserver.ListResultsParams) {
	h.HandleListResults(c)
}

func (h *Handler) CreateResult(c *gin.Context) {
	h.HandleCreateResult(c)
}

func (h *Handler) GetResult(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetResult(c)
}

func (h *Handler) UpdateResult(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateResult(c)
}

func (h *Handler) DeleteResult(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteResult(c)
}

func (h *Handler) GetGameScoreboard(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetGameScoreboard(c)
}

func (h *Handler) GetGlobalScoreboard(c *gin.Context) {
	h.HandleGetGlobalScoreboard(c)
}

func (h *Handler) HandleListServices(c *gin.Context) {
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
	var publicFilter *bool
	if v := c.Query("public"); v != "" {
		b := v == "true"
		publicFilter = &b
	}
	var q *string
	if v := c.Query("q"); v != "" {
		q = &v
	}

	role, hasRole := middleware.CurrentRole(c)
	isAdmin := hasRole && role == roleAdmin
	includeSource := hasRole && (role == roleAdmin || role == rolePlayer)

	if !isAdmin {
		b := true
		publicFilter = &b
	}

	result, err := h.svcService.List(c.Request.Context(), page, perPage, publicFilter, q, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.Service, len(result.Items))
	for i, s := range result.Items {
		items[i] = serviceToHTTP(s, includeSource)
	}

	c.JSON(http.StatusOK, httpserver.ServiceList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    result.Page,
			PerPage: result.PerPage,
			Total:   int(result.Total),
		},
	})
}

func (h *Handler) HandleCreateService(c *gin.Context) {
	req, ok := bindJSON[httpserver.ServiceCreate](c)
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	pub := true
	if req.Public != nil {
		pub = *req.Public
	}
	var training json.RawMessage
	if req.Ctf01dTraining != nil {
		b, _ := json.Marshal(*req.Ctf01dTraining)
		training = b
	}
	params := svcsvc.CreateParams{
		Name:               req.Name,
		PublicDescription:  req.PublicDescription,
		PrivateDescription: req.PrivateDescription,
		Author:             req.Author,
		Copyright:          req.Copyright,
		AvatarUrl:          req.AvatarUrl,
		Public:             pub,
		ServiceArchiveUrl:  req.ServiceArchiveUrl,
		CheckerArchiveUrl:  req.CheckerArchiveUrl,
		WriteupUrl:         req.WriteupUrl,
		ExploitsUrl:        req.ExploitsUrl,
		Ctf01dTraining:     training,
	}
	if req.GitSource != nil {
		params.GitSource = &svcsvc.GitSourceInput{}
		if req.GitSource.RepoUrl != nil {
			params.GitSource.RepoURL = *req.GitSource.RepoUrl
		}
		if req.GitSource.Ref != nil {
			params.GitSource.Ref = *req.GitSource.Ref
		}
		if req.GitSource.Subdir != nil {
			params.GitSource.Subdir = *req.GitSource.Subdir
		}
	}
	if req.Ports != nil {
		params.Ports = *req.Ports
	}
	if req.TechStack != nil {
		params.TechStack = *req.TechStack
	}
	svc, err := h.svcService.Create(c.Request.Context(), params, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, serviceToHTTP(*svc, true))
}

func (h *Handler) HandleGetService(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, hasRole := middleware.CurrentRole(c)
	isAdmin := hasRole && role == roleAdmin
	includeSource := hasRole && (role == roleAdmin || role == rolePlayer)
	svc, err := h.svcService.GetByID(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	if !svc.Public && !isAdmin {
		respondError(c, errs.ErrNotFound)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc, includeSource))
}

func (h *Handler) HandleUpdateService(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	req, ok := bindJSON[httpserver.ServiceUpdate](c)
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	var training json.RawMessage
	if req.Ctf01dTraining != nil {
		b, _ := json.Marshal(*req.Ctf01dTraining)
		training = b
	}
	params := svcsvc.UpdateParams{
		Name:               req.Name,
		PublicDescription:  req.PublicDescription,
		PrivateDescription: req.PrivateDescription,
		Author:             req.Author,
		Copyright:          req.Copyright,
		AvatarUrl:          req.AvatarUrl,
		Public:             req.Public,
		ServiceArchiveUrl:  req.ServiceArchiveUrl,
		CheckerArchiveUrl:  req.CheckerArchiveUrl,
		WriteupUrl:         req.WriteupUrl,
		ExploitsUrl:        req.ExploitsUrl,
		Ctf01dTraining:     training,
	}
	if req.GitSource != nil {
		params.GitSource = &svcsvc.GitSourceInput{}
		if req.GitSource.RepoUrl != nil {
			params.GitSource.RepoURL = *req.GitSource.RepoUrl
		}
		if req.GitSource.Ref != nil {
			params.GitSource.Ref = *req.GitSource.Ref
		}
		if req.GitSource.Subdir != nil {
			params.GitSource.Subdir = *req.GitSource.Subdir
		}
	}
	if req.Ports != nil {
		params.Ports = *req.Ports
	}
	if req.TechStack != nil {
		params.TechStack = *req.TechStack
	}
	svc, err := h.svcService.Update(c.Request.Context(), id, params, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc, true))
}

func (h *Handler) HandleDeleteService(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svcService.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleToggleServicePublic(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	svc, err := h.svcService.TogglePublic(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc, true))
}

func (h *Handler) HandleCheckServiceChecker(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	svc, err := h.svcChecker.CheckChecker(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc, true))
}

func (h *Handler) HandleRedownloadServiceArchives(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	svc, err := h.svcArchives.Redownload(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc, true))
}

func (h *Handler) HandleUploadServiceArchives(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes+maxBytesReaderOverhead)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "expected multipart form"})
		return
	}

	var serviceFile io.Reader
	if files := form.File["service_archive"]; len(files) > 0 {
		f, err := files[0].Open()
		if err != nil {
			respondError(c, err)
			return
		}
		defer f.Close()
		serviceFile = f
	}

	var checkerFile io.Reader
	if files := form.File["checker_archive"]; len(files) > 0 {
		f, err := files[0].Open()
		if err != nil {
			respondError(c, err)
			return
		}
		defer f.Close()
		checkerFile = f
	}

	svc, err := h.svcArchives.UploadArchives(c.Request.Context(), id, serviceFile, checkerFile, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc, true))
}

func (h *Handler) HandleDownloadServiceArchive(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	kind := c.Param("kind")

	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	isPlayer := role == rolePlayer

	svc, err := h.svcService.GetByID(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	if !svc.Public && !isAdmin && !isPlayer {
		c.JSON(http.StatusForbidden, errorResponse{Code: codeForbidden, Message: "service is not public"})
		return
	}

	rc, filename, err := h.svcArchives.OpenLocal(c.Request.Context(), id, kind)
	if err != nil {
		respondError(c, err)
		return
	}
	defer rc.Close()

	c.Header("Content-Type", "application/zip")
	safeName := sanitizeFilename(filename)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeName))
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, rc); err != nil {
		_ = c.Error(fmt.Errorf("copy service archive: %w", err))
	}
}

func (h *Handler) HandleSyncServiceFromGit(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	svc, err := h.svcImport.SyncFromGit(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc, true))
}

func (h *Handler) HandleImportServiceFromGit(c *gin.Context) {
	req, ok := bindJSON[httpserver.GitImportRequest](c)
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	importReq := svcsvc.GitImportRequest{
		RepoURL: req.RepoUrl,
	}
	if req.Ref != nil {
		importReq.Ref = *req.Ref
	}
	if req.Subdir != nil {
		importReq.Subdir = *req.Subdir
	}
	result, err := h.svcImport.ImportFromGit(c.Request.Context(), importReq, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, importResultToHTTP(result, true))
}

func (h *Handler) HandlePreviewServiceGitImport(c *gin.Context) {
	req, ok := bindJSON[httpserver.GitImportRequest](c)
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	importReq := svcsvc.GitImportRequest{
		RepoURL: req.RepoUrl,
	}
	if req.Ref != nil {
		importReq.Ref = *req.Ref
	}
	if req.Subdir != nil {
		importReq.Subdir = *req.Subdir
	}
	result, err := h.svcImport.PreviewFromGit(c.Request.Context(), importReq, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, importPreviewToHTTP(result))
}

func (h *Handler) HandleImportServiceFromZip(c *gin.Context) {
	file, err := c.FormFile("archive")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "archive file is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer f.Close()
	zipBytes, err := io.ReadAll(io.LimitReader(f, h.maxUploadBytes+1))
	if err != nil {
		respondError(c, err)
		return
	}
	if int64(len(zipBytes)) > h.maxUploadBytes {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "archive file too large"})
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	result, err := h.svcImport.ImportFromZipUpload(c.Request.Context(), zipBytes, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, importResultToHTTP(result, true))
}

func (h *Handler) HandlePreviewServiceZipImport(c *gin.Context) {
	file, err := c.FormFile("archive")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "archive file is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer f.Close()
	zipBytes, err := io.ReadAll(io.LimitReader(f, h.maxUploadBytes+1))
	if err != nil {
		respondError(c, err)
		return
	}
	if int64(len(zipBytes)) > h.maxUploadBytes {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "archive file too large"})
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == roleAdmin
	result, err := h.svcImport.PreviewFromZipUpload(c.Request.Context(), zipBytes, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, importPreviewToHTTP(result))
}

func (h *Handler) ListServices(c *gin.Context, _ httpserver.ListServicesParams) {
	h.HandleListServices(c)
}

func (h *Handler) CreateService(c *gin.Context) {
	h.HandleCreateService(c)
}

func (h *Handler) GetService(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetService(c)
}

func (h *Handler) UpdateService(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateService(c)
}

func (h *Handler) DeleteService(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleDeleteService(c)
}

func (h *Handler) CheckServiceChecker(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleCheckServiceChecker(c)
}

func (h *Handler) DownloadServiceArchive(c *gin.Context, id int64, kind httpserver.DownloadServiceArchiveParamsKind) {
	c.Set("id", id)
	c.Set("kind", string(kind))
	h.HandleDownloadServiceArchive(c)
}

func (h *Handler) RedownloadServiceArchives(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleRedownloadServiceArchives(c)
}

func (h *Handler) ToggleServicePublic(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleToggleServicePublic(c)
}

func (h *Handler) UploadServiceArchives(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUploadServiceArchives(c)
}

func (h *Handler) SyncServiceFromGit(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleSyncServiceFromGit(c)
}

func (h *Handler) ImportServiceFromGit(c *gin.Context) {
	h.HandleImportServiceFromGit(c)
}

func (h *Handler) PreviewServiceGitImport(c *gin.Context) {
	h.HandlePreviewServiceGitImport(c)
}

func (h *Handler) ImportServiceFromZip(c *gin.Context) {
	h.HandleImportServiceFromZip(c)
}

func (h *Handler) PreviewServiceZipImport(c *gin.Context) {
	h.HandlePreviewServiceZipImport(c)
}

func (h *Handler) GetCtf01dExportOptions(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetCtf01dExportOptions(c)
}

func (h *Handler) ExportCtf01d(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleExportCtf01d(c)
}

func sanitizeFilename(name string) string {
	r := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		b := name[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.' {
			r = append(r, b)
		} else {
			r = append(r, '_')
		}
	}
	return string(r)
}

var _ httpserver.ServerInterface = (*Handler)(nil)
