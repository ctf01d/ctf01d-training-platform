package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type mockQuerier struct {
	services map[int64]db.Service
	byName   map[string]int64
	nextID   int64
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		services: make(map[int64]db.Service),
		byName:   make(map[string]int64),
		nextID:   1,
	}
}

func mustCreateService(t *testing.T, svc *Service, params CreateParams) *ServiceModel {
	t.Helper()
	result, err := svc.Create(context.Background(), params, true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return result
}

func (m *mockQuerier) CreateService(_ context.Context, arg db.CreateServiceParams) (db.Service, error) {
	if _, exists := m.byName[arg.Name]; exists {
		return db.Service{}, &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	}
	id := m.nextID
	m.nextID++
	now := time.Now()
	svc := db.Service{
		ID:                 id,
		Name:               arg.Name,
		PublicDescription:  arg.PublicDescription,
		PrivateDescription: arg.PrivateDescription,
		Author:             arg.Author,
		Copyright:          arg.Copyright,
		AvatarUrl:          arg.AvatarUrl,
		Public:             arg.Public,
		ServiceArchiveUrl:  arg.ServiceArchiveUrl,
		CheckerArchiveUrl:  arg.CheckerArchiveUrl,
		WriteupUrl:         arg.WriteupUrl,
		ExploitsUrl:        arg.ExploitsUrl,
		CheckStatus:        arg.CheckStatus,
		Ctf01dTraining:     arg.Ctf01dTraining,
		SourceKind:         arg.SourceKind,
		GitRepoUrl:         arg.GitRepoUrl,
		GitRef:             arg.GitRef,
		GitSubdir:          arg.GitSubdir,
		GitSyncStatus:      arg.GitSyncStatus,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	m.services[id] = svc
	m.byName[arg.Name] = id
	return svc, nil
}

func (m *mockQuerier) GetServiceByID(_ context.Context, id int64) (db.Service, error) {
	svc, ok := m.services[id]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	return svc, nil
}

func (m *mockQuerier) ListServices(_ context.Context, arg db.ListServicesParams) ([]db.Service, error) {
	var result []db.Service
	for _, svc := range m.services {
		if arg.PublicFilter != nil && svc.Public != *arg.PublicFilter {
			continue
		}
		if arg.SearchQuery != nil && *arg.SearchQuery != "" {
			if !contains(svc.Name, *arg.SearchQuery) {
				continue
			}
		}
		result = append(result, svc)
	}
	if len(result) > int(arg.Limit) {
		result = result[:arg.Limit]
	}
	return result, nil
}

func (m *mockQuerier) CountServices(_ context.Context, arg db.CountServicesParams) (int64, error) {
	var count int64
	for _, svc := range m.services {
		if arg.PublicFilter != nil && svc.Public != *arg.PublicFilter {
			continue
		}
		if arg.SearchQuery != nil && *arg.SearchQuery != "" {
			if !contains(svc.Name, *arg.SearchQuery) {
				continue
			}
		}
		count++
	}
	return count, nil
}

func (m *mockQuerier) UpdateService(_ context.Context, arg db.UpdateServiceParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.Name = arg.Name
	if arg.PublicDescription != nil {
		svc.PublicDescription = arg.PublicDescription
	}
	if arg.PrivateDescription != nil {
		svc.PrivateDescription = arg.PrivateDescription
	}
	if arg.Author != nil {
		svc.Author = arg.Author
	}
	if arg.Copyright != nil {
		svc.Copyright = arg.Copyright
	}
	if arg.AvatarUrl != nil {
		svc.AvatarUrl = arg.AvatarUrl
	}
	svc.Public = arg.Public
	if arg.ServiceArchiveUrl != nil {
		svc.ServiceArchiveUrl = arg.ServiceArchiveUrl
	}
	if arg.CheckerArchiveUrl != nil {
		svc.CheckerArchiveUrl = arg.CheckerArchiveUrl
	}
	if arg.WriteupUrl != nil {
		svc.WriteupUrl = arg.WriteupUrl
	}
	if arg.ExploitsUrl != nil {
		svc.ExploitsUrl = arg.ExploitsUrl
	}
	if arg.Ctf01dTraining != nil {
		svc.Ctf01dTraining = arg.Ctf01dTraining
	}
	svc.UpdatedAt = time.Now()
	m.services[arg.ID] = svc
	delete(m.byName, svc.Name)
	m.byName[svc.Name] = arg.ID
	return svc, nil
}

func (m *mockQuerier) DeleteService(_ context.Context, id int64) error {
	svc, ok := m.services[id]
	if !ok {
		return nil
	}
	delete(m.byName, svc.Name)
	delete(m.services, id)
	return nil
}

func (m *mockQuerier) SetPublic(_ context.Context, arg db.SetPublicParams) (db.Service, error) {
	svc, ok := m.services[arg.ID]
	if !ok {
		return db.Service{}, pgx.ErrNoRows
	}
	svc.Public = arg.Public
	svc.UpdatedAt = time.Now()
	m.services[arg.ID] = svc
	return svc, nil
}

func (m *mockQuerier) SetGitSource(_ context.Context, arg db.SetGitSourceParams) (db.Service, error) {
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
	svc.UpdatedAt = time.Now()
	m.services[arg.ID] = svc
	return svc, nil
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestCreate_Success(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	author := "test-author"
	avatar := "https://example.com/avatar.png"
	result, err := svc.Create(context.Background(), CreateParams{
		Name:      "test-service",
		Author:    &author,
		AvatarUrl: &avatar,
		Public:    true,
	}, true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("ID = %d, want 1", result.ID)
	}
	if result.Name != "test-service" {
		t.Errorf("Name = %q, want %q", result.Name, "test-service")
	}
	if result.Author == nil || *result.Author != "test-author" {
		t.Errorf("Author = %v, want test-author", result.Author)
	}
	if result.CheckStatus != "unknown" {
		t.Errorf("CheckStatus = %q, want %q", result.CheckStatus, "unknown")
	}
}

func TestCreate_DuplicateName(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "test-service"})
	_, err := svc.Create(context.Background(), CreateParams{Name: "test-service"}, true)
	if err != errs.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_InvalidURL(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Create(context.Background(), CreateParams{
		Name:      "test-service",
		AvatarUrl: strPtr("ftp://bad.com"),
	}, true)
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestCreate_AvatarDataImage(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	uri := tinyPNGDataURI(t)
	model, err := svc.Create(context.Background(), CreateParams{
		Name:      "test-service",
		AvatarUrl: &uri,
	}, true)
	if err != nil {
		t.Fatalf("expected no error for data:image avatar, got %v", err)
	}
	if model.AvatarUrl == nil || !strings.HasPrefix(*model.AvatarUrl, "data:image/png;base64,") {
		t.Fatalf("avatar should be normalized to a png data URI, got %v", model.AvatarUrl)
	}
}

func TestCreate_AvatarInvalidDataImage(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Create(context.Background(), CreateParams{
		Name:      "test-service",
		AvatarUrl: strPtr("data:image/png;base64,abc"),
	}, true)
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Fatalf("expected ValidationError for undecodable data:image avatar, got %v", err)
	}
}

// tinyPNGDataURI returns a data: URI for a minimal valid transparent PNG.
func tinyPNGDataURI(t *testing.T) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	img.Set(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 0})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestCreate_InvalidArchiveURL(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Create(context.Background(), CreateParams{
		Name:              "test-service",
		ServiceArchiveUrl: strPtr("not-a-url"),
	}, true)
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestCreate_GitSourceRequiresAdmin(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Create(context.Background(), CreateParams{
		Name: "test-service",
		GitSource: &GitSourceInput{
			RepoURL: "ssh://git@example.com/team/repo.git",
		},
	}, false)
	if err != errs.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "test-service"})

	result, err := svc.GetByID(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if result.Name != "test-service" {
		t.Errorf("Name = %q, want %q", result.Name, "test-service")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.GetByID(context.Background(), 999, true)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByID_AdminMasking(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	private := "secret desc"
	mustCreateService(t, svc, CreateParams{
		Name:               "test-service",
		PrivateDescription: &private,
	})

	adminResult, _ := svc.GetByID(context.Background(), 1, true)
	if adminResult.PrivateDescription == nil || *adminResult.PrivateDescription != "secret desc" {
		t.Errorf("admin should see private_description")
	}

	userResult, _ := svc.GetByID(context.Background(), 1, false)
	if userResult.PrivateDescription != nil {
		t.Errorf("non-admin should not see private_description, got %v", userResult.PrivateDescription)
	}
}

func TestList(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "svc-public", Public: true})
	mustCreateService(t, svc, CreateParams{Name: "svc-private", Public: false})

	result, err := svc.List(context.Background(), 1, 10, nil, nil, true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(result.Items))
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
}

func TestList_PublicFilter(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "svc-public", Public: true})
	mustCreateService(t, svc, CreateParams{Name: "svc-private", Public: false})

	result, err := svc.List(context.Background(), 1, 10, boolPtr(true), nil, true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(result.Items))
	}
	if result.Items[0].Name != "svc-public" {
		t.Errorf("Name = %q, want %q", result.Items[0].Name, "svc-public")
	}
}

