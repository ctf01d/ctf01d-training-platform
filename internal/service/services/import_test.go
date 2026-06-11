package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func createZip(files map[string]string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	w.Close()
	return buf.Bytes()
}

func createZipWithBytes(files map[string][]byte) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, _ := w.Create(name)
		f.Write(content)
	}
	w.Close()
	return buf.Bytes()
}

func createBundleZip(serviceFiles, checkerFiles map[string]string) []byte {
	files := make(map[string]string)
	for k, v := range serviceFiles {
		files["service/"+k] = v
	}
	for k, v := range checkerFiles {
		files["checker/"+k] = v
	}
	return createZip(files)
}

func createBundleZipWithSubdir(subdir string, serviceFiles, checkerFiles map[string]string) []byte {
	files := make(map[string]string)
	for k, v := range serviceFiles {
		files[subdir + "/service/" + k] = v
	}
	for k, v := range checkerFiles {
		files[subdir + "/checker/" + k] = v
	}
	return createZip(files)
}

func createFlatRepoZip(files map[string]string) []byte {
	prefixed := make(map[string]string)
	for k, v := range files {
		prefixed["myrepo/"+k] = v
	}
	return createZip(prefixed)
}

type mockImportQuerier struct {
	services    map[int64]*db.Service
	byName      map[string]int64
	nextID      int64
	serviceByID map[int64]*db.Service
	checkStatus map[int64]string
	checkedAt   map[int64]time.Time
	localPath   map[int64]map[string]string
}

func newMockImportQuerier() *mockImportQuerier {
	return &mockImportQuerier{
		services:    make(map[int64]*db.Service),
		byName:      make(map[string]int64),
		nextID:      1,
		serviceByID: make(map[int64]*db.Service),
		checkStatus: make(map[int64]string),
		checkedAt:   make(map[int64]time.Time),
		localPath:   make(map[int64]map[string]string),
	}
}

func (m *mockImportQuerier) GetServiceByName(_ context.Context, name string) (db.Service, error) {
	id, ok := m.byName[name]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	return *m.services[id], nil
}

func (m *mockImportQuerier) CreateService(_ context.Context, arg db.CreateServiceParams) (db.Service, error) {
	if _, exists := m.byName[arg.Name]; exists {
		return db.Service{}, fmt.Errorf("duplicate key value violates unique constraint")
	}
	id := m.nextID
	m.nextID++
	now := time.Now()
	svc := db.Service{
		ID:                  id,
		Name:                arg.Name,
		PublicDescription:   arg.PublicDescription,
		Copyright:           arg.Copyright,
		Public:              arg.Public,
		ServiceArchiveUrl:   arg.ServiceArchiveUrl,
		Ctf01dTraining:      arg.Ctf01dTraining,
		CheckStatus:         arg.CheckStatus,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	m.services[id] = &svc
	m.byName[arg.Name] = id
	m.serviceByID[id] = &svc
	return svc, nil
}

func (m *mockImportQuerier) UpdateService(_ context.Context, arg db.UpdateServiceParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.Name = arg.Name
	if arg.PublicDescription != nil {
		svc.PublicDescription = arg.PublicDescription
	}
	if arg.Copyright != nil {
		svc.Copyright = arg.Copyright
	}
	if arg.ServiceArchiveUrl != nil {
		svc.ServiceArchiveUrl = arg.ServiceArchiveUrl
	}
	if arg.Ctf01dTraining != nil {
		svc.Ctf01dTraining = arg.Ctf01dTraining
	}
	svc.Public = arg.Public
	svc.UpdatedAt = time.Now()
	m.byName[arg.Name] = arg.ID
	return *svc, nil
}

func (m *mockImportQuerier) GetServiceByID(_ context.Context, id int64) (db.Service, error) {
	svc, ok := m.services[id]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	return *svc, nil
}

func (m *mockImportQuerier) SetServiceLocal(_ context.Context, arg db.SetServiceLocalParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.ServiceLocalPath = arg.ServiceLocalPath
	svc.ServiceLocalSize = arg.ServiceLocalSize
	svc.ServiceLocalSha256 = arg.ServiceLocalSha256
	svc.ServiceDownloadedAt = arg.ServiceDownloadedAt
	return *svc, nil
}

func (m *mockImportQuerier) SetCheckerLocal(_ context.Context, arg db.SetCheckerLocalParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.CheckerLocalPath = arg.CheckerLocalPath
	svc.CheckerLocalSize = arg.CheckerLocalSize
	svc.CheckerLocalSha256 = arg.CheckerLocalSha256
	svc.CheckerDownloadedAt = arg.CheckerDownloadedAt
	return *svc, nil
}

func (m *mockImportQuerier) SetArchiveURLs(_ context.Context, arg db.SetArchiveURLsParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	if arg.ServiceArchiveUrl != nil {
		svc.ServiceArchiveUrl = arg.ServiceArchiveUrl
	}
	if arg.CheckerArchiveUrl != nil {
		svc.CheckerArchiveUrl = arg.CheckerArchiveUrl
	}
	return *svc, nil
}

func (m *mockImportQuerier) SetCheckStatus(_ context.Context, arg db.SetCheckStatusParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.CheckStatus = arg.CheckStatus
	svc.CheckedAt = arg.CheckedAt
	return *svc, nil
}

func TestBuildBundle_WithServiceDir(t *testing.T) {
	input := createZip(map[string]string{
		"service/README.md": "# My Service\n\nA great service",
		"service/main.py":   "print('hello')",
	})

	result, err := BuildBundle(input)
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}

	found := make(map[string]bool)
	for _, f := range r.File {
		found[f.Name] = true
	}
	if !found["service/README.md"] {
		t.Error("missing service/README.md")
	}
	if !found["service/main.py"] {
		t.Error("missing service/main.py")
	}
}

