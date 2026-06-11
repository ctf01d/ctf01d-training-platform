package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	authsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/auth"
	gameteamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/gameteams"
	gamesvc "github.com/ctf01d/ctf01d-training-platform/internal/service/games"
	membersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/memberships"
	resultsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/results"
	scoreboardsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/scoreboard"
	ctf01dsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/ctf01d"
	svcsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/services"
	teamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/teams"
	unisvc "github.com/ctf01d/ctf01d-training-platform/internal/service/universities"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	users         *usersvc.Service
	auth          *authsvc.Service
	jwtMgr        *auth.Manager
	universities  *unisvc.Service
	teams         *teamsvc.Service
	memberships   *membersvc.Service
	games         *gamesvc.Service
	gameTeams     *gameteamsvc.Service
	results       *resultsvc.Service
	scoreboard    *scoreboardsvc.Service
	gameTeamsQ    *db.Queries
	svcService    *svcsvc.Service
	svcArchives   *svcsvc.ArchiveService
	svcChecker    *svcsvc.CheckerService
	svcImport     *svcsvc.ImportService
	ctf01dBuilder *ctf01dsvc.Builder
}

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
	scoreboard *scoreboardsvc.Service,
	gameTeamsQ *db.Queries,
	svcService *svcsvc.Service,
	svcArchives *svcsvc.ArchiveService,
	svcChecker *svcsvc.CheckerService,
	svcImport *svcsvc.ImportService,
	ctf01dBuilder *ctf01dsvc.Builder,
) *Handler {
	return &Handler{
		users:         users,
		auth:          authSvc,
		jwtMgr:        jwtMgr,
		universities:  universities,
		teams:         teams,
		memberships:   memberships,
		games:         games,
		gameTeams:     gameTeams,
		results:       results,
		scoreboard:    scoreboard,
		gameTeamsQ:    gameTeamsQ,
		svcService:    svcService,
		svcArchives:   svcArchives,
		svcChecker:    svcChecker,
		svcImport:     svcImport,
		ctf01dBuilder: ctf01dBuilder,
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

func (h *Handler) ListResults(c *gin.Context, params httpserver.ListResultsParams) {
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
	isAdmin := hasRole && role == "admin"

	result, err := h.svcService.List(c.Request.Context(), page, perPage, publicFilter, q, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.Service, len(result.Items))
	for i, s := range result.Items {
		items[i] = serviceToHTTP(s)
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
	isAdmin := role == "admin"
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
	svc, err := h.svcService.Create(c.Request.Context(), params, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, serviceToHTTP(*svc))
}

func (h *Handler) HandleGetService(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, hasRole := middleware.CurrentRole(c)
	isAdmin := hasRole && role == "admin"
	svc, err := h.svcService.GetByID(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	if !svc.Public && !isAdmin {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "message": "service not found"})
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc))
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
	isAdmin := role == "admin"
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
	svc, err := h.svcService.Update(c.Request.Context(), id, params, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc))
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
	isAdmin := role == "admin"
	svc, err := h.svcService.TogglePublic(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc))
}

func (h *Handler) HandleCheckServiceChecker(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == "admin"
	svc, err := h.svcChecker.CheckChecker(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc))
}

func (h *Handler) HandleRedownloadServiceArchives(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == "admin"
	svc, err := h.svcArchives.Redownload(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, serviceToHTTP(*svc))
}

func (h *Handler) HandleUploadServiceArchives(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == "admin"

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"code": "validation_error", "message": "expected multipart form"})
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
	c.JSON(http.StatusOK, serviceToHTTP(*svc))
}

func (h *Handler) HandleDownloadServiceArchive(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	kind := c.Param("kind")

	role, _ := middleware.CurrentRole(c)
	isAdmin := role == "admin"

	svc, err := h.svcService.GetByID(c.Request.Context(), id, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	if !svc.Public && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": "forbidden", "message": "service is not public"})
		return
	}

	rc, filename, err := h.svcArchives.OpenLocal(c.Request.Context(), id, kind)
	if err != nil {
		respondError(c, err)
		return
	}
	defer rc.Close()

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, rc)
}

func (h *Handler) HandleImportServiceFromGithub(c *gin.Context) {
	req, ok := bindJSON[httpserver.GithubImportRequest](c)
	if !ok {
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == "admin"
	importReq := svcsvc.GithubImportRequest{
		RepoURL: req.RepoUrl,
	}
	if req.Ref != nil {
		importReq.Ref = *req.Ref
	}
	if req.Subdir != nil {
		importReq.Subdir = *req.Subdir
	}
	result, err := h.svcImport.ImportFromGithub(c.Request.Context(), importReq, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, importResultToHTTP(result))
}

func (h *Handler) HandleImportServiceFromZip(c *gin.Context) {
	file, err := c.FormFile("archive")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"code": "validation_error", "message": "archive file is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer f.Close()
	zipBytes, err := io.ReadAll(f)
	if err != nil {
		respondError(c, err)
		return
	}
	role, _ := middleware.CurrentRole(c)
	isAdmin := role == "admin"
	result, err := h.svcImport.ImportFromZipUpload(c.Request.Context(), zipBytes, isAdmin)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, importResultToHTTP(result))
}

func (h *Handler) ListServices(c *gin.Context, params httpserver.ListServicesParams) {
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

func (h *Handler) ImportServiceFromGithub(c *gin.Context) {
	h.HandleImportServiceFromGithub(c)
}

func (h *Handler) ImportServiceFromZip(c *gin.Context) {
	h.HandleImportServiceFromZip(c)
}

func (h *Handler) GetCtf01dExportOptions(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetCtf01dExportOptions(c)
}

func (h *Handler) ExportCtf01d(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleExportCtf01d(c)
}

var _ httpserver.ServerInterface = (*Handler)(nil)
