package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/handler"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type mockStore struct {
	err error
}

func (m *mockStore) Ping() error {
	return m.err
}

func newTestEngine(store Store) *gin.Engine {
	cfg := &config.Config{
		Env: "development",
		CORS: config.CORSConfig{
			AllowedOrigins: "http://localhost:5173",
		},
	}
	log, _ := zap.NewDevelopment()
	jwtMgr := auth.NewManager("test-secret", 24)
	h := handler.New(nil, nil, jwtMgr, nil, nil, nil, nil, nil, nil, nil, nil)
	return New(cfg, log, store, h)
}

func TestHealthz_OK(t *testing.T) {
	engine := newTestEngine(&mockStore{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHealthz_Unhealthy(t *testing.T) {
	engine := newTestEngine(&mockStore{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVersion(t *testing.T) {
	engine := newTestEngine(&mockStore{})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