func TestBuildBundle_WithSubdir(t *testing.T) {
	input := createBundleZipWithSubdir("myrepo", map[string]string{
		"README.md": "# Test Service\n\nDescription here",
		"main.py":   "print('hello')",
	}, map[string]string{
		"checker.py": "# checker code with 101 and 102",
	})

	result, err := BuildBundle(input)
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}

	found := make(map[string]bool)
	for _, f := range r.File {
		found[f.Name] = true
	}
	if !found["service/README.md"] {
		t.Error("missing service/README.md")
	}
	if !found["service/main.py"] {
		t.Error("missing service/main.py")
	}
	if !found["checker/checker.py"] {
		t.Error("missing checker/checker.py")
	}
}

func TestBuildBundle_FlatRepo(t *testing.T) {
	input := createFlatRepoZip(map[string]string{
		"README.md": "# My Service\nFlat structure",
		"main.py":   "print('hello')",
	})

	result, err := BuildBundle(input)
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}

	hasServiceMain := false
	for _, f := range r.File {
		if f.Name == "service/main.py" {
			hasServiceMain = true
		}
	}
	if !hasServiceMain {
		t.Error("expected service/main.py to exist in result")
	}
}

func TestBuildBundle_Empty(t *testing.T) {
	_, err := BuildBundle([]byte{})
	if err == nil {
		t.Fatal("expected error for empty zip")
	}
}

func TestBuildBundle_PathTraversal(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("service/../../../etc/passwd")
	f.Write([]byte("root:x:0:0"))
	w.Close()

	_, err := BuildBundle(buf.Bytes())
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestBuildBundle_ServiceAndChecker(t *testing.T) {
	input := createBundleZip(
		map[string]string{"main.py": "service code"},
		map[string]string{"checker.py": "checker code with exit(101)"},
	)

	result, err := BuildBundle(input)
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}

	found := make(map[string]bool)
	for _, f := range r.File {
		found[f.Name] = true
	}
	if !found["service/main.py"] {
		t.Error("missing service/main.py")
	}
	if !found["checker/checker.py"] {
		t.Error("missing checker/checker.py")
	}
}

