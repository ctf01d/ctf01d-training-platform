package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	authsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/auth"
	gameteamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/gameteams"
	gamesvc "github.com/ctf01d/ctf01d-training-platform/internal/service/games"
	membersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/memberships"
	resultsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/results"
	scoreboardsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/scoreboard"
	teamsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/teams"
	unisvc "github.com/ctf01d/ctf01d-training-platform/internal/service/universities"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/server"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/handler"
	"github.com/ctf01d/ctf01d-training-platform/internal/testutil"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func setupTest(t *testing.T) (*gin.Engine, *repository.Store, func()) {
	t.Helper()

	store := testutil.NewTestStore(t)
	testutil.TruncateAll(t, store)

	log, _ := zap.NewDevelopment()
	cfg := &config.Config{
		Env: "development",
		CORS: config.CORSConfig{
			AllowedOrigins: "http://localhost:5173",
		},
	}

	jwtMgr := auth.NewManager("test-integration-secret", 24)
	userService := usersvc.NewService(store.Queries)
	authService := authsvc.NewService(store.Queries, jwtMgr, &auth.PasswordCheckerImpl{})
	universityService := unisvc.NewService(store.Queries)
	teamService := teamsvc.NewService(store.Queries, store.Queries, store.Queries, store)
	membershipService := membersvc.NewService(store.Queries, store.Queries, store.Queries, store)
	gameService := gamesvc.NewService(store.Queries, store.Queries, store.Queries, store.Queries, store)
	gameTeamService := gameteamsvc.NewService(store.Queries, store)
	resultService := resultsvc.NewService(store.Queries, store.Queries)
	scoreboardService := scoreboardsvc.NewService(store.Queries, store.Queries, store.Queries, store.Queries)
	h := handler.New(userService, authService, jwtMgr, universityService, teamService, membershipService, gameService, gameTeamService, resultService, scoreboardService, store.Queries)

	engine := server.New(cfg, log, store, h)
	return engine, store, func() {}
}

func seedUser(t *testing.T, store *repository.Store, userName, displayName, password, role string) (int64, string) {
	t.Helper()
	ctx := context.Background()
	userService := usersvc.NewService(store.Queries)
	user, err := userService.Create(ctx, usersvc.CreateParams{
		UserName:    userName,
		DisplayName: displayName,
		Password:    password,
		Role:        role,
	})
	if err != nil {
		t.Fatalf("seed user %s: %v", userName, err)
	}
	jwtMgr := auth.NewManager("test-integration-secret", 24)
	token, err := jwtMgr.Generate(user.ID, user.Role, user.UserName)
	if err != nil {
		t.Fatalf("generate token for %s: %v", userName, err)
	}
	return user.ID, token
}

func makeReq(t *testing.T, engine *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshaling body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("parsing JSON: %v, body: %s", err, w.Body.String())
	}
	return result
}

func parseJSONArray(t *testing.T, w *httptest.ResponseRecorder) []map[string]interface{} {
	t.Helper()
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("parsing JSON array: %v, body: %s", err, w.Body.String())
	}
	return result
}

func TestAuthFlow(t *testing.T) {
	engine, store, _ := setupTest(t)

	_, adminToken := seedUser(t, store, "admin", "Admin", "admin12345", "admin")

	w := makeReq(t, engine, http.MethodPost, "/api/v1/session", map[string]interface{}{
		"user_name": "admin", "password": "admin12345",
	}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	loginToken := parseJSON(t, w)["token"].(string)
	_ = adminToken

	w = makeReq(t, engine, http.MethodPost, "/api/v1/users", map[string]interface{}{
		"user_name": "player1", "display_name": "Player One", "password": "password123", "role": "player",
	}, loginToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create player: %d %s", w.Code, w.Body.String())
	}
	player := parseJSON(t, w)
	playerID := int64(player["id"].(float64))

	w = makeReq(t, engine, http.MethodGet, "/api/v1/users", nil, loginToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list users: %d %s", w.Code, w.Body.String())
	}

	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", playerID), nil, loginToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get user: %d %s", w.Code, w.Body.String())
	}

	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/users/%d", playerID), map[string]interface{}{
		"display_name": "Updated Name",
	}, loginToken)
	if w.Code != http.StatusOK {
		t.Fatalf("update user: %d %s", w.Code, w.Body.String())
	}

	w = makeReq(t, engine, http.MethodPost, "/api/v1/session", map[string]interface{}{
		"user_name": "player1", "password": "password123",
	}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("login player: %d %s", w.Code, w.Body.String())
	}
	playerToken := parseJSON(t, w)["token"].(string)

	w = makeReq(t, engine, http.MethodGet, "/api/v1/profile", nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get profile: %d %s", w.Code, w.Body.String())
	}
	profile := parseJSON(t, w)
	if profile["user_name"] != "player1" {
		t.Errorf("expected player1, got %v", profile["user_name"])
	}

	w = makeReq(t, engine, http.MethodPost, "/api/v1/session", map[string]interface{}{
		"user_name": "player1", "password": "wrongpassword",
	}, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}

	w = makeReq(t, engine, http.MethodGet, "/api/v1/users", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}

	w = makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", playerID), nil, loginToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete user: %d %s", w.Code, w.Body.String())
	}
}
