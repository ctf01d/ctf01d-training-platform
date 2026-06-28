package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

func createZip(files map[string]string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			panic(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			panic(err)
		}
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func createZipWithBytes(files map[string][]byte) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			panic(err)
		}
		if _, err := f.Write(content); err != nil {
			panic(err)
		}
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
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
		files[subdir+"/service/"+k] = v
	}
	for k, v := range checkerFiles {
		files[subdir+"/checker/"+k] = v
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

func createSourceImportZip(rootDir, serviceID, displayName, description string, training map[string]any) []byte {
	if rootDir == "" {
		rootDir = "repo"
	}
	if displayName == "" {
		displayName = serviceID
	}

	trainingData := make(map[string]any, len(training)+2)
	for key, value := range training {
		trainingData[key] = value
	}
	if _, ok := trainingData["display_name"]; !ok && displayName != "" {
		trainingData["display_name"] = displayName
	}
	if _, ok := trainingData["description"]; !ok && description != "" {
		trainingData["description"] = description
	}

	files := map[string]string{
		rootDir + "/README.md":                            fmt.Sprintf("# %s\n\n%s", displayName, description),
		rootDir + "/.ctf01d-service.yml":                  fmt.Sprintf("checker-config-v0.5.2:\n  id: %s\n  service_name: %s\n  script_path: ./checker.py\n", serviceID, displayName),
		rootDir + "/vuln-service/docker-compose.yml":      "services: {}\n",
		rootDir + "/vuln-service/app.py":                  "print('service')\n",
		rootDir + "/checker_" + serviceID + "/checker.py": "exit(101)\nexit(102)\nexit(103)\nexit(104)\n",
		rootDir + "/writeups/README.md":                   "writeup\n",
		rootDir + "/exploits/poc.py":                      "exploit\n",
	}
	if len(trainingData) > 0 {
		trainingJSON, _ := json.Marshal(trainingData)
		files[rootDir+"/ctf01d-training.json"] = string(trainingJSON)
	}

	return createZip(files)
}

func importStrPtr(s string) *string {
	return &s
}

func createTestGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	repoDir := t.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(repoDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", fullPath, err)
		}
	}

	run := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, strings.TrimSpace(string(out)))
		}
	}

	run("init", "-b", "main")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")
	run("add", ".")
	run("commit", "-m", "initial")

	return repoDir
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
		return db.Service{}, &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	}
	id := m.nextID
	m.nextID++
	now := time.Now()
	svc := db.Service{
		ID:                id,
		Name:              arg.Name,
		PublicDescription: arg.PublicDescription,
		Author:            arg.Author,
		Copyright:         arg.Copyright,
		Public:            arg.Public,
		ServiceArchiveUrl: arg.ServiceArchiveUrl,
		Ctf01dTraining:    arg.Ctf01dTraining,
		CheckStatus:       arg.CheckStatus,
		SourceKind:        arg.SourceKind,
		GitRepoUrl:        arg.GitRepoUrl,
		GitRef:            arg.GitRef,
		GitSubdir:         arg.GitSubdir,
		GitSyncStatus:     arg.GitSyncStatus,
		CreatedAt:         now,
		UpdatedAt:         now,
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

func (m *mockImportQuerier) ApplyServiceImportMetadata(_ context.Context, arg db.ApplyServiceImportMetadataParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	if currentID, exists := m.byName[svc.Name]; exists && currentID == arg.ID {
		delete(m.byName, svc.Name)
	}
	svc.Name = arg.Name
	svc.PublicDescription = arg.PublicDescription
	svc.Author = arg.Author
	svc.Copyright = arg.Copyright
	svc.Ctf01dTraining = arg.Ctf01dTraining
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

func (m *mockImportQuerier) SetGitSource(_ context.Context, arg db.SetGitSourceParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.SourceKind = arg.SourceKind
	svc.GitRepoUrl = arg.GitRepoUrl
	svc.GitRef = arg.GitRef
	svc.GitSubdir = arg.GitSubdir
	svc.GitLastCommit = nil
	svc.GitSyncStatus = arg.GitSyncStatus
	svc.GitSyncError = nil
	svc.GitSyncedAt = pgtype.Timestamptz{}
	return *svc, nil
}

func (m *mockImportQuerier) SetGitSyncState(_ context.Context, arg db.SetGitSyncStateParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.GitLastCommit = arg.GitLastCommit
	svc.GitSyncedAt = arg.GitSyncedAt
	svc.GitSyncStatus = arg.GitSyncStatus
	svc.GitSyncError = arg.GitSyncError
	return *svc, nil
}

type fakeGitFetcher struct {
	fetched *fetchedGitRepo
	err     error
}

func (f fakeGitFetcher) Fetch(context.Context, GitImportRequest) (*fetchedGitRepo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.fetched, nil
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
	f, err := w.Create("service/../../../etc/passwd")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := f.Write([]byte("root:x:0:0")); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	_, err = BuildBundle(buf.Bytes())
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

func TestBuildBundle_CybersibirLayout(t *testing.T) {
	input := createZip(map[string]string{
		"2027-cybersibir-service-bank/README.md":                       "# Bank\nDesc",
		"2027-cybersibir-service-bank/vuln-service/docker-compose.yml": "services: {}",
		"2027-cybersibir-service-bank/vuln-service/app.py":             "print('service')",
		"2027-cybersibir-service-bank/checker_bank/checker.py":         "exit(101)",
		"2027-cybersibir-service-bank/writeups/README.md":              "writeup",
		"2027-cybersibir-service-bank/exploits/poc.py":                 "exploit",
		"2027-cybersibir-service-bank/vuln-service_dev/build.sh":       "build",
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
	if !found["service/app.py"] {
		t.Error("missing service/app.py")
	}
	if !found["checker/checker.py"] {
		t.Error("missing checker/checker.py")
	}
	if !found["writeups/README.md"] {
		t.Error("missing writeups/README.md")
	}
	if !found["exploits/poc.py"] {
		t.Error("missing exploits/poc.py")
	}
	if found["service/vuln-service_dev/build.sh"] {
		t.Error("vuln-service_dev should not be copied into service/")
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
		"service/README.md":            []byte("# Readme Title\nReadme desc"),
		"service/ctf01d-training.json": trainingJSON,
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

func TestExtractMetadata_AuthorAndCopyrightFromTrainingJSON(t *testing.T) {
	training := map[string]interface{}{
		"display_name": "VaultNotes",
		"description":  "A trading service",
		"author":       "IgorPolyakov (@hotorcelexo)",
		"year":         2026,
	}
	trainingJSON, _ := json.Marshal(training)

	bundle := createZipWithBytes(map[string][]byte{
		"service/ctf01d-training.json": trainingJSON,
	})

	meta, err := ExtractMetadata(bundle)
	if err != nil {
		t.Fatalf("ExtractMetadata: %v", err)
	}
	if meta.Author != "IgorPolyakov (@hotorcelexo)" {
		t.Errorf("Author = %q, want %q", meta.Author, "IgorPolyakov (@hotorcelexo)")
	}
	// No LICENSE present, so copyright is derived from author + year.
	if meta.Copyright != "© 2026 IgorPolyakov (@hotorcelexo)" {
		t.Errorf("Copyright = %q, want %q", meta.Copyright, "© 2026 IgorPolyakov (@hotorcelexo)")
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
			t.Errorf("detectLicense(%q) = %q, want %q", tt.text[:minInt(30, len(tt.text))], result, tt.expected)
		}
	}
}

func minInt(a, b int) int {
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

func TestParseGitRepoURL(t *testing.T) {
	tests := []struct {
		url      string
		host     string
		owner    string
		repo     string
		ref      string
		hasError bool
	}{
		{"https://github.com/owner/repo", "github.com", "owner", "repo", "", false},
		{"https://github.com/owner/repo#main", "github.com", "owner", "repo", "main", false},
		{"git@github.com:owner/repo.git", "github.com", "owner", "repo", "", false},
		{"ssh://git@gitlab.example.com/group/sub/repo.git", "gitlab.example.com", "group/sub", "repo", "", false},
		{"not-a-url", "", "", "", "", true},
		{"https://github.com/owner", "", "", "", "", true},
	}
	for _, tt := range tests {
		ref, err := parseGitRepoURL(tt.url)
		if tt.hasError {
			if err == nil {
				t.Errorf("parseGitRepoURL(%q) expected error", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseGitRepoURL(%q) unexpected error: %v", tt.url, err)
			continue
		}
		if ref.Host != tt.host || ref.Owner != tt.owner || ref.Repo != tt.repo || ref.Ref != tt.ref {
			t.Errorf("parseGitRepoURL(%q) = (%q,%q,%q,%q), want (%q,%q,%q,%q)", tt.url, ref.Host, ref.Owner, ref.Repo, ref.Ref, tt.host, tt.owner, tt.repo, tt.ref)
		}
	}
}

func TestParseGitRepoURL_LocalRelativeDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	workdir := t.TempDir()
	if err := os.Mkdir(filepath.Join(workdir, "myrepo"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("Chdir(%q): %v", workdir, err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	ref, err := parseGitRepoURL("myrepo")
	if err != nil {
		t.Fatalf("parseGitRepoURL(local dir): %v", err)
	}
	if ref.Host != "local" {
		t.Fatalf("Host = %q, want local", ref.Host)
	}
	if ref.Repo != "myrepo" {
		t.Fatalf("Repo = %q, want myrepo", ref.Repo)
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

func TestImportFromGit_BasicFlow(t *testing.T) {
	repoZip := createZip(map[string]string{
		"2026-cybersibir-service-bank/README.md":                       "# Bank\n\nA test service description",
		"2026-cybersibir-service-bank/.ctf01d-service.yml":             "checker-config-v0.5.2:\n  id: bank\n  service_name: Bank\n  script_path: ./checker.py\n  script_wait_in_sec: 10\n  time_sleep_between_run_scripts_in_sec: 30\n  enabled: true\n",
		"2026-cybersibir-service-bank/vuln-service/docker-compose.yml": "services: {}\n",
		"2026-cybersibir-service-bank/vuln-service/app.py":             "print('hello')",
		"2026-cybersibir-service-bank/checker_bank/checker.py":         "exit(101)\nexit(102)\nexit(103)\nexit(104)",
		"2026-cybersibir-service-bank/writeups/README.md":              "writeup",
		"2026-cybersibir-service-bank/exploits/poc.py":                 "exploit",
	})
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)
	svc.gitFetcher = fakeGitFetcher{
		fetched: &fetchedGitRepo{
			ZipBytes: repoZip,
			Commit:   strings.Repeat("a", 40),
			Ref:      "main",
			RepoURL:  "git@github.com:test/2026-cybersibir-service-bank.git",
			Source: importSourceInfo{
				Source: sourceGit,
				Host:   "github.com",
				Owner:  "test",
				Repo:   "2026-cybersibir-service-bank",
				Path:   "test/2026-cybersibir-service-bank",
			},
		},
	}

	result, err := svc.ImportFromGit(context.Background(), GitImportRequest{
		RepoURL: "git@github.com:test/2026-cybersibir-service-bank.git",
		Ref:     "main",
	}, true)
	if err != nil {
		t.Fatalf("ImportFromGit: %v", err)
	}
	if result.Service == nil {
		t.Fatal("Service should not be nil")
	}
	if result.Service.Name != "bank" {
		t.Errorf("Name = %q, want %q", result.Service.Name, "bank")
	}
	if result.Service.ServiceLocalPath == nil {
		t.Error("ServiceLocalPath should not be nil")
	}
	if result.Service.CheckerLocalPath == nil {
		t.Error("CheckerLocalPath should not be nil (bundle has checker)")
	}
	if result.Service.Source.Kind != sourceGit {
		t.Errorf("Source.Kind = %q, want %q", result.Service.Source.Kind, sourceGit)
	}
}

func TestImportFromGit_InvalidURL(t *testing.T) {
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromGit(context.Background(), GitImportRequest{
		RepoURL: "not-a-url",
	}, true)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestImportFromGit_RequiresAdmin(t *testing.T) {
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromGit(context.Background(), GitImportRequest{
		RepoURL: "ssh://git@example.com/team/repo.git",
	}, false)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestImportFromGit_UsesManifestID(t *testing.T) {
	repoZip := createZip(map[string]string{
		"repo/README.md":                       "# Fancy Repo\n\nSome description",
		"repo/.ctf01d-service.yml":             "checker-config-v0.5.2:\n  id: easyas\n  service_name: EasyAs\n  script_path: ./checker.py\n",
		"repo/vuln-service/docker-compose.yml": "services: {}\n",
		"repo/vuln-service/app.py":             "code",
		"repo/checker_easyas/checker.py":       "exit(101)",
		"repo/writeups/README.md":              "writeup",
		"repo/exploits/poc.py":                 "exploit",
	})
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)
	svc.gitFetcher = fakeGitFetcher{
		fetched: &fetchedGitRepo{
			ZipBytes: repoZip,
			Commit:   strings.Repeat("b", 40),
			Ref:      "main",
			RepoURL:  "ssh://git@example.com/team/repo.git",
			Source: importSourceInfo{
				Source: sourceGit,
				Host:   "example.com",
				Owner:  "team",
				Repo:   "repo",
				Path:   "team/repo",
			},
		},
	}

	result, err := svc.ImportFromGit(context.Background(), GitImportRequest{
		RepoURL: "ssh://git@example.com/team/repo.git",
		Ref:     "main",
	}, true)
	if err != nil {
		t.Fatalf("ImportFromGit: %v", err)
	}
	if result.Service.Name != "easyas" {
		t.Errorf("Name = %q, want %q", result.Service.Name, "easyas")
	}
}

func TestImportFromZip_BasicFlow(t *testing.T) {
	zipData := createSourceImportZip("repo", "ZipService", "ZipService", "Imported from zip", nil)

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

func TestImportFromZip_SetsAuthorFromTraining(t *testing.T) {
	training := map[string]any{
		"display_name": "AuthSvc",
		"description":  "A service with an author",
		"author":       "Jane Doe",
		"year":         2026,
	}
	zipData := createSourceImportZip("repo", "AuthSvc", "AuthSvc", "Desc", training)

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	result, err := svc.ImportFromZip(context.Background(), zipData, true)
	if err != nil {
		t.Fatalf("ImportFromZip: %v", err)
	}
	if result.Service.Author == nil || *result.Service.Author != "Jane Doe" {
		t.Errorf("Author = %v, want %q", result.Service.Author, "Jane Doe")
	}
	if result.Service.Copyright == nil || *result.Service.Copyright != "© 2026 Jane Doe" {
		t.Errorf("Copyright = %v, want %q", result.Service.Copyright, "© 2026 Jane Doe")
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

func TestImportFromZip_ValidatesPreview(t *testing.T) {
	zipData := createBundleZip(map[string]string{
		"README.md": "# ZipService\n\nImported from zip",
		"main.py":   "print('zip')",
	}, map[string]string{
		"checker.py": "exit(101)\nexit(102)",
	})

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.ImportFromZip(context.Background(), zipData, true)
	if err == nil {
		t.Fatal("expected validation error for legacy bundle zip")
	}
}

func TestImportFromZipUpload_BasicFlow(t *testing.T) {
	zipData := createSourceImportZip("repo", "UploadService", "UploadService", "Uploaded", nil)

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

func TestPreviewFromZipUpload_CybersibirLayout(t *testing.T) {
	zipData := createZip(map[string]string{
		"2027-cybersibir-service-bank/README.md":                       "# Bank\nDesc",
		"2027-cybersibir-service-bank/vuln-service/docker-compose.yml": "services: {}",
		"2027-cybersibir-service-bank/vuln-service/app.py":             "print('service')",
		"2027-cybersibir-service-bank/checker_bank/checker.py":         "exit(101)",
		"2027-cybersibir-service-bank/writeups/README.md":              "writeup",
		"2027-cybersibir-service-bank/exploits/poc.py":                 "exploit",
		"2027-cybersibir-service-bank/vuln-service_dev/build.sh":       "build",
	})

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	preview, err := svc.PreviewFromZipUpload(context.Background(), zipData, true)
	if err != nil {
		t.Fatalf("PreviewFromZipUpload: %v", err)
	}
	if !preview.Valid {
		t.Fatalf("preview should be valid, got requirements: %#v", preview.Requirements)
	}
	if preview.ServiceName != "bank" {
		t.Errorf("ServiceName = %q, want %q", preview.ServiceName, "bank")
	}
	if preview.ExpectedRepositoryName != "YYYY-cybersibir-service-<service-id>" {
		t.Errorf("ExpectedRepositoryName = %q", preview.ExpectedRepositoryName)
	}
	if preview.CheckerDirectory != "checker_bank" {
		t.Errorf("CheckerDirectory = %q, want %q", preview.CheckerDirectory, "checker_bank")
	}
}

func TestPreviewFromGit_AcceptsGenericHost(t *testing.T) {
	repoZip := createZip(map[string]string{
		"2026-cybersibir-service-vault-notes/README.md":                       "# Vault Notes\nDesc",
		"2026-cybersibir-service-vault-notes/.ctf01d-service.yml":             "checker-config-v0.5.2:\n  id: vaultnotes\n  service_name: Vault Notes\n  script_path: ./checker.py\n",
		"2026-cybersibir-service-vault-notes/vuln-service/docker-compose.yml": "services: {}",
		"2026-cybersibir-service-vault-notes/vuln-service/app.py":             "print('service')",
		"2026-cybersibir-service-vault-notes/checker_vaultnotes/checker.py":   "exit(101)",
		"2026-cybersibir-service-vault-notes/writeups/README.md":              "writeup",
		"2026-cybersibir-service-vault-notes/exploits/poc.py":                 "exploit",
	})

	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)
	svc.gitFetcher = fakeGitFetcher{
		fetched: &fetchedGitRepo{
			ZipBytes: repoZip,
			Commit:   strings.Repeat("c", 40),
			Ref:      "main",
			RepoURL:  "ssh://git@gitlab.example.com/group/2026-cybersibir-service-vault-notes.git",
			Source: importSourceInfo{
				Source: sourceGit,
				Host:   "gitlab.example.com",
				Owner:  "group",
				Repo:   "2026-cybersibir-service-vault-notes",
				Path:   "group/2026-cybersibir-service-vault-notes",
			},
		},
	}

	preview, err := svc.PreviewFromGit(context.Background(), GitImportRequest{
		RepoURL: "ssh://git@gitlab.example.com/group/2026-cybersibir-service-vault-notes.git",
		Ref:     "main",
	}, true)
	if err != nil {
		t.Fatalf("PreviewFromGit: %v", err)
	}
	if !preview.Valid {
		t.Fatalf("preview should be valid, got requirements: %#v", preview.Requirements)
	}
	if preview.ServiceName != "vaultnotes" {
		t.Errorf("ServiceName = %q, want %q", preview.ServiceName, "vaultnotes")
	}
	if preview.ExpectedRepositoryName != "YYYY-cybersibir-service-<service-id>" {
		t.Errorf("ExpectedRepositoryName = %q", preview.ExpectedRepositoryName)
	}
}

func TestPreviewFromGit_RequiresAdmin(t *testing.T) {
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.PreviewFromGit(context.Background(), GitImportRequest{
		RepoURL: "ssh://git@example.com/team/repo.git",
	}, false)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
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
	rootReadme := readFirstFromZip(r, "myrepo/", readmeCandidates)
	if rootReadme != nil {
		t.Fatal("expected no root readme in this test zip")
	}

	input2 := createZip(map[string]string{
		"myrepo/README.md":       "# Root Readme\nRoot description",
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
		"service/main.py":     "code",
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

func TestSummarizeMarkdown_UTF8(t *testing.T) {
	md := []byte("# Заголовок\n\n" + strings.Repeat("ёж", summaryMaxLength))
	result := summarizeMarkdown(md)
	if !utf8.ValidString(result) {
		t.Fatalf("summary should remain valid UTF-8, got %q", result)
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

func TestExtractCopyright_UTF8(t *testing.T) {
	input := "Copyright © " + strings.Repeat("ёж", fieldValueMaxChars)
	result := extractCopyright(input)
	if !utf8.ValidString(result) {
		t.Fatalf("copyright should remain valid UTF-8, got %q", result)
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

func TestExecGitArchiveFetcher_FetchWithSubdir(t *testing.T) {
	repoDir := createTestGitRepo(t, map[string]string{
		"nested/README.md":                       "# Repo\n",
		"nested/.ctf01d-service.yml":             "checker-config-v0.5.2:\n  id: nestedsvc\n  service_name: Nested Service\n  script_path: ./checker.py\n",
		"nested/vuln-service/docker-compose.yml": "services: {}\n",
		"nested/vuln-service/app.py":             "print('svc')\n",
		"nested/checker_nestedsvc/checker.py":    "exit(101)\n",
		"nested/writeups/README.md":              "writeup\n",
		"nested/exploits/poc.py":                 "exploit\n",
	})

	fetcher := newExecGitArchiveFetcher(50 * 1024 * 1024)
	result, err := fetcher.Fetch(context.Background(), GitImportRequest{
		RepoURL: repoDir,
		Ref:     "main",
		Subdir:  "nested",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if result.Ref != "main" {
		t.Errorf("Ref = %q, want %q", result.Ref, "main")
	}
	if result.Commit == "" {
		t.Fatal("Commit should not be empty")
	}
	if result.Source.Repo != filepath.Base(repoDir) {
		t.Errorf("Repo = %q, want %q", result.Source.Repo, filepath.Base(repoDir))
	}
	if err := validateZipBytes(result.ZipBytes); err != nil {
		t.Fatalf("validateZipBytes: %v", err)
	}
}

func TestSyncFromGit_RequiresAdmin(t *testing.T) {
	q := newMockImportQuerier()
	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)

	_, err := svc.SyncFromGit(context.Background(), 1, false)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSyncFromGit_PreservesUnsetRef(t *testing.T) {
	repoZip := createSourceImportZip("2026-cybersibir-service-bank", "bank", "Bank", "Updated description", map[string]any{
		"display_name": "Bank",
		"description":  "Updated description",
		"author":       "Bob",
	})

	q := newMockImportQuerier()
	q.services[1] = &db.Service{
		ID:            1,
		Name:          "bank",
		CheckStatus:   checkStatusUnknown,
		SourceKind:    sourceGit,
		GitRepoUrl:    importStrPtr("ssh://git@example.com/team/repo.git"),
		GitSyncStatus: syncStatusUnknown,
	}
	q.byName["bank"] = 1

	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)
	svc.gitFetcher = fakeGitFetcher{
		fetched: &fetchedGitRepo{
			ZipBytes: repoZip,
			Commit:   strings.Repeat("d", 40),
			Ref:      "main",
			RepoURL:  "ssh://git@example.com/team/repo.git",
			Source: importSourceInfo{
				Source: sourceGit,
				Host:   "example.com",
				Owner:  "team",
				Repo:   "repo",
				Path:   "team/repo",
			},
		},
	}

	result, err := svc.SyncFromGit(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("SyncFromGit: %v", err)
	}
	if result.Source.Ref != nil {
		t.Fatalf("Source.Ref = %v, want nil", result.Source.Ref)
	}

	current, err := q.GetServiceByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetServiceByID: %v", err)
	}
	if current.GitRef != nil {
		t.Fatalf("stored GitRef = %v, want nil", current.GitRef)
	}
}

func TestSyncFromGit_RejectsLegacyRepositoryLayout(t *testing.T) {
	repoZip := createZip(map[string]string{
		"repo/README.md":            "# EasyAs\n\nLegacy service description",
		"repo/ctf01d-training.json": `{"display_name":"EasyAs","description":"Legacy service description","author":"SibirCTF"}`,
		"repo/easyas.py":            "print('legacy')\n",
	})

	q := newMockImportQuerier()
	q.services[1] = &db.Service{
		ID:            1,
		Name:          "EasyAs",
		CheckStatus:   checkStatusUnknown,
		SourceKind:    sourceGit,
		GitRepoUrl:    importStrPtr("https://github.com/SibirCTF/2015-easyas.git"),
		GitRef:        importStrPtr("master"),
		GitSyncStatus: syncStatusUnknown,
	}
	q.byName["EasyAs"] = 1

	store := newMemStorage()
	svc := NewImportService(q, store, 50*1024*1024)
	svc.gitFetcher = fakeGitFetcher{
		fetched: &fetchedGitRepo{
			ZipBytes: repoZip,
			Commit:   strings.Repeat("e", 40),
			Ref:      "master",
			RepoURL:  "https://github.com/SibirCTF/2015-easyas.git",
			Source: importSourceInfo{
				Source: sourceGit,
				Host:   "github.com",
				Owner:  "SibirCTF",
				Repo:   "2015-easyas",
				Path:   "SibirCTF/2015-easyas",
			},
		},
	}

	_, err := svc.SyncFromGit(context.Background(), 1, true)
	if err == nil {
		t.Fatal("expected validation error for legacy repository layout")
	}

	current, getErr := q.GetServiceByID(context.Background(), 1)
	if getErr != nil {
		t.Fatalf("GetServiceByID: %v", getErr)
	}
	if current.GitSyncStatus != syncStatusFailed {
		t.Fatalf("GitSyncStatus = %q, want %q", current.GitSyncStatus, syncStatusFailed)
	}
	if current.GitSyncError == nil {
		t.Fatal("GitSyncError = nil, want layout validation details")
	}
	if !strings.Contains(*current.GitSyncError, "vuln-service/ directory is required") {
		t.Fatalf("GitSyncError = %q, want layout validation details", *current.GitSyncError)
	}
}

func TestParseServiceManifest_DuplicateCheckerSections(t *testing.T) {
	_, err := parseServiceManifest([]byte(`
checker-config-a:
  id: first
checker-config-b:
  id: second
`))
	if err == nil {
		t.Fatal("expected duplicate checker-config error")
	}
}

func TestZipDirectory_RespectsMaxArchiveBytesDuringWalk(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "data.txt"), []byte(strings.Repeat("a", 1024)), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := zipDirectory(rootDir, "repo", 512)
	if err == nil {
		t.Fatal("expected repository size error")
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
		"README.md":        "# Service\nDesc",
		"main.py":          "code",
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
	zipData := createSourceImportZip("repo", "DupService", "DupService", "Desc", nil)

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
var (
	_ = io.ReadAll
	_ = pgtype.Timestamptz{}
)