func TestExtractMetadata_FromReadme(t *testing.T) {
	bundle := createBundleZip(map[string]string{
		"README.md": "# Awesome Service\n\nThis is a great service for CTF",
		"LICENSE":   "MIT License\n\nCopyright (c) 2024 Test Author",
	}, nil)

	meta, err := ExtractMetadata(bundle)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}

	if meta.Name != "Awesome Service" {
		t.Errorf("Name = %q, want %q", meta.Name, "Awesome Service")
	}
	if !strings.Contains(meta.PublicDescription, "great service") {
		t.Errorf("PublicDescription = %q, should contain 'great service'", meta.PublicDescription)
	}
	if meta.License != "MIT" {
		t.Errorf("License = %q, want %q", meta.License, "MIT")
	}
	if !strings.Contains(meta.Copyright, "Test Author") {
		t.Errorf("Copyright = %q, should contain 'Test Author'", meta.Copyright)
	}
}

func TestExtractMetadata_FromTrainingJSON(t *testing.T) {
	training := map[string]interface{}{
		"display_name": "SuperChecker",
		"description":  "A super checker service",
	}
	trainingJSON, _ := json.Marshal(training)

	bundle := createZipWithBytes(map[string][]byte{
		"service/README.md":              []byte("# Readme Title\nReadme desc"),
		"service/ctf01d-training.json":   trainingJSON,
	})

	meta, err := ExtractMetadata(bundle)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}

	if meta.Name != "SuperChecker" {
		t.Errorf("Name = %q, want %q", meta.Name, "SuperChecker")
	}
	if meta.PublicDescription != "A super checker service" {
		t.Errorf("PublicDescription = %q, want %q", meta.PublicDescription, "A super checker service")
	}
}

func TestExtractMetadata_ApacheLicense(t *testing.T) {
	bundle := createBundleZip(map[string]string{
		"README.md": "# Test",
		"LICENSE":   "Apache License\nVersion 2.0\n\nSome text",
	}, nil)

	meta, err := ExtractMetadata(bundle)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if meta.License != "Apache-2.0" {
		t.Errorf("License = %q, want %q", meta.License, "Apache-2.0")
	}
}

func TestExtractMetadata_BSD3Clause(t *testing.T) {
	bundle := createBundleZip(map[string]string{
		"README.md": "# Test",
		"LICENSE":   "Redistribution and use in source and binary forms\nneither the name of the contributor",
	}, nil)

	meta, err := ExtractMetadata(bundle)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if meta.License != "BSD-3-Clause" {
		t.Errorf("License = %q, want %q", meta.License, "BSD-3-Clause")
	}
}

