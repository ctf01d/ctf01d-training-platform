package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func init() {
	blockedIPCheck = func(net.IP) bool { return false }
}

type mockArchiveQuerier struct {
	services map[int64]*db.Service
	nextID   int64
}

func newMockArchiveQuerier() *mockArchiveQuerier {
	return &mockArchiveQuerier{
		services: make(map[int64]*db.Service),
		nextID:   1,
	}
}

func (m *mockArchiveQuerier) addService(svc db.Service) int64 {
	id := m.nextID
	m.nextID++
	svc.ID = id
	m.services[id] = &svc
	return id
}

func (m *mockArchiveQuerier) GetServiceByID(_ context.Context, id int64) (db.Service, error) {
	svc, ok := m.services[id]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	return *svc, nil
}

func (m *mockArchiveQuerier) SetServiceLocal(_ context.Context, arg db.SetServiceLocalParams) (db.Service, error) {
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

func (m *mockArchiveQuerier) SetCheckerLocal(_ context.Context, arg db.SetCheckerLocalParams) (db.Service, error) {
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

type memStorage struct {
	files map[string][]byte
}

func newMemStorage() *memStorage {
	return &memStorage{files: make(map[string][]byte)}
}

func (s *memStorage) Save(_ context.Context, key string, r io.Reader) (storage.FileInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return storage.FileInfo{}, err
	}
	hash := sha256.Sum256(data)
	s.files[key] = data
	return storage.FileInfo{Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}, nil
}

func (s *memStorage) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	data, ok := s.files[key]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", key)
	}
	return &readSeekCloser{Reader: bytes.NewReader(data)}, nil
}

func (s *memStorage) Delete(_ context.Context, key string) error {
	delete(s.files, key)
	return nil
}

func (s *memStorage) Stat(_ context.Context, key string) (storage.FileInfo, error) {
	data, ok := s.files[key]
	if !ok {
		return storage.FileInfo{}, fmt.Errorf("file not found: %s", key)
	}
	hash := sha256.Sum256(data)
	return storage.FileInfo{Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}, nil
}

type readSeekCloser struct {
	*bytes.Reader
}

func (r *readSeekCloser) Close() error { return nil }

func makeZipData(size int) []byte {
	buf := new(bytes.Buffer)
	buf.Write(zipMagic)
	extra := make([]byte, size)
	for i := range extra {
		extra[i] = byte(i % 256)
	}
	buf.Write(extra)
	return buf.Bytes()
}

func TestRedownload_Success(t *testing.T) {
	zipData := makeZipData(100)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Write(zipData)
	}))
	defer server.Close()

	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{
		Name:              "test-svc",
		ServiceArchiveUrl: strPtr(server.URL + "/service.zip"),
	})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	result, err := arcSvc.Redownload(context.Background(), id, true)
	if err != nil {
		t.Fatalf("Redownload: %v", err)
	}
	if result.ServiceLocalPath == nil {
		t.Fatal("ServiceLocalPath should not be nil")
	}
	if result.ServiceLocalSize == nil || *result.ServiceLocalSize <= 0 {
		t.Fatal("ServiceLocalSize should be > 0")
	}
	if result.ServiceLocalSha256 == nil || *result.ServiceLocalSha256 == "" {
		t.Fatal("ServiceLocalSha256 should not be empty")
	}
}

func TestRedownload_NotZip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not a zip file"))
	}))
	defer server.Close()

	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{
		Name:              "test-svc",
		ServiceArchiveUrl: strPtr(server.URL + "/service.zip"),
	})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, err := arcSvc.Redownload(context.Background(), id, true)
	if err == nil {
		t.Fatal("expected error for non-zip download")
	}
	if !strings.Contains(err.Error(), "not a valid ZIP") {
		t.Errorf("error should mention invalid ZIP, got: %v", err)
	}
}