func TestList_Search(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "web-nginx"})
	mustCreateService(t, svc, CreateParams{Name: "crypto-rsa"})

	result, err := svc.List(context.Background(), 1, 10, nil, strPtr("nginx"), true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(result.Items))
	}
	if result.Items[0].Name != "web-nginx" {
		t.Errorf("Name = %q, want %q", result.Items[0].Name, "web-nginx")
	}
}

func TestList_Pagination(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	for i := 0; i < 5; i++ {
		mustCreateService(t, svc, CreateParams{
			Name:   "svc-" + string(rune('0'+i)),
			Public: true,
		})
	}

	result, err := svc.List(context.Background(), 1, 3, nil, nil, true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(result.Items))
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
	if result.Page != 1 || result.PerPage != 3 {
		t.Errorf("Page=%d PerPage=%d, want 1,3", result.Page, result.PerPage)
	}
}

func TestUpdate_Success(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "test-service"})

	newName := "updated-service"
	result, err := svc.Update(context.Background(), 1, UpdateParams{
		Name: &newName,
	}, true)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result.Name != "updated-service" {
		t.Errorf("Name = %q, want %q", result.Name, "updated-service")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Update(context.Background(), 999, UpdateParams{
		Name: strPtr("x"),
	}, true)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdate_InvalidURL(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "test-service"})

	_, err := svc.Update(context.Background(), 1, UpdateParams{
		AvatarUrl: strPtr("ftp://bad.com"),
	}, true)
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestUpdate_GitSourceRequiresAdmin(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "test-service"})

	_, err := svc.Update(context.Background(), 1, UpdateParams{
		GitSource: &GitSourceInput{
			RepoURL: "ssh://git@example.com/team/repo.git",
		},
	}, false)
	if err != errs.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "test-service"})

	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.GetByID(context.Background(), 1, true)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestTogglePublic(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateService(t, svc, CreateParams{Name: "test-service", Public: true})

	result, err := svc.TogglePublic(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("TogglePublic: %v", err)
	}
	if result.Public {
		t.Errorf("Public = true, want false after toggle")
	}

	result, err = svc.TogglePublic(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("TogglePublic: %v", err)
	}
	if !result.Public {
		t.Errorf("Public = false, want true after second toggle")
	}
}