func TestDetectLicense_All(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"MIT License\nPermission is hereby granted, free of charge", "MIT"},
		{"Apache License Version 2.0", "Apache-2.0"},
		{"Redistribution and use in source and binary forms with conditions", "BSD-2-Clause"},
		{"GNU General Public License version 3", "GPL-3.0"},
		{"GNU General Public License version 2", "GPL-2.0"},
		{"GNU General Public License", "GPL"},
		{"GNU Lesser General Public License version 3", "LGPL-3.0"},
		{"GNU Lesser General Public License", "LGPL"},
		{"Mozilla Public License", "MPL-2.0"},
		{"ISC License", "ISC"},
		{"This is free and unencumbered software released into the public domain", "Unlicense"},
		{"random text with no license", ""},
		{"", ""},
	}
	for _, tt := range tests {
		result := detectLicense(tt.text)
		if result != tt.expected {
			t.Errorf("detectLicense(%q) = %q, want %q", tt.text[:min(30, len(tt.text))], result, tt.expected)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestSafeRelPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo/bar.txt", "foo/bar.txt"},
		{"../etc/passwd", ""},
		{"./foo", ""},
		{"foo/../bar", ""},
		{"", ""},
		{"/leading/slash", "leading/slash"},
		{"trailing/", "trailing"},
	}
	for _, tt := range tests {
		result := safeRelPath(tt.input)
		if result != tt.expected {
			t.Errorf("safeRelPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		url      string
		owner    string
		repo     string
		ref      string
		hasError bool
	}{
		{"https://github.com/owner/repo", "owner", "repo", "", false},
		{"https://github.com/owner/repo/tree/main", "owner", "repo", "main", false},
		{"https://github.com/owner/repo.git", "owner", "repo", "", false},
		{"https://example.com/owner/repo", "", "", "", true},
		{"not-a-url", "", "", "", true},
		{"https://github.com/owner", "", "", "", true},
	}
	for _, tt := range tests {
		owner, repo, ref, err := parseGitHubURL(tt.url)
		if tt.hasError {
			if err == nil {
				t.Errorf("parseGitHubURL(%q) expected error", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseGitHubURL(%q) unexpected error: %v", tt.url, err)
			continue
		}
		if owner != tt.owner || repo != tt.repo || ref != tt.ref {
			t.Errorf("parseGitHubURL(%q) = (%q,%q,%q), want (%q,%q,%q)", tt.url, owner, repo, ref, tt.owner, tt.repo, tt.ref)
		}
	}
}

func TestContainsToken(t *testing.T) {
	tests := []struct {
		data     string
		token    string
		expected bool
	}{
		{"exit(101)", "101", true},
		{"code 101 done", "101", true},
		{"code1010 done", "101", false},
		{"1010 code", "101", false},
		{"the 101 is here", "101", true},
		{"nothing here", "101", false},
		{"", "101", false},
		{"101", "101", true},
	}
	for _, tt := range tests {
		result := containsToken(tt.data, tt.token)
		if result != tt.expected {
			t.Errorf("containsToken(%q, %q) = %v, want %v", tt.data, tt.token, result, tt.expected)
		}
	}
}

func TestInspectCheckerFromBytes_AllCodes(t *testing.T) {
	zipData := createBundleZip(
		map[string]string{"main.py": "service"},
		map[string]string{"checker.py": "exit(101)\nexit(102)\nexit(103)\nexit(104)"},
	)

	result := InspectCheckerFromBytes(zipData)
	if result.Status != "codes" {
		t.Errorf("Status = %q, want %q", result.Status, "codes")
	}
	if len(result.FoundCodes) != 4 {
		t.Errorf("FoundCodes = %v, want 4 codes", result.FoundCodes)
	}
}

func TestInspectCheckerFromBytes_PartialCodes(t *testing.T) {
	zipData := createBundleZip(
		map[string]string{"main.py": "service"},
		map[string]string{"checker.py": "exit(101)\nexit(102)"},
	)

	result := InspectCheckerFromBytes(zipData)
	if result.Status != "present" {
		t.Errorf("Status = %q, want %q", result.Status, "present")
	}
	if len(result.FoundCodes) != 2 {
		t.Errorf("FoundCodes = %v, want 2 codes", result.FoundCodes)
	}
}

func TestInspectCheckerFromBytes_MissingChecker(t *testing.T) {
	zipData := createBundleZip(
		map[string]string{"main.py": "service"},
		nil,
	)

	result := InspectCheckerFromBytes(zipData)
	if result.Status != "missing" {
		t.Errorf("Status = %q, want %q", result.Status, "missing")
	}
}

func TestInspectCheckerFromBytes_TokenBoundary(t *testing.T) {
	zipData := createBundleZip(
		map[string]string{"main.py": "service"},
		map[string]string{"checker.py": "code1010 should not match 101 but standalone 101 should"},
	)

	result := InspectCheckerFromBytes(zipData)
	if result.Status != "present" {
		t.Errorf("Status = %q, want %q", result.Status, "present")
	}
}

func TestCheckerService_CheckChecker_NoArchive(t *testing.T) {
	q := newMockImportQuerier()
	id := int64(1)
	q.services[id] = &db.Service{ID: id, Name: "test", CheckStatus: "unknown"}
	q.byName["test"] = id

	cs := NewCheckerService(q, nil)
	result, err := cs.CheckChecker(context.Background(), id, true)
	if err != nil {
		t.Fatalf("CheckChecker: %v", err)
	}
	if result.CheckStatus != "unknown" {
		t.Errorf("CheckStatus = %q, want %q", result.CheckStatus, "unknown")
	}
}

func TestCheckerService_CheckChecker_NotFound(t *testing.T) {
	q := newMockImportQuerier()
	cs := NewCheckerService(q, nil)
	_, err := cs.CheckChecker(context.Background(), 999, true)
	if err == nil {
		t.Fatal("expected error for not found service")
	}
}

type mockCheckerQuerier struct {
	*mockImportQuerier
}

func (m *mockCheckerQuerier) GetServiceByID(ctx context.Context, id int64) (db.Service, error) {
	return m.mockImportQuerier.GetServiceByID(ctx, id)
}

func (m *mockCheckerQuerier) SetCheckStatus(ctx context.Context, arg db.SetCheckStatusParams) (db.Service, error) {
	return m.mockImportQuerier.SetCheckStatus(ctx, arg)
}

func TestImportFromGithub_BasicFlow(t *testing.T) {
	repoZip := createBundleZipWithSubdir("myrepo", map[string]string{
		"README.md": "# TestService\n\nA test service description",
		"main.py":   "print('hello')",
	}, map[string]string{
		"checker.py": "exit(101)\nexit(102)\nexit(103)\nexit(104)",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Write(repoZip)
	}))
	defer server.Close()

	origCodeloadURL := codeloadURL
	defer func() { codeloadURL = origCodeloadURL }()
	codeloadURL = func(owner, repo, refPath string) string {
		return server.URL
	}

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	result, err := svc.ImportFromGithub(context.Background(), GithubImportRequest{
		RepoURL: "https://github.com/testorg/testrepo",
		Ref:     "main",
	}, true)
	if err != nil {
		t.Fatalf("ImportFromGithub: %v", err)
	}
	if result.Service == nil {
		t.Fatal("Service should not be nil")
	}
	if result.Service.Name != "TestService" {
		t.Errorf("Name = %q, want %q", result.Service.Name, "TestService")
	}
	if result.Service.ServiceLocalPath == nil {
		t.Error("ServiceLocalPath should not be nil")
	}
	if result.Service.CheckerLocalPath == nil {
		t.Error("CheckerLocalPath should not be nil (bundle has checker)")
	}
}

func TestImportFromGithub_InvalidURL(t *testing.T) {
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromGithub(context.Background(), GithubImportRequest{
		RepoURL: "not-a-url",
	}, true)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestImportFromGithub_NameFallback(t *testing.T) {
	repoZip := createFlatRepoZip(map[string]string{
		"README.md": "# RepoName\n\nSome description",
		"main.py":   "code",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(repoZip)
	}))
	defer server.Close()

	origCodeloadURL := codeloadURL
	defer func() { codeloadURL = origCodeloadURL }()
	codeloadURL = func(owner, repo, refPath string) string {
		return server.URL
	}

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	result, err := svc.ImportFromGithub(context.Background(), GithubImportRequest{
		RepoURL: "https://github.com/testorg/myrepo",
		Ref:     "main",
	}, true)
	if err != nil {
		t.Fatalf("ImportFromGithub: %v", err)
	}
	if result.Service.Name != "RepoName" {
		t.Errorf("Name = %q, want %q", result.Service.Name, "RepoName")
	}
}

func TestImportFromZip_BasicFlow(t *testing.T) {
	zipData := createBundleZip(map[string]string{
		"README.md": "# ZipService\n\nImported from zip",
		"main.py":   "print('zip')",
	}, map[string]string{
		"checker.py": "exit(101)\nexit(102)",
	})

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	result, err := svc.ImportFromZip(context.Background(), zipData, true)
	if err != nil {
		t.Fatalf("ImportFromZip: %v", err)
	}
	if result.Service == nil {
		t.Fatal("Service should not be nil")
	}
	if result.Service.Name != "ZipService" {
		t.Errorf("Name = %q, want %q", result.Service.Name, "ZipService")
	}
	if result.Service.ServiceLocalPath == nil {
		t.Error("ServiceLocalPath should not be nil")
	}
}

func TestImportFromZip_Empty(t *testing.T) {
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromZip(context.Background(), []byte{}, true)
	if err == nil {
		t.Fatal("expected error for empty zip")
	}
}

func TestImportFromZipUpload_BasicFlow(t *testing.T) {
	zipData := createBundleZip(map[string]string{
		"README.md": "# UploadService\n\nUploaded",
		"main.py":   "code",
	}, nil)

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	result, err := svc.ImportFromZipUpload(context.Background(), zipData, true)
	if err != nil {
		t.Fatalf("ImportFromZipUpload: %v", err)
	}
	if result.Service.Name != "UploadService" {
		t.Errorf("Name = %q, want %q", result.Service.Name, "UploadService")
	}
}

func TestImportFromZipUpload_Empty(t *testing.T) {
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromZipUpload(context.Background(), []byte{}, true)
	if err == nil {
		t.Fatal("expected error for empty upload")
	}
}

func TestExtractMetadata_NoReadme(t *testing.T) {
	bundle := createBundleZip(map[string]string{
		"main.py": "code",
	}, nil)

	meta, err := ExtractMetadata(bundle)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if meta.Name != "" {
		t.Errorf("Name should be empty when no readme, got %q", meta.Name)
	}
}

func TestBuildBundle_RootFilesPromoted(t *testing.T) {
	input := createBundleZipWithSubdir("myrepo", map[string]string{
		"main.py": "code",
	}, nil)
	r, _ := zip.NewReader(bytes.NewReader(input), int64(len(input)))
	_, rootReadme := readFirstFromZip(r, "myrepo/", readmeCandidates)
	if rootReadme != nil {
		t.Fatal("expected no root readme in this test zip")
	}

	input2 := createZip(map[string]string{
		"myrepo/README.md":    "# Root Readme\nRoot description",
		"myrepo/service/main.py": "code",
	})

	result, err := BuildBundle(input2)
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	r2, _ := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	found := false
	for _, f := range r2.File {
		if f.Name == "service/README.md" {
			found = true
		}
	}
	if !found {
		t.Error("README.md should be promoted to service/README.md")
	}
}

func TestExtractMetadata_TrainingJSONFallback(t *testing.T) {
	training := map[string]interface{}{"display_name": "Fallback"}
	trainingJSON, _ := json.Marshal(training)

	bundle := createZip(map[string]string{
		"service/ctf01d-training.json": string(trainingJSON),
	})

	meta, err := ExtractMetadata(bundle)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if meta.Name != "Fallback" {
		t.Errorf("Name = %q, want %q", meta.Name, "Fallback")
	}
	if meta.Ctf01dTraining == nil {
		t.Error("Ctf01dTraining should not be nil")
	}
}

func TestBuildBundle_SkipsGitDir(t *testing.T) {
	input := createZip(map[string]string{
		"service/main.py":    "code",
		"service/.git/config": "gitconfig",
	})

	result, err := BuildBundle(input)
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	r, _ := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, ".git/") {
			t.Errorf(".git/ entries should be skipped, found: %s", f.Name)
		}
		if strings.HasPrefix(f.Name, "service/.git/") {
			t.Errorf("service/.git/ entries should be skipped, found: %s", f.Name)
		}
	}
}

func TestSummarizeMarkdown(t *testing.T) {
	md := []byte("# Title\n\nSome description here\n\n## Details\n\nMore info")
	result := summarizeMarkdown(md)
	if strings.Contains(result, "#") {
		t.Errorf("headings should be removed, got: %q", result)
	}
	if !strings.Contains(result, "Some description") {
		t.Errorf("should contain description text, got: %q", result)
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"# Hello World\nSome text", "Hello World"},
		{"## Sub Title\nSome text", "Sub Title"},
		{"No heading here", ""},
		{"# [Link Text](http://example.com)", "Link Text"},
		{"# `Code Title`", "Code Title"},
	}
	for _, tt := range tests {
		result := extractTitle([]byte(tt.input))
		if result != tt.expected {
			t.Errorf("extractTitle(%q) = %q, want %q", tt.input[:min(30, len(tt.input))], result, tt.expected)
		}
	}
}