func TestRedownload_ExceedsSize(t *testing.T) {
	zipData := makeZipData(1000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer server.Close()

	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{
		Name:              "test-svc",
		ServiceArchiveUrl: strPtr(server.URL + "/service.zip"),
	})

	arcSvc := NewArchiveService(q, store, 100)
	_, err := arcSvc.Redownload(context.Background(), id, true)
	if err == nil {
		t.Fatal("expected error for exceeding size")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("error should mention size limit, got: %v", err)
	}
}

func TestRedownload_BothArchives(t *testing.T) {
	zipData := makeZipData(50)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer server.Close()

	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{
		Name:              "test-svc",
		ServiceArchiveUrl: strPtr(server.URL + "/service.zip"),
		CheckerArchiveUrl: strPtr(server.URL + "/checker.zip"),
	})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	result, err := arcSvc.Redownload(context.Background(), id, true)
	if err != nil {
		t.Fatalf("Redownload: %v", err)
	}
	if result.ServiceLocalPath == nil {
		t.Fatal("ServiceLocalPath should not be nil")
	}
	if result.CheckerLocalPath == nil {
		t.Fatal("CheckerLocalPath should not be nil")
	}
}

func TestRedownload_NoURLs(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{
		Name: "test-svc",
	})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	result, err := arcSvc.Redownload(context.Background(), id, true)
	if err != nil {
		t.Fatalf("Redownload: %v", err)
	}
	if result.ServiceLocalPath != nil {
		t.Error("ServiceLocalPath should be nil when no URLs")
	}
}

func TestUploadArchives_Success(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{Name: "test-svc"})

	zipData := makeZipData(100)

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	result, err := arcSvc.UploadArchives(context.Background(), id, bytes.NewReader(zipData), nil, true)
	if err != nil {
		t.Fatalf("UploadArchives: %v", err)
	}
	if result.ServiceLocalPath == nil {
		t.Fatal("ServiceLocalPath should not be nil")
	}
	if _, ok := store.files["services/1/service.zip"]; !ok {
		t.Fatal("file should be saved in storage")
	}
}

func TestUploadArchives_BothFiles(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{Name: "test-svc"})

	serviceZip := makeZipData(50)
	checkerZip := makeZipData(30)

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	result, err := arcSvc.UploadArchives(context.Background(), id, bytes.NewReader(serviceZip), bytes.NewReader(checkerZip), true)
	if err != nil {
		t.Fatalf("UploadArchives: %v", err)
	}
	if result.ServiceLocalPath == nil {
		t.Fatal("ServiceLocalPath should not be nil")
	}
	if result.CheckerLocalPath == nil {
		t.Fatal("CheckerLocalPath should not be nil")
	}
}

func TestUploadArchives_NotZip(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{Name: "test-svc"})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, err := arcSvc.UploadArchives(context.Background(), id, bytes.NewReader([]byte("not a zip")), nil, true)
	if err == nil {
		t.Fatal("expected error for non-zip upload")
	}
}

func TestUploadArchives_ExceedsSize(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{Name: "test-svc"})

	zipData := makeZipData(1000)

	arcSvc := NewArchiveService(q, store, 100)
	_, err := arcSvc.UploadArchives(context.Background(), id, bytes.NewReader(zipData), nil, true)
	if err == nil {
		t.Fatal("expected error for exceeding size")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("error should mention size limit, got: %v", err)
	}
}

