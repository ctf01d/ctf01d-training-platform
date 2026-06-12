package integration

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServicesFlow(t *testing.T) {
	engine, store := setupTest(t)

	_, adminToken := seedUser(t, store, "admin", "Admin", "admin12345", "admin")
	_, playerToken := seedUser(t, store, "player1", "Player One", "password123", "player")
	_, guestToken := seedUser(t, store, "guest1", "Guest", "password123", "guest")

	t.Log("Step: Create a service")
	w := makeReq(t, engine, http.MethodPost, "/api/v1/services", map[string]interface{}{
		"name":                "test-service",
		"public_description":  "A test service",
		"private_description": "Internal details",
		"public":              true,
	}, playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create service: %d %s", w.Code, w.Body.String())
	}
	svc := parseJSON(t, w)
	svcID := int64(svc["id"].(float64))
	if svc["name"] != "test-service" {
		t.Errorf("expected name test-service, got %v", svc["name"])
	}
	if svc["public"] != true {
		t.Errorf("expected public=true")
	}

	t.Log("Step: Guest cannot create service")
	w = makeReq(t, engine, http.MethodPost, "/api/v1/services", map[string]interface{}{
		"name": "unauthorized",
	}, guestToken)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for guest creating service, got %d", w.Code)
	}

	t.Log("Step: Get service - non-admin sees no private_description")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/services/%d", svcID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get service: %d %s", w.Code, w.Body.String())
	}
	svc = parseJSON(t, w)
	if svc["private_description"] != nil {
		t.Errorf("private_description should be hidden from non-admin, got %v", svc["private_description"])
	}

	t.Log("Step: Admin sees private_description")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/services/%d", svcID), nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get service admin: %d %s", w.Code, w.Body.String())
	}
	svc = parseJSON(t, w)
	if svc["private_description"] == nil {
		t.Errorf("admin should see private_description")
	}

	t.Log("Step: List services")
	w = makeReq(t, engine, http.MethodGet, "/api/v1/services", nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list services: %d %s", w.Code, w.Body.String())
	}
	list := parseJSON(t, w)
	items := list["items"].([]interface{})
	if len(items) < 1 {
		t.Errorf("expected at least 1 service, got %d", len(items))
	}

	t.Log("Step: List services with public filter")
	w = makeReq(t, engine, http.MethodGet, "/api/v1/services?public=true", nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("list public services: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: List services with search")
	w = makeReq(t, engine, http.MethodGet, "/api/v1/services?q=test", nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("search services: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Update service")
	w = makeReq(t, engine, http.MethodPatch, fmt.Sprintf("/api/v1/services/%d", svcID), map[string]interface{}{
		"name": "updated-service",
	}, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("update service: %d %s", w.Code, w.Body.String())
	}
	svc = parseJSON(t, w)
	if svc["name"] != "updated-service" {
		t.Errorf("expected name updated-service, got %v", svc["name"])
	}

	t.Log("Step: Toggle public")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/services/%d/toggle-public", svcID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("toggle public: %d %s", w.Code, w.Body.String())
	}
	svc = parseJSON(t, w)
	if svc["public"] != false {
		t.Errorf("expected public=false after toggle")
	}

	t.Log("Step: Toggle back to public")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/services/%d/toggle-public", svcID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("toggle public back: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Upload archives")
	zipBuf := createTestZip(t, map[string]string{"service/hello.txt": "hello world"})
	w = makeMultipartUpload(t, engine, fmt.Sprintf("/api/v1/services/%d/upload-archives", svcID), zipBuf, "service_archive", "service.zip", playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("upload archives: %d %s", w.Code, w.Body.String())
	}
	svc = parseJSON(t, w)
	if svc["service_archive"] != nil {
		t.Errorf("service_archive metadata should be hidden from non-admin, got %v", svc["service_archive"])
	}

	t.Log("Step: Admin sees uploaded archive metadata")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/services/%d", svcID), nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("get service after upload as admin: %d %s", w.Code, w.Body.String())
	}
	svc = parseJSON(t, w)
	if svc["service_archive"] == nil {
		t.Errorf("expected service_archive metadata for admin after upload")
	}

	t.Log("Step: Download service archive")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/services/%d/download/service", svcID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("download service archive: %d %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/zip" {
		t.Errorf("expected content-type application/zip, got %s", ct)
	}
	if w.Body.Len() == 0 {
		t.Errorf("expected non-empty body for download")
	}

	t.Log("Step: Check checker (no checker uploaded - should return unknown status)")
	w = makeReq(t, engine, http.MethodPost, fmt.Sprintf("/api/v1/services/%d/check-checker", svcID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("check checker: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Import from zip")
	importZip := createServiceBundleZip(t, "imported-service", "Imported description")
	w = makeMultipartUpload(t, engine, "/api/v1/services/import/zip", importZip, "archive", "bundle.zip", playerToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("import from zip: %d %s", w.Code, w.Body.String())
	}
	importResult := parseJSON(t, w)
	importedSvc := importResult["service"].(map[string]interface{})
	importedID := int64(importedSvc["id"].(float64))
	if importedSvc["name"] != "imported-service" {
		t.Errorf("expected imported-service, got %v", importedSvc["name"])
	}

	t.Log("Step: Download imported service archive")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/services/%d/download/service", importedID), nil, playerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("download imported archive: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Delete service")
	w = makeReq(t, engine, http.MethodDelete, fmt.Sprintf("/api/v1/services/%d", svcID), nil, playerToken)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete service: %d %s", w.Code, w.Body.String())
	}

	t.Log("Step: Verify service deleted")
	w = makeReq(t, engine, http.MethodGet, fmt.Sprintf("/api/v1/services/%d", svcID), nil, playerToken)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted service, got %d", w.Code)
	}

	_ = adminToken
}

func createTestZip(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("creating zip entry %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("writing zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing zip: %v", err)
	}
	return &buf
}

func createServiceBundleZip(t *testing.T, name, description string) *bytes.Buffer {
	t.Helper()
	trainingJSON := fmt.Sprintf(`{"display_name":"%s","description":"%s"}`, name, description)
	return createTestZip(t, map[string]string{
		"service/hello.txt":            "hello",
		"service/ctf01d-training.json": trainingJSON,
		"checker/checker.sh":           "#!/bin/bash\necho 101",
	})
}

func makeMultipartUpload(t *testing.T, engine *gin.Engine, path string, fileData *bytes.Buffer, fieldName, fileName, token string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	if _, err := io.Copy(part, fileData); err != nil {
		t.Fatalf("copying file data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing multipart writer: %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}