func TestExtractCopyright(t *testing.T) {
 	tests := []struct {
 		input    string
 		contains string
 	}{
 		{"Copyright (c) 2024 Test Author", "2024 Test Author"},
 		{"\u00a9 2024 ACME Inc", "2024 ACME Inc"},
 		{"No license info here", ""},
 	}
 	for _, tt := range tests {
 		result := extractCopyright(tt.input)
 		if tt.contains == "" {
 			if result != "" {
 				t.Errorf("extractCopyright(%q) = %q, want empty", tt.input, result)
 			}
 		} else {
 			if !strings.Contains(result, tt.contains) {
 				t.Errorf("extractCopyright(%q) = %q, want to contain %q", tt.input, result, tt.contains)
 			}
 		}
 	}
}

func TestValidateZipBytes(t *testing.T) {
	if err := validateZipBytes(nil); err == nil {
		t.Error("expected error for nil")
	}
	if err := validateZipBytes([]byte{1, 2, 3}); err == nil {
		t.Error("expected error for short data")
	}
	if err := validateZipBytes([]byte{0x50, 0x4B, 0x03, 0x04}); err != nil {
		t.Errorf("expected no error for valid zip magic, got: %v", err)
	}
}

func TestGithubArchiveURL(t *testing.T) {
	url := githubArchiveURL("owner", "repo", "main")
	expected := "https://github.com/owner/repo/archive/refs/heads/main.zip"
	if url != expected {
		t.Errorf("githubArchiveURL = %q, want %q", url, expected)
	}
}

