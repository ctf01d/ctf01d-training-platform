package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
)

type authResponse struct {
	HasUser bool   `json:"has_user"`
	Role    string `json:"role"`
	UserID  int64  `json:"user_id"`
}

func setupOpenAPIRouter(_ *auth.Manager, method, path string, requiresBearer bool, middlewares ...httpserver.MiddlewareFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Handle(method, path, func(c *gin.Context) {
		if requiresBearer {
			c.Set(string(httpserver.BearerAuthScopes), []string{})
		}
		for _, middleware := range middlewares {
			middleware(c)
			if c.IsAborted() {
				return
			}
		}

		userID, hasUser := CurrentUserID(c)
		role, _ := CurrentRole(c)
		c.JSON(http.StatusOK, authResponse{
			HasUser: hasUser,
			Role:    role,
			UserID:  userID,
		})
	})
	return r
}

func makeToken(t *testing.T, mgr *auth.Manager, userID int64, role, userName string) string {
	t.Helper()

	token, err := mgr.Generate(userID, role, userName, "test-jti")
	if err != nil {
		t.Fatalf("generating token: %v", err)
	}
	return token
}

func readAuthResponse(t *testing.T, w *httptest.ResponseRecorder) authResponse {
	t.Helper()

	var body authResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

func TestOpenAPIAuth_RequiredBearerMissingToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupOpenAPIRouter(mgr, http.MethodGet, "/test", true, OpenAPIAuth(mgr, nil))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOpenAPIAuth_PublicRouteWithoutToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupOpenAPIRouter(mgr, http.MethodGet, "/test", false, OpenAPIAuth(mgr, nil))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := readAuthResponse(t, w)
	if body.HasUser {
		t.Fatalf("expected no user context, got %+v", body)
	}
}

func TestOpenAPIAuth_ValidToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupOpenAPIRouter(mgr, http.MethodGet, "/test", true, OpenAPIAuth(mgr, nil))

	token := makeToken(t, mgr, 42, "player", "alice")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := readAuthResponse(t, w)
	if !body.HasUser || body.UserID != 42 || body.Role != "player" {
		t.Fatalf("unexpected user context: %+v", body)
	}
}

func TestOpenAPIAuth_ExpiredToken(t *testing.T) {
	expiredMgr := auth.NewManager("test-secret", -1)
	token := makeToken(t, expiredMgr, 1, "player", "alice")

	validMgr := auth.NewManager("test-secret", 24)
	r := setupOpenAPIRouter(validMgr, http.MethodGet, "/test", true, OpenAPIAuth(validMgr, nil))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOpenAPIRole_InsufficientRole(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupOpenAPIRouter(
		mgr,
		http.MethodPatch,
		"/users/:id/role",
		true,
		OpenAPIAuth(mgr, nil),
		OpenAPIRole(),
	)

	token := makeToken(t, mgr, 1, "player", "bob")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/users/1/role", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOpenAPIRole_SufficientRole(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupOpenAPIRouter(
		mgr,
		http.MethodPatch,
		"/users/:id/role",
		true,
		OpenAPIAuth(mgr, nil),
		OpenAPIRole(),
	)

	token := makeToken(t, mgr, 1, "admin", "admin")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/users/1/role", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRoleHierarchy(t *testing.T) {
	tests := []struct {
		current  string
		required string
		ok       bool
	}{
		{"guest", "guest", true},
		{"guest", "player", false},
		{"guest", "admin", false},
		{"player", "guest", true},
		{"player", "player", true},
		{"player", "admin", false},
		{"admin", "guest", true},
		{"admin", "player", true},
		{"admin", "admin", true},
	}
	for _, tt := range tests {
		got := hasRoleLevel(tt.current, tt.required)
		if got != tt.ok {
			t.Errorf("hasRoleLevel(%q, %q) = %v, want %v", tt.current, tt.required, got, tt.ok)
		}
	}
}
