package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

func setupRouter(jwtMgr *auth.Manager, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(handler)
	r.GET("/test", func(c *gin.Context) {
		userID, _ := CurrentUserID(c)
		role, _ := CurrentRole(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": role})
	})
	return r
}

func makeToken(t *testing.T, mgr *auth.Manager, userID int64, role, userName string) string {
	t.Helper()
	token, err := mgr.Generate(userID, role, userName)
	if err != nil {
		t.Fatalf("generating token: %v", err)
	}
	return token
}

func TestRequireAuth_NoToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupRouter(mgr, RequireAuth(mgr))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupRouter(mgr, RequireAuth(mgr))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	r := setupRouter(mgr, RequireAuth(mgr))

	token := makeToken(t, mgr, 42, "player", "alice")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !containsStr(body, `"user_id":42`) {
		t.Errorf("expected user_id 42 in body, got %s", body)
	}
}

func TestRequireRole_InsufficientRole(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAuth(mgr))
	r.Use(RequireRole("admin"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := makeToken(t, mgr, 1, "player", "bob")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireRole_SufficientRole(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAuth(mgr))
	r.Use(RequireRole("player"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := makeToken(t, mgr, 1, "admin", "admin")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuth_NoToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OptionalAuth(mgr))
	r.GET("/test", func(c *gin.Context) {
		_, exists := CurrentUserID(c)
		c.JSON(http.StatusOK, gin.H{"has_user": exists})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuth_ValidToken(t *testing.T) {
	mgr := auth.NewManager("test-secret", 24)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OptionalAuth(mgr))
	r.GET("/test", func(c *gin.Context) {
		userID, exists := CurrentUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "has_user": exists})
	})

	token := makeToken(t, mgr, 7, "guest", "visitor")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	expiredMgr := auth.NewManager("test-secret", -1)
	token := makeToken(t, expiredMgr, 1, "player", "alice")

	validMgr := auth.NewManager("test-secret", 24)
	r := setupRouter(validMgr, RequireAuth(validMgr))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
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

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
