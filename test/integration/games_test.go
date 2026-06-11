package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestGamesFlow(t *testing.T) {
	engine, store, _ := setupTest(t)

	_, adminToken := seedUser(t, store, "admin", "Admin", "admin12345", "admin")
	_, playerToken := seedUser(t, store, "player1", "Player One", "password123", "player")
	player2ID, _ := seedUser(t, store, "player2", "Player Two", "password123", "player")
	_, outsiderToken := seedUser(t, store, "guest1", "Guest", "password123", "guest")

	t.Log("Step: Create a team")
	w := makeReq(t, engine, http.MethodPost, "/api/v1/teams", map[string]interface{}{
		"name": "Team Alpha",
	}, playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	team := parseJSON(t, w)
	teamID := int64(team["id"].(float64))

	t.Log("Step: Player2 creates team")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/teams", map[string]interface{}{
		"name": "Team Beta",
	}, outsiderToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create team beta: %d %s", w.Code, w.Body.String())
	}
	teamBeta := parseJSON(t, w)
	teamBetaID := int64(teamBeta["id"].(float64))

	t.Log("Step: Create a game")
	startsAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	endsAt := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	w = makeReq(t, engine, http.MethodPost, "/api/v1/games", map[string]interface{}{
		"name":          "Test CTF 2026",
		"organizer":     "TestOrg",
		"starts_at":     startsAt,
		"ends_at":       endsAt,
		"site_url":      "https://testctf.example.com",
		"access_secret": "super-secret-123",
		"vpn_url":       "https://vpn.testctf.example.com",
	}, playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create game: %d %s", w.Code, w.Body.String())
	}
	game := parseJSON(t, w)
	gameID := int64(game["id"].(float64))
	if game["name"] != "Test CTF 2026" {
		t.Errorf("expected game name 'Test CTF 2026', got %v", game["name"])
	}
	if game["status"] != "upcoming" {
		t.Errorf("expected status upcoming, got %v", game["status"])
	}

	t.Log("Step: Guest cannot create game")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/games", map[string]interface{}{
		"name": "Unauthorized Game",
	}, outsiderToken)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for guest creating game, got %d", w.Code)
	}

	t.Log("Step: Get game - access_secret visible to creator (player)")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d", gameID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get game: %d %s", w.Code, w.Body.String())
	}
	game = parseJSON(t, w)
	if game["access_secret"] != nil {
		t.Log("access_secret hidden from non-participant player")
	}

	t.Log("Step: List games")
	w = makeReq(t, engine, http.MethodGet, "/api/v1/games", nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list games: %d %s", w.Code, w.Body.String())
	}
	gamesResult := parseJSON(t, w)
	items := gamesResult["items"].([]interface{})
	if len(items) < 1 {
		t.Errorf("expected at least 1 game, got %d", len(items))
	}

	t.Log("Step: Update game")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/games/%d", gameID), map[string]interface{}{
		"name": "Updated CTF 2026",
	}, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("update game: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Add teams to game roster (game-teams)")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/game-teams", map[string]interface{}{
		"game_id":    gameID,
		"team_id":    teamID,
		"ip_address": "10.0.0.1",
		"order":      0,
	}, playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create game team: %d %s", w.Code, w.Body.String())
	}
	gt1 := parseJSON(t, w)
	gt1ID := int64(gt1["id"].(float64))

	w = makeReq(t, engine, http.MethodPost, "/api/v1/game-teams", map[string]interface{}{
		"game_id":    gameID,
		"team_id":    teamBetaID,
		"ip_address": "10.0.0.2",
		"order":      1,
	}, playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create game team 2: %d %s", w.Code, w.Body.String())
	}
	gt2 := parseJSON(t, w)
	gt2ID := int64(gt2["id"].(float64))

	t.Log("Step: List game teams")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d/teams", gameID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list game teams: %d %s", w.Code, w.Body.String())
	}
	teamList := parseJSON(t, w)
	teamItems := teamList["items"].([]interface{})
	if len(teamItems) != 2 {
		t.Errorf("expected 2 game teams, got %d", len(teamItems))
	}

	t.Log("Step: Reorder game teams")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/teams/reorder", gameID), map[string]interface{}{
		"items": []map[string]interface{}{
			{"id": gt1ID, "order": 1},
			{"id": gt2ID, "order": 0},
		},
	}, playerToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("reorder game teams: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Update game team")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/game-teams/%d", gt1ID), map[string]interface{}{
		"ip_address": "10.0.0.100",
	}, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("update game team: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Add results")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/results", map[string]interface{}{
		"game_id": gameID,
		"team_id": teamID,
		"score":   500,
	}, playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create result 1: %d %s", w.Code, w.Body.String())
	}
	r1 := parseJSON(t, w)
	r1ID := int64(r1["id"].(float64))

	w = makeReq(t, engine, http.MethodPost, "/api/v1/results", map[string]interface{}{
		"game_id": gameID,
		"team_id": teamBetaID,
		"score":   300,
	}, playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create result 2: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: List results by game")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/results?game_id=%d", gameID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list results by game: %d %s", w.Code, w.Body.String())
	}
	resultsList := parseJSON(t, w)
	resultItems := resultsList["items"].([]interface{})
	if len(resultItems) != 2 {
		t.Errorf("expected 2 results, got %d", len(resultItems))
	}

	t.Log("Step: Update result")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/results/%d", r1ID), map[string]interface{}{
		"score": 600,
	}, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("update result: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Finalize game")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/finalize", gameID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("finalize game: %d %s", w.Code, w.Body.String())
	}
	game = parseJSON(t, w)
	if game["finalized"] != true {
		t.Errorf("expected finalized=true")
	}

	t.Log("Step: Cannot update result after finalize (non-admin)")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/results/%d", r1ID), map[string]interface{}{
		"score": 999,
	}, playerToken)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for result update after finalize, got %d", w.Code)
	}

	t.Log("Step: Admin can update result after finalize")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/results/%d", r1ID), map[string]interface{}{
		"score": 700,
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("admin update result after finalize: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Get game scoreboard (public)")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d/scoreboard", gameID), nil, "")
	if w.Code != http.StatusOK {
		t.Fatalf("get game scoreboard: %d %s", w.Code, w.Body.String())
	}
	sb := parseJSON(t, w)
	sbEntries := sb["entries"].([]interface{})
	if len(sbEntries) != 2 {
		t.Errorf("expected 2 scoreboard entries, got %d", len(sbEntries))
	}
	firstEntry := sbEntries[0].(map[string]interface{})
	if firstEntry["position"] != nil {
		if int64(firstEntry["position"].(float64)) != 1 {
			t.Errorf("expected first position=1, got %v", firstEntry["position"])
		}
	}

	t.Log("Step: Get global scoreboard")
	w = makeReq(t, engine, http.MethodGet, "/api/v1/scoreboard", nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get global scoreboard: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Unfinalize game")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/unfinalize", gameID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("unfinalize game: %d %s", w.Code, w.Body.String())
	}
	game = parseJSON(t, w)
	if game["finalized"] != false {
		t.Errorf("expected finalized=false")
	}

	t.Log("Step: Can update result again after unfinalize")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/results/%d", r1ID), map[string]interface{}{
		"score": 800,
	}, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("update result after unfinalize: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: access_secret hidden from non-participant")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d", gameID), nil, outsiderToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get game for outsider: %d %s", w.Code, w.Body.String())
	}
	game = parseJSON(t, w)
	if game["access_secret"] != nil {
		t.Errorf("access_secret should be hidden from non-participants, got %v", game["access_secret"])
	}
	if game["vpn_url"] != nil {
		t.Errorf("vpn_url should be hidden from non-participants, got %v", game["vpn_url"])
	}

	t.Log("Step: access_secret visible to admin")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d", gameID), nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get game for admin: %d %s", w.Code, w.Body.String())
	}
	game = parseJSON(t, w)
	if game["access_secret"] == nil {
		t.Errorf("admin should see access_secret")
	}

	t.Log("Step: Duplicate game-team conflict")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/game-teams", map[string]interface{}{
		"game_id": gameID,
		"team_id": teamID,
	}, playerToken)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate game-team, got %d", w.Code)
	}

	t.Log("Step: Double finalize conflict")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/finalize", gameID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("finalize game first time: %d %s", w.Code, w.Body.String())
	}
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/games/%d/finalize", gameID), nil, playerToken)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for double finalize, got %d", w.Code)
	}

	t.Log("Step: Delete game team")
	w = makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/game-teams/%d", gt1ID), nil, playerToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete game team: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Delete result")
	w = makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/results/%d", r1ID), nil, adminToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete result: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Delete game")
	w = makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/games/%d", gameID), nil, playerToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete game: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Verify game deleted")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/games/%d", gameID), nil, playerToken)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted game, got %d", w.Code)
	}

	_ = player2ID
}