func TestCodeloadURL(t *testing.T) {
	url := codeloadURL("owner", "repo", "refs/heads/main")
	expected := "https://codeload.github.com/owner/repo/zip/refs/heads/main"
	if url != expected {
		t.Errorf("codeloadURL = %q, want %q", url, expected)
	}
}

func TestComputeSHA256Hex(t *testing.T) {
	result := computeSHA256Hex([]byte("test"))
	if len(result) != 64 {
		t.Errorf("SHA256 hex should be 64 chars, got %d", len(result))
	}
}

func TestBuildBundle_WithCheckerExcludedFromService(t *testing.T) {
	input := createZip(map[string]string{
		"README.md":       "# Service\nDesc",
		"main.py":         "code",
		"checker/check.py": "checker code",
	})

	result, err := BuildBundle(input)
	if err != nil {
		t.Fatalf("BuildBundle: %v", err)
	}

	r, _ := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	hasServiceMain := false
	hasChecker := false
	hasServiceChecker := false
	for _, f := range r.File {
		if f.Name == "service/main.py" {
			hasServiceMain = true
		}
		if f.Name == "checker/check.py" {
			hasChecker = true
		}
		if f.Name == "service/checker/check.py" {
			hasServiceChecker = true
		}
	}
	if !hasServiceMain {
		t.Error("service/main.py should exist")
	}
	if !hasChecker {
		t.Error("checker/check.py should exist")
	}
	if hasServiceChecker {
		t.Error("service/checker/ should NOT exist (checker excluded from service)")
	}
}