func TestOpenLocal_Service(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()

	zipData := makeZipData(50)
	ctx := context.Background()
	store.Save(ctx, "services/1/service.zip", bytes.NewReader(zipData))

	path := "services/1/service.zip"
	q.addService(db.Service{
		Name:             "test-svc",
		ServiceLocalPath: &path,
	})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	rc, filename, err := arcSvc.OpenLocal(ctx, 1, "service")
	if err != nil {
		t.Fatalf("OpenLocal: %v", err)
	}
	defer rc.Close()

	if filename != "test-svc-service.zip" {
		t.Errorf("filename = %q, want %q", filename, "test-svc-service.zip")
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(data) != len(zipData) {
		t.Errorf("data length = %d, want %d", len(data), len(zipData))
	}
}

func TestOpenLocal_Checker(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()

	zipData := makeZipData(50)
	ctx := context.Background()
	store.Save(ctx, "services/1/checker.zip", bytes.NewReader(zipData))

	path := "services/1/checker.zip"
	q.addService(db.Service{
		Name:             "test-svc",
		CheckerLocalPath: &path,
	})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	rc, filename, err := arcSvc.OpenLocal(ctx, 1, "checker")
	if err != nil {
		t.Fatalf("OpenLocal: %v", err)
	}
	defer rc.Close()

	if filename != "test-svc-checker.zip" {
		t.Errorf("filename = %q, want %q", filename, "test-svc-checker.zip")
	}
}

func TestOpenLocal_NoArchive(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	q.addService(db.Service{Name: "test-svc"})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, _, err := arcSvc.OpenLocal(context.Background(), 1, "service")
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOpenLocal_InvalidKind(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	q.addService(db.Service{Name: "test-svc"})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, _, err := arcSvc.OpenLocal(context.Background(), 1, "invalid")
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestOpenLocal_NotFound(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, _, err := arcSvc.OpenLocal(context.Background(), 999, "service")
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRedownload_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	q := newMockArchiveQuerier()
	store := newMemStorage()
	id := q.addService(db.Service{
		Name:              "test-svc",
		ServiceArchiveUrl: strPtr(server.URL + "/service.zip"),
	})

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, err := arcSvc.Redownload(context.Background(), id, true)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRedownload_ServiceNotFound(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, err := arcSvc.Redownload(context.Background(), 999, true)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUploadArchives_NotFound(t *testing.T) {
	q := newMockArchiveQuerier()
	store := newMemStorage()
	zipData := makeZipData(50)

	arcSvc := NewArchiveService(q, store, 10*1024*1024)
	_, err := arcSvc.UploadArchives(context.Background(), 999, bytes.NewReader(zipData), nil, true)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPgtypeTz(t *testing.T) {
	now := time.Now()
	ts := pgtypeTz(now)
	if !ts.Valid {
		t.Fatal("pgtypeTz should return valid timestamptz")
	}
	if !ts.Time.Equal(now) {
		t.Errorf("Time = %v, want %v", ts.Time, now)
	}

	var zeroPgtype pgtype.Timestamptz = pgtype.Timestamptz{}
	_ = zeroPgtype
}

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip string
		ok bool
	}{
		{"127.0.0.1", false},
		{"::1", false},
		{"10.0.0.1", false},
		{"192.168.1.1", false},
		{"172.16.0.1", false},
		{"169.254.169.254", false},
		{"0.0.0.1", false},
		{"fd00::1", false},
		{"fc00::1", false},
		{"fe80::1", false},
		{"::", false},
		{"100.64.0.1", false},
		{"100.127.255.254", false},
		{"198.18.0.1", false},
		{"198.19.255.254", false},
		{"192.0.2.1", false},
		{"198.51.100.1", false},
		{"203.0.113.1", false},
		{"2001:db8::1", false},
		{"224.0.0.1", false},
		{"239.255.255.255", false},
		{"ff02::1", false},
		{"240.0.0.1", false},
		{"255.255.255.255", false},
		{"192.0.0.1", false},
		{"192.88.99.1", false},
		{"64:ff9b:1::1", false},
		{"100::1", false},
		{"100:0:0:1::1", false},
		{"2001:2::1", false},
		{"2002:c0a8:101::", false},
		{"3fff::1", false},
		{"5f00::1", false},
		{"93.184.216.34", true},
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2606:4700:4700::1111", true},
		{"2001:4860:4860::8888", true},
	}
	for _, tc := range cases {
		blocked := isBlockedIP(net.ParseIP(tc.ip))
		if tc.ok && blocked {
			t.Errorf("isBlockedIP(%q): expected allowed, got blocked", tc.ip)
		}
		if !tc.ok && !blocked {
			t.Errorf("isBlockedIP(%q): expected blocked, got allowed", tc.ip)
		}
	}
}