func TestTogglePublic_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.TogglePublic(context.Background(), 999, true)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestURLValidation_AllValidHTTP(t *testing.T) {
	err := validateServiceURLs(
		strPtr("https://example.com/avatar.png"),
		strPtr("https://example.com/service.zip"),
		strPtr("https://example.com/checker.zip"),
		strPtr("https://example.com/writeup"),
		strPtr("https://example.com/exploits"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestURLValidation_AvatarDataImage(t *testing.T) {
	err := validateServiceURLs(
		strPtr("data:image/png;base64,abc123"),
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("expected no error for data:image, got %v", err)
	}
}

func TestURLValidation_InvalidServiceArchiveURL(t *testing.T) {
	err := validateServiceURLs(
		nil,
		strPtr("not-a-url"),
		nil, nil, nil,
	)
	if verr, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	} else if _, exists := verr.Fields["service_archive_url"]; !exists {
		t.Errorf("expected field 'service_archive_url' in validation error, got %v", verr.Fields)
	}
}

func TestURLValidation_EmptyStrings(t *testing.T) {
	err := validateServiceURLs(
		strPtr(""),
		strPtr(""),
		strPtr(""),
		strPtr(""),
		strPtr(""),
	)
	if err != nil {
		t.Fatalf("empty strings should be valid (no validation), got %v", err)
	}
}

func TestURLValidation_NilValues(t *testing.T) {
	err := validateServiceURLs(nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("nil values should be valid, got %v", err)
	}
}

func TestFromDB_DefaultTraining(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	result, err := svc.Create(context.Background(), CreateParams{
		Name:           "test-service",
		Ctf01dTraining: nil,
	}, true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.Ctf01dTraining == nil {
		t.Fatal("Ctf01dTraining should not be nil")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result.Ctf01dTraining, &parsed); err != nil {
		t.Fatalf("unmarshal Ctf01dTraining: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("Ctf01dTraining = %v, want empty object", parsed)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && strings.Contains(s, sub)
}