func TestLimitedReader(t *testing.T) {
	data := strings.NewReader("hello world")
	lr := &limitedReader{r: data, n: 5}
	buf := make([]byte, 100)
	n, _ := lr.Read(buf)
	if n > 5 {
		t.Errorf("should read at most 5 bytes, got %d", n)
	}
}

func TestIsDigit(t *testing.T) {
	if !isDigit('5') {
		t.Error("5 should be a digit")
	}
	if isDigit('a') {
		t.Error("'a' should not be a digit")
	}
}

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
}

func TestImportFromZip_NoReadmeNoName(t *testing.T) {
	zipData := createBundleZip(map[string]string{
		"main.py": "code",
	}, nil)

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromZip(context.Background(), zipData, true)
	if err == nil {
		t.Fatal("expected error when no name can be determined")
	}
}

func TestCheckerService_WithLocalArchive(t *testing.T) {
	q := newMockImportQuerier()
	id := int64(1)
	path := "services/1/checker.zip"
	q.services[id] = &db.Service{
		ID:               id,
		Name:             "test",
		CheckStatus:      "unknown",
		CheckerLocalPath: &path,
	}
	q.byName["test"] = id

	cs := NewCheckerService(q, nil)

	svc, err := cs.CheckChecker(context.Background(), id, true)
	if err != nil {
		t.Fatalf("CheckChecker: %v", err)
	}
	if svc == nil {
		t.Fatal("service should not be nil")
	}
}

