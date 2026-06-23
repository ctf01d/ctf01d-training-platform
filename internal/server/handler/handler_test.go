package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// HandleUpdateUserRole must reject an admin trying to change their own role so
// that the last admin cannot accidentally lock everyone out (issue #79). The
// check fires before the service is touched, so a zero-value Handler is enough.
func TestHandleUpdateUserRole_RejectsSelf(t *testing.T) {
	h := &Handler{}
	r := setupGin()
	r.PUT("/users/:id/role", func(c *gin.Context) {
		c.Set("user_id", int64(7))
		h.HandleUpdateUserRole(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/users/7/role",
		strings.NewReader(`{"role":"guest"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when changing own role, got %d (%s)", w.Code, w.Body.String())
	}
	if resp := decodeBody(t, w); resp.Code != codeForbidden {
		t.Errorf("expected code %q, got %q", codeForbidden, resp.Code)
	}
}

func TestHandleUpdateUserRole_Unauthenticated(t *testing.T) {
	h := &Handler{}
	r := setupGin()
	r.PUT("/users/:id/role", func(c *gin.Context) {
		h.HandleUpdateUserRole(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/users/7/role",
		strings.NewReader(`{"role":"guest"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without an authenticated user, got %d", w.Code)
	}
}
