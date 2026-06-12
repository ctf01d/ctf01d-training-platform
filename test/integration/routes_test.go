package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRemainingHTTPRoutesFlow(t *testing.T) {
	engine, store, _ := setupTest(t)

	_, adminToken := seedUser(t, store, "admin", "Admin", "admin12345", "admin")
	_, ownerToken := seedUser(t, store, "owner", "Owner", "password123", "player")
	directID, _ := seedUser(t, store, "direct_member", "Direct Member", "password123", "player")
	rejectID, _ := seedUser(t, store, "reject_member", "Reject Member", "password123", "player")
	declineID, declineToken := seedUser(t, store, "decline_member", "Decline Member", "password123", "player")

	t.Log("Step: health and version")
	requireStatus(t, makeReq(t, engine, http.MethodGet, "/healthz", nil, ""), http.StatusOK, "healthz")
	requireStatus(t, makeReq(t, engine, http.MethodGet, "/version", nil, ""), http.StatusOK, "version")

	t.Log("Step: logout and profile update")
	requireStatus(t, makeReq(t, engine, http.MethodDelete, "/api/v1/session", nil, ownerToken), http.StatusNoContent, "logout")
	w := makeReq(t, engine, http.MethodPatch, "/api/v1/profile", map[string]interface{}{
		"display_name": "Owner Updated",
	}, ownerToken)
	requireStatus(t, w, http.StatusOK, "update profile")

	t.Log("Step: update user role")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/users", map[string]interface{}{
		"user_name": "role_user", "display_name": "Role User", "password": "password123", "role": "guest",
	}, adminToken)
	requireStatus(t, w, http.StatusCreated, "create user for role update")
	roleUserID := jsonID(t, parseJSON(t, w))
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/users/%d/role", roleUserID), map[string]interface{}{
		"role": "player",
	}, adminToken)
	requireStatus(t, w, http.StatusOK, "update user role")

	t.Log("Step: create team for membership routes")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/teams", map[string]interface{}{
		"name": "Route Coverage Team",
	}, ownerToken)
	requireStatus(t, w, http.StatusCreated, "create team")
	teamID := jsonID(t, parseJSON(t, w))

	t.Log("Step: direct team-memberships CRUD")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/team-memberships", map[string]interface{}{
		"team_id": teamID,
		"user_id": directID,
		"role":    "guest",
		"status":  "pending",
	}, adminToken)
	requireStatus(t, w, http.StatusCreated, "create team membership")
	membershipID := jsonID(t, parseJSON(t, w))
	requireStatus(t, makeReq(t, engine, http.MethodGet, "/api/v1/team-memberships", nil, ownerToken), http.StatusOK, "list team memberships")
	requireStatus(t, makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/team-memberships/%d", membershipID), nil, ownerToken), http.StatusOK, "get team membership")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/team-memberships/%d", membershipID), map[string]interface{}{
		"role": "player",
	}, adminToken)
	requireStatus(t, w, http.StatusOK, "update team membership")
	requireStatus(t, makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/team-memberships/%d", membershipID), nil, adminToken), http.StatusNoContent, "delete team membership")

	t.Log("Step: reject and decline membership actions")
	requireStatus(t, makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/teams/%d/invite", teamID), map[string]interface{}{
		"user_id": rejectID,
	}, ownerToken), http.StatusNoContent, "invite reject user")
	rejectMembershipID := membershipIDForUser(t, engine, teamID, rejectID, ownerToken)
	requireStatus(t, makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/team-memberships/%d/reject", rejectMembershipID), nil, ownerToken), http.StatusNoContent, "reject membership")

	requireStatus(t, makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/teams/%d/invite", teamID), map[string]interface{}{
		"user_id": declineID,
	}, ownerToken), http.StatusNoContent, "invite decline user")
	declineMembershipID := membershipIDForUser(t, engine, teamID, declineID, ownerToken)
	requireStatus(t, makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/team-memberships/%d/decline", declineMembershipID), nil, declineToken), http.StatusNoContent, "decline membership")

	t.Log("Step: delete team with memberships/events")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/teams", map[string]interface{}{
		"name": "Temporary Team",
	}, ownerToken)
	requireStatus(t, w, http.StatusCreated, "create temporary team")
	tempTeamID := jsonID(t, parseJSON(t, w))
	requireStatus(t, makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/teams/%d", tempTeamID), nil, ownerToken), http.StatusNoContent, "delete temporary team")
	requireStatus(t, makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d", tempTeamID), nil, ownerToken), http.StatusNotFound, "get deleted team")

	t.Log("Step: create game and service for game-service/export/writeup routes")
	startsAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	endsAt := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	w = makeReq(t, engine, http.MethodPost, "/api/v1/games", map[string]interface{}{
		"name":      "Route Coverage Game",
		"starts_at": startsAt,
		"ends_at":   endsAt,
	}, ownerToken)
	requireStatus(t, w, http.StatusCreated, "create game")
	gameID := jsonID(t, parseJSON(t, w))

	w = makeReq(t, engine, http.MethodPost, "/api/v1/services", map[string]interface{}{
		"name":               "route-coverage-service",
		"public_description": "route coverage",
		"public":             true,
	}, ownerToken)
	requireStatus(t, w, http.StatusCreated, "create service")
	serviceID := jsonID(t, parseJSON(t, w))

	t.Log("Step: service actions redownload, checker upload/download, github validation")
	requireStatus(t, makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/services/%d/redownload", serviceID), nil, ownerToken), http.StatusOK, "redownload service with no URLs")
	checkerZip := createTestZip(t, map[string]string{"checker.py": "print(101)\n"})
	requireStatus(t, makeMultipartUpload(t, engine, fmt.Sprintf("/api/v1/services/%d/upload-archives", serviceID), checkerZip, "checker_archive", "checker.zip", ownerToken), http.StatusOK, "upload checker archive")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/services/%d/download/checker", serviceID), nil, ownerToken)
	requireStatus(t, w, http.StatusOK, "download checker archive")
	if w.Body.Len() == 0 {
		t.Fatal("download checker archive: empty body")
	}
	requireStatus(t, makeReq(t, engine, http.MethodPost, "/api/v1/services/import/github", map[string]interface{}{
		"repo_url": "https://github.com/example/repo",
		"subdir":   "service",
	}, ownerToken), http.StatusUnprocessableEntity, "github import unsupported subdir")

	t.Log("Step: game-services routes")
	requireStatus(t, makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/services", gameID), map[string]interface{}{
		"service_id": serviceID,
	}, ownerToken), http.StatusOK, "add game service")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d/services", gameID), nil, ownerToken)
	requireStatus(t, w, http.StatusOK, "list game services")
	serviceIDs := parseNumberArray(t, w)
	if len(serviceIDs) != 1 || serviceIDs[0] != serviceID {
		t.Fatalf("list game services: got %v, want [%d]", serviceIDs, serviceID)
	}
	requireStatus(t, makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/games/%d/services/%d", gameID, serviceID), nil, ownerToken), http.StatusNoContent, "remove game service")

	t.Log("Step: ctf01d export routes")
	requireStatus(t, makeReq(t, engine, http.MethodPost, "/api/v1/game-teams", map[string]interface{}{
		"game_id":    gameID,
		"team_id":    teamID,
		"ip_address": "10.10.0.1",
	}, ownerToken), http.StatusCreated, "add game team for export")
	requireStatus(t, makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/services", gameID), map[string]interface{}{
		"service_id": serviceID,
	}, ownerToken), http.StatusOK, "add game service for export")
	requireStatus(t, makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d/export/ctf01d/options", gameID), nil, ownerToken), http.StatusOK, "get ctf01d export options")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/export/ctf01d", gameID), map[string]interface{}{
		"include_html":    false,
		"include_compose": false,
		"prefix":          "ctf01dtest",
		"port":            8080,
	}, ownerToken)
	requireStatus(t, w, http.StatusOK, "export ctf01d")
	if w.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("export ctf01d: content-type = %q, want application/zip", w.Header().Get("Content-Type"))
	}
	if w.Body.Len() == 0 {
		t.Fatal("export ctf01d: empty zip body")
	}

	t.Log("Step: writeups routes")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/writeups", map[string]interface{}{
		"game_id": gameID,
		"team_id": teamID,
		"title":   "Route Coverage Writeup",
		"url":     "https://example.com/writeup",
	}, ownerToken)
	requireStatus(t, w, http.StatusCreated, "create writeup")
	writeupID := jsonID(t, parseJSON(t, w))
	requireStatus(t, makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/writeups?game_id=%d&team_id=%d", gameID, teamID), nil, ownerToken), http.StatusOK, "list writeups")
	requireStatus(t, makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/writeups/%d", writeupID), nil, ownerToken), http.StatusOK, "get writeup")
	requireStatus(t, makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/writeups/%d", writeupID), nil, ownerToken), http.StatusNoContent, "delete writeup")
}

func membershipIDForUser(t *testing.T, engine *gin.Engine, teamID, userID int64, token string) int64 {
	t.Helper()
	w := makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d/members", teamID), nil, token)
	requireStatus(t, w, http.StatusOK, "list team members")
	for _, member := range parseItems(t, w) {
		rawUserID, ok := member["user_id"].(float64)
		if ok && int64(rawUserID) == userID {
			return jsonID(t, member)
		}
	}
	t.Fatalf("membership for user %d not found in team %d", userID, teamID)
	return 0
}

func parseNumberArray(t *testing.T, w *httptest.ResponseRecorder) []int64 {
	t.Helper()
	var raw []float64
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("parsing number array: %v, body: %s", err, w.Body.String())
	}
	result := make([]int64, len(raw))
	for i, v := range raw {
		result[i] = int64(v)
	}
	return result
}