func TestImportFromZip_DuplicateName(t *testing.T) {
	zipData := createBundleZip(map[string]string{
		"README.md": "# DupService\nDesc",
		"main.py":   "code",
	}, nil)

	q := newMockImportQuerier()
	store := newMemStorage()
	q.byName["DupService"] = 1
	q.services[1] = &db.Service{ID: 1, Name: "DupService"}

	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromZip(context.Background(), zipData, true)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestBuildBundle_LargeEntry(t *testing.T) {
	largeContent := strings.Repeat("x", 60*1024*1024)
	input := createZip(map[string]string{
		"service/main.py": largeContent,
	})

	_, err := BuildBundle(input)
	if err == nil {
		t.Fatal("expected error for too large entry")
	}
}

func TestBuildBundle_TooManyFiles(t *testing.T) {
	files := make(map[string]string)
	for i := 0; i < 10001; i++ {
		files[fmt.Sprintf("service/file_%d.txt", i)] = "content"
	}
	input := createZip(files)

	_, err := BuildBundle(input)
	if err == nil {
		t.Fatal("expected error for too many files")
	}
}

func TestExtractCheckerFromBundle(t *testing.T) {
	bundle := createBundleZip(
		map[string]string{"main.py": "service code"},
		map[string]string{"checker.py": "checker code"},
	)

	checkerZip := extractCheckerFromBundle(bundle)
	if checkerZip == nil {
		t.Fatal("checker zip should not be nil")
	}

	r, err := zip.NewReader(bytes.NewReader(checkerZip), int64(len(checkerZip)))
	if err != nil {
		t.Fatalf("reading checker zip: %v", err)
	}

	found := false
	for _, f := range r.File {
		if f.Name == "checker/checker.py" {
			found = true
		}
	}
	if !found {
		t.Error("checker/checker.py should exist in extracted checker zip")
	}
}

func TestExtractCheckerFromBundle_NoChecker(t *testing.T) {
	bundle := createBundleZip(
		map[string]string{"main.py": "service code"},
		nil,
	)

	checkerZip := extractCheckerFromBundle(bundle)
	if checkerZip != nil {
		t.Error("checker zip should be nil when no checker dir")
	}
}

func TestReadEntryFromZip_NotFound(t *testing.T) {
	data := createZip(map[string]string{"other.txt": "content"})
	r, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	result := readEntryFromZip(r, "nonexistent.txt")
	if result != nil {
		t.Error("should return nil for non-existent entry")
	}
}

func TestDetectRootPrefix_Multiple(t *testing.T) {
	data := createZip(map[string]string{
		"foo/file1.txt": "a",
		"bar/file2.txt": "b",
	})
	r, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	prefix := detectRootPrefix(r)
	if prefix != "" {
		t.Errorf("expected empty prefix for multiple roots, got %q", prefix)
	}
}

func TestDetectRootPrefix_Service(t *testing.T) {
	data := createBundleZip(map[string]string{"main.py": "code"}, nil)
	r, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	prefix := detectRootPrefix(r)
	if prefix != "" {
		t.Errorf("expected empty prefix for 'service' root, got %q", prefix)
	}
}

func TestPgtypeTzInImport(t *testing.T) {
	now := time.Now()
	ts := pgtypeTz(now)
	if !ts.Valid {
		t.Fatal("pgtypeTz should return valid timestamptz")
	}
}

// Silence unused import
var _ = io.ReadAll
var _ = pgtype.Timestamptz{}
