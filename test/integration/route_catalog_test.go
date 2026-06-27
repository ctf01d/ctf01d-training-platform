package integration

import (
	"net/http"
	"testing"
)

func TestHTTPRouteCatalogIsCoveredByIntegrationSuite(t *testing.T) {
	engine, _ := setupTest(t)

	expected := map[string]bool{
		"GET /healthz":                                  true,
		"GET /version":                                  true,
		"POST /api/v1/session":                          true,
		"DELETE /api/v1/session":                        true,
		"GET /api/v1/profile":                           true,
		"PATCH /api/v1/profile":                         true,
		"PUT /api/v1/profile/password":                  true,
		"POST /api/v1/profile/avatar":                   true,
		"GET /api/v1/profile/sessions":                  true,
		"GET /api/v1/users":                             true,
		"POST /api/v1/users":                            true,
		"GET /api/v1/users/:id":                         true,
		"PATCH /api/v1/users/:id":                       true,
		"PATCH /api/v1/users/:id/role":                  true,
		"DELETE /api/v1/users/:id":                      true,
		"PATCH /api/v1/users/:id/profile":               true,
		"PUT /api/v1/users/:id/password":                true,
		"POST /api/v1/users/:id/block":                  true,
		"GET /api/v1/users/:id/avatar":                  true,
		"POST /api/v1/users/:id/avatar":                 true,
		"GET /api/v1/users/:id/sessions":                true,
		"DELETE /api/v1/users/:id/sessions/:sessionId":  true,
		"GET /api/v1/universities":                      true,
		"POST /api/v1/universities":                     true,
		"GET /api/v1/universities/:id":                  true,
		"PATCH /api/v1/universities/:id":                true,
		"DELETE /api/v1/universities/:id":               true,
		"GET /api/v1/teams":                             true,
		"POST /api/v1/teams":                            true,
		"GET /api/v1/teams/:id":                         true,
		"PATCH /api/v1/teams/:id":                       true,
		"DELETE /api/v1/teams/:id":                      true,
		"POST /api/v1/teams/:id/join-request":           true,
		"POST /api/v1/teams/:id/invite":                 true,
		"GET /api/v1/teams/:id/members":                 true,
		"GET /api/v1/teams/:id/events":                  true,
		"GET /api/v1/team-memberships":                  true,
		"POST /api/v1/team-memberships":                 true,
		"GET /api/v1/team-memberships/:id":              true,
		"PATCH /api/v1/team-memberships/:id":            true,
		"DELETE /api/v1/team-memberships/:id":           true,
		"POST /api/v1/team-memberships/:id/approve":     true,
		"POST /api/v1/team-memberships/:id/reject":      true,
		"POST /api/v1/team-memberships/:id/accept":      true,
		"POST /api/v1/team-memberships/:id/decline":     true,
		"POST /api/v1/team-memberships/:id/set-role":    true,
		"GET /api/v1/games":                             true,
		"POST /api/v1/games":                            true,
		"GET /api/v1/games/:id":                         true,
		"PATCH /api/v1/games/:id":                       true,
		"DELETE /api/v1/games/:id":                      true,
		"POST /api/v1/games/:id/finalize":               true,
		"POST /api/v1/games/:id/unfinalize":             true,
		"POST /api/v1/games/:id/publish":                true,
		"GET /api/v1/games/:id/services":                true,
		"POST /api/v1/games/:id/services":               true,
		"DELETE /api/v1/games/:id/services/:service_id": true,
		"PATCH /api/v1/games/:id/services/:service_id":  true,
		"GET /api/v1/games/:id/teams":                   true,
		"POST /api/v1/games/:id/teams/reorder":          true,
		"GET /api/v1/games/:id/scoreboard":              true,
		"GET /api/v1/games/:id/export/ctf01d/options":   true,
		"POST /api/v1/games/:id/export/ctf01d":          true,
		"POST /api/v1/game-teams":                       true,
		"PATCH /api/v1/game-teams/:id":                  true,
		"DELETE /api/v1/game-teams/:id":                 true,
		"GET /api/v1/results":                           true,
		"POST /api/v1/results":                          true,
		"GET /api/v1/results/:id":                       true,
		"PATCH /api/v1/results/:id":                     true,
		"DELETE /api/v1/results/:id":                    true,
		"GET /api/v1/writeups":                          true,
		"POST /api/v1/writeups":                         true,
		"GET /api/v1/writeups/:id":                      true,
		"DELETE /api/v1/writeups/:id":                   true,
		"GET /api/v1/scoreboard":                        true,
		"GET /api/v1/services":                          true,
		"POST /api/v1/services":                         true,
		"POST /api/v1/services/import/github":           true,
		"POST /api/v1/services/import/zip":              true,
		"POST /api/v1/services/import/github/preview":   true,
		"POST /api/v1/services/import/zip/preview":      true,
		"DELETE /api/v1/services/:id":                   true,
		"GET /api/v1/services/:id":                      true,
		"PATCH /api/v1/services/:id":                    true,
		"POST /api/v1/services/:id/check-checker":       true,
		"GET /api/v1/services/:id/download/:kind":       true,
		"POST /api/v1/services/:id/redownload":          true,
		"POST /api/v1/services/:id/toggle-public":       true,
		"POST /api/v1/services/:id/upload-archives":     true,
	}

	actual := make(map[string]bool)
	for _, route := range engine.Routes() {
		if route.Method == http.MethodOptions {
			continue
		}
		actual[route.Method+" "+route.Path] = true
	}

	for route := range actual {
		if !expected[route] {
			t.Errorf("route is not listed in e2e coverage catalog: %s", route)
		}
	}
	for route := range expected {
		if !actual[route] {
			t.Errorf("e2e coverage catalog contains missing route: %s", route)
		}
	}
}
