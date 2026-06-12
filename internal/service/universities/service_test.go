package universities

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type mockQuerier struct {
	universities map[int64]db.University
	nextID       int64
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		universities: make(map[int64]db.University),
		nextID:       1,
	}
}

func mustCreateUniversity(t *testing.T, svc *Service, params CreateParams) *University {
	t.Helper()
	university, err := svc.Create(context.Background(), params)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return university
}

func (m *mockQuerier) CreateUniversity(_ context.Context, arg db.CreateUniversityParams) (db.University, error) {
	id := m.nextID
	m.nextID++
	now := time.Now()
	u := db.University{
		ID:        id,
		Name:      arg.Name,
		SiteUrl:   arg.SiteUrl,
		AvatarUrl: arg.AvatarUrl,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.universities[id] = u
	return u, nil
}

func (m *mockQuerier) GetUniversityByID(_ context.Context, id int64) (db.University, error) {
	u, ok := m.universities[id]
	if !ok {
		return db.University{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *mockQuerier) ListUniversities(_ context.Context, arg db.ListUniversitiesParams) ([]db.University, error) {
	var result []db.University
	for i := int32(0); i < arg.Limit; i++ {
		idx := arg.Offset + i + 1
		if u, ok := m.universities[int64(idx)]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockQuerier) CountUniversities(_ context.Context) (int64, error) {
	return int64(len(m.universities)), nil
}

func (m *mockQuerier) UpdateUniversity(_ context.Context, arg db.UpdateUniversityParams) (db.University, error) {
	u, ok := m.universities[arg.ID]
	if !ok {
		return db.University{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		u.Name = arg.Name
	}
	if arg.SiteUrl != nil {
		u.SiteUrl = arg.SiteUrl
	}
	if arg.AvatarUrl != nil {
		u.AvatarUrl = arg.AvatarUrl
	}
	u.UpdatedAt = time.Now()
	m.universities[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) DeleteUniversity(_ context.Context, id int64) error {
	delete(m.universities, id)
	return nil
}

func TestCreate(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	name := "MIT"
	site := "https://mit.edu"
	u, err := svc.Create(context.Background(), CreateParams{
		Name:    &name,
		SiteUrl: &site,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID != 1 {
		t.Errorf("ID = %d, want 1", u.ID)
	}
	if u.Name == nil || *u.Name != "MIT" {
		t.Errorf("Name = %v, want MIT", u.Name)
	}
}

func TestGetByID_Success(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	name := "MIT"
	mustCreateUniversity(t, svc, CreateParams{Name: &name})

	u, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if u.Name == nil || *u.Name != "MIT" {
		t.Errorf("Name = %v, want MIT", u.Name)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.GetByID(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	for i := 0; i < 5; i++ {
		n := "Uni" + string(rune('0'+i))
		mustCreateUniversity(t, svc, CreateParams{Name: &n})
	}

	result, err := svc.List(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(result.Items))
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
}

func TestUpdate(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	name := "MIT"
	mustCreateUniversity(t, svc, CreateParams{Name: &name})

	newName := "Stanford"
	u, err := svc.Update(context.Background(), 1, UpdateParams{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if u.Name == nil || *u.Name != "Stanford" {
		t.Errorf("Name = %v, want Stanford", u.Name)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	newName := "Stanford"
	_, err := svc.Update(context.Background(), 999, UpdateParams{Name: &newName})
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	name := "MIT"
	mustCreateUniversity(t, svc, CreateParams{Name: &name})

	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.GetByID(context.Background(), 1)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
