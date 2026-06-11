package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/gin-gonic/gin"
)

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

func TestRespondError_NotFound(t *testing.T) {
	r := setupGin()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, errs.ErrNotFound)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	resp := decodeBody(t, w)
	if resp.Code != "not_found" {
		t.Errorf("expected code not_found, got %s", resp.Code)
	}
}

func TestRespondError_Conflict(t *testing.T) {
	r := setupGin()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, errs.ErrConflict)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestRespondError_Forbidden(t *testing.T) {
	r := setupGin()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, errs.ErrForbidden)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRespondError_Unauthorized(t *testing.T) {
	r := setupGin()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, errs.ErrUnauthorized)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRespondError_Validation(t *testing.T) {
	r := setupGin()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, errs.NewValidationError(map[string]string{"email": "invalid"}))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
	resp := decodeBody(t, w)
	if resp.Code != "validation_error" {
		t.Errorf("expected code validation_error, got %s", resp.Code)
	}
	if resp.Details == nil {
		t.Error("expected details to be set")
	}
	if resp.Details["email"] != "invalid" {
		t.Errorf("expected details.email=invalid, got %v", resp.Details["email"])
	}
}

func TestRespondError_WrappedSentinel(t *testing.T) {
	r := setupGin()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, fmt.Errorf("user 42: %w", errs.ErrNotFound))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for wrapped error, got %d", w.Code)
	}
}

func TestRespondError_Unknown(t *testing.T) {
	r := setupGin()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, errors.New("something unexpected"))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	resp := decodeBody(t, w)
	if resp.Code != "internal_error" {
		t.Errorf("expected code internal_error, got %s", resp.Code)
	}
}

func TestBindJSON_Success(t *testing.T) {
	type sample struct {
		Name string `json:"name" binding:"required"`
	}

	r := setupGin()
	r.POST("/test", func(c *gin.Context) {
		req, ok := bindJSON[sample](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	body := `{"name":"test"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestBindJSON_InvalidJSON(t *testing.T) {
	type sample struct {
		Name string `json:"name" binding:"required"`
	}

	r := setupGin()
	r.POST("/test", func(c *gin.Context) {
		_, ok := bindJSON[sample](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, nil)
	})

	body := `{invalid`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestBindJSON_ValidationFail(t *testing.T) {
	type sample struct {
		Name string `json:"name" binding:"required"`
	}

	r := setupGin()
	r.POST("/test", func(c *gin.Context) {
		_, ok := bindJSON[sample](c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, nil)
	})

	body := `{"name":""}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for validation failure, got %d", w.Code)
	}
}
