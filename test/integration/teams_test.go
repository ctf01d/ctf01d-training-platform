package integration

import (
	"fmt"
	"net/http"
	"testing"
)

func TestTeamsMembershipsFlow(t *testing.T) {
	engine, store := setupTest(t)

	_, adminToken := seedUser(t, store, "admin", "Admin", "admin12345", "admin")
	_, ownerToken := seedUser(t, store, "owner1", "Owner", "password123", "player")
	player1ID, player1Token := seedUser(t, store, "player1", "Player One", "password123", "player")
	player2ID, player2Token := seedUser(t, store, "player2", "Player Two", "password123", "player")
	_, outsiderToken := seedUser(t, store, "outsider", "Outsider", "password123", "player")

	t.Log("Step: Owner creates team")
	w := makeReq(t, engine, http.MethodPost, "/api/v1/teams", map[string]interface{}{
		"name": "Test Team", "description": "A test team",
	}, ownerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	team := parseJSON(t, w)
	teamID := int64(team["id"].(float64))
	if team["name"] != "Test Team" {
		t.Errorf("expected team name 'Test Team', got %v", team["name"])
	}

	t.Log("Step: Owner invites player1")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/teams/%d/invite", teamID), map[string]interface{}{
		"user_id": player1ID,
	}, ownerToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("invite player1: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: List members after invite")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d/members", teamID), nil, ownerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list members: %d %s", w.Code, w.Body.String())
	}
	members := parseItems(t, w)
	var inviteID float64
	for _, m := range members {
		if int64(m["user_id"].(float64)) == player1ID {
			inviteID = m["id"].(float64)
			if m["status"] != "pending" {
				t.Errorf("expected pending status for invited player, got %v", m["status"])
			}
		}
	}
	if inviteID == 0 {
		t.Fatal("invite not found in members list")
	}

	t.Log("Step: Player1 accepts invitation")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/team-memberships/%d/accept", int64(inviteID)), nil, player1Token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("accept invite: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Owner sets player1 as captain")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d/members", teamID), nil, ownerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list members: %d %s", w.Code, w.Body.String())
	}
	members = parseItems(t, w)
	var player1MemberID float64
	for _, m := range members {
		if int64(m["user_id"].(float64)) == player1ID {
			player1MemberID = m["id"].(float64)
		}
	}
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/team-memberships/%d/set-role", int64(player1MemberID)), map[string]interface{}{
		"role": "captain",
	}, ownerToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("set role captain: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Verify captain_id on team")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d", teamID), nil, ownerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get team: %d %s", w.Code, w.Body.String())
	}
	team = parseJSON(t, w)
	if team["captain_id"] == nil {
		t.Fatal("expected captain_id to be set")
	}
	captainID := int64(team["captain_id"].(float64))
	if captainID != player1ID {
		t.Errorf("expected captain_id=%d, got %d", player1ID, captainID)
	}

	t.Log("Step: Player2 requests to join")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/teams/%d/join-request", teamID), nil, player2Token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("join request: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Get player2's membership ID")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d/members", teamID), nil, ownerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list members: %d %s", w.Code, w.Body.String())
	}
	members = parseItems(t, w)
	var joinReqID float64
	for _, m := range members {
		if int64(m["user_id"].(float64)) == player2ID {
			joinReqID = m["id"].(float64)
			if m["status"] != "pending" {
				t.Errorf("expected pending for join request, got %v", m["status"])
			}
		}
	}
	if joinReqID == 0 {
		t.Fatal("join request not found in members list")
	}

	t.Log("Step: Owner approves join request")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/team-memberships/%d/approve", int64(joinReqID)), nil, ownerToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("approve: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: List events")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d/events", teamID), nil, ownerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list events: %d %s", w.Code, w.Body.String())
	}
	eventsResult := parseJSON(t, w)
	events := eventsResult["items"].([]interface{})
	if len(events) < 5 {
		t.Errorf("expected at least 5 events, got %d", len(events))
	}

	t.Log("Step: Outsider cannot manage team (update)")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/teams/%d", teamID), map[string]interface{}{
		"name": "Hacked Team",
	}, outsiderToken)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for outsider update, got %d", w.Code)
	}

	t.Log("Step: Outsider cannot invite")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/teams/%d/invite", teamID), map[string]interface{}{
		"user_id": player1ID,
	}, outsiderToken)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for outsider invite, got %d", w.Code)
	}

	t.Log("Step: Universities CRUD (admin)")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/universities", map[string]interface{}{
		"name": "Test University", "site_url": "https://test.edu",
	}, adminToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create university: %d %s", w.Code, w.Body.String())
	}
	uni := parseJSON(t, w)
	uniID := int64(uni["id"].(float64))

	w = makeReq(t, engine, http.MethodGet, "/api/v1/universities", nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list universities: %d %s", w.Code, w.Body.String())
	}

	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/universities/%d", uniID), nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get university: %d %s", w.Code, w.Body.String())
	}

	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/universities/%d", uniID), map[string]interface{}{
		"name": "Updated University",
	}, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("update university: %d %s", w.Code, w.Body.String())
	}

	w = makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/universities/%d", uniID), nil, adminToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete university: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Player cannot create university")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/universities", map[string]interface{}{
		"name": "Unauthorized Uni",
	}, player1Token)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for player creating university, got %d", w.Code)
	}

	t.Log("Step: List teams")
	w = makeReq(t, engine, http.MethodGet, "/api/v1/teams", nil, ownerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list teams: %d %s", w.Code, w.Body.String())
	}
	teamsResult := parseJSON(t, w)
	items := teamsResult["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 team, got %d", len(items))
	}
}

func TestUniversitiesOrderedByTeamCount(t *testing.T) {
	engine, store := setupTest(t)

	_, adminToken := seedUser(t, store, "admin_uni_order", "Admin", "admin12345", "admin")
	_, ownerToken := seedUser(t, store, "owner_uni_order", "Owner", "password123", "player")

	createUniversity := func(name string) int64 {
		t.Helper()
		w := makeReq(t, engine, http.MethodPost, "/api/v1/universities", map[string]interface{}{
			"name": name,
		}, adminToken)
		if w.Code != http.StatusCreated {
			t.Fatalf("create university %q: %d %s", name, w.Code, w.Body.String())
		}
		return jsonID(t, parseJSON(t, w))
	}

	createTeam := func(name string, universityID int64) {
		t.Helper()
		w := makeReq(t, engine, http.MethodPost, "/api/v1/teams", map[string]interface{}{
			"name":          name,
			"university_id": universityID,
		}, ownerToken)
		if w.Code != http.StatusCreated {
			t.Fatalf("create team %q: %d %s", name, w.Code, w.Body.String())
		}
	}

	uniOneID := createUniversity("University One")
	uniTwoID := createUniversity("University Two")
	uniThreeID := createUniversity("University Three")

	createTeam("Team One A", uniOneID)
	createTeam("Team Two A", uniTwoID)
	createTeam("Team Two B", uniTwoID)

	w := makeReq(t, engine, http.MethodGet, "/api/v1/universities", nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list universities: %d %s", w.Code, w.Body.String())
	}

	items := parseItems(t, w)
	if len(items) < 3 {
		t.Fatalf("expected at least 3 universities, got %d", len(items))
	}

	gotOrder := []int64{
		jsonID(t, items[0]),
		jsonID(t, items[1]),
		jsonID(t, items[2]),
	}
	wantOrder := []int64{uniTwoID, uniOneID, uniThreeID}

	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("unexpected universities order: got %v, want %v", gotOrder, wantOrder)
		}
	}
}
