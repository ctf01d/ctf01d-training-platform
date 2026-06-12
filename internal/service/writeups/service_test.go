package writeups

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type mockQuerier struct {
	writeups map[int64]db.Writeup
	nextID   int64
}

type mockTeamManager struct {
	err error
}

func newMocks() (*mockQuerier, *mockTeamManager) {
	return &mockQuerier{writeups: make(map[int64]db.Writeup), nextID: 1}, &mockTeamManager{}
}

func (m *mockTeamManager) CanManage(_ context.Context, _ int64, _ int64, _ string) error {
	return m.err
}

func (m *mockQuerier) CreateWriteup(_ context.Context, arg db.CreateWriteupParams) (db.Writeup, error) {
	for _, w := range m.writeups {
		if w.GameID == arg.GameID && w.TeamID == arg.TeamID && w.Title == arg.Title {
			return db.Writeup{}, &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
		}
	}
	id := m.nextID
	m.nextID++
	now := time.Now()
	w := db.Writeup{
		ID:        id,
		GameID:    arg.GameID,
		TeamID:    arg.TeamID,
		Title:     arg.Title,
		Url:       arg.Url,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.writeups[id] = w
	return w, nil
}

func (m *mockQuerier) GetWriteupByID(_ context.Context, id int64) (db.Writeup, error) {
	w, ok := m.writeups[id]
	if !ok {
		return db.Writeup{}, pgx.ErrNoRows
	}
	return w, nil
}

func (m *mockQuerier) ListWriteupsByGame(_ context.Context, gameID int64) ([]db.Writeup, error) {
	result := make([]db.Writeup, 0)
	for _, w := range m.writeups {
		if w.GameID == gameID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListWriteupsByTeam(_ context.Context, teamID int64) ([]db.Writeup, error) {
	result := make([]db.Writeup, 0)
	for _, w := range m.writeups {
		if w.TeamID == teamID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListWriteupsByGameAndTeam(_ context.Context, arg db.ListWriteupsByGameAndTeamParams) ([]db.Writeup, error) {
	result := make([]db.Writeup, 0)
	for _, w := range m.writeups {
		if w.GameID == arg.GameID && w.TeamID == arg.TeamID {
			result = append(result, w)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListAllWriteups(_ context.Context) ([]db.Writeup, error) {
	result := make([]db.Writeup, 0, len(m.writeups))
	for _, w := range m.writeups {
		result = append(result, w)
	}
	return result, nil
}

func (m *mockQuerier) DeleteWriteup(_ context.Context, id int64) error {
	delete(m.writeups, id)
	return nil
}

func TestCreate_Success(t *testing.T) {
	q, tm := newMocks()
	svc := NewService(q, tm)

	writeup, err := svc.Create(context.Background(), 10, "player", CreateParams{
		GameID: 1,
		TeamID: 2,
		Title:  "writeup",
		URL:    "https://example.com/writeup",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if writeup.ID != 1 {
		t.Errorf("ID = %d, want 1", writeup.ID)
	}
	if writeup.Title != "writeup" {
		t.Errorf("Title = %q, want writeup", writeup.Title)
	}
}

func TestCreate_Validation(t *testing.T) {
	q, tm := newMocks()
	svc := NewService(q, tm)

	_, err := svc.Create(context.Background(), 10, "player", CreateParams{
		GameID: 1,
		TeamID: 2,
		Title:  "",
		URL:    "ftp://example.com/writeup",
	})

	var validationErr *errs.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if validationErr.Fields["title"] == "" {
		t.Errorf("missing title validation")
	}
	if validationErr.Fields["url"] == "" {
		t.Errorf("missing url validation")
	}
}

func TestCreate_Forbidden(t *testing.T) {
	q, tm := newMocks()
	tm.err = errs.ErrForbidden
	svc := NewService(q, tm)

	_, err := svc.Create(context.Background(), 10, "player", CreateParams{
		GameID: 1,
		TeamID: 2,
		Title:  "writeup",
		URL:    "https://example.com/writeup",
	})
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreate_Duplicate(t *testing.T) {
	q, tm := newMocks()
	svc := NewService(q, tm)
	params := CreateParams{
		GameID: 1,
		TeamID: 2,
		Title:  "writeup",
		URL:    "https://example.com/writeup",
	}

	if _, err := svc.Create(context.Background(), 10, "player", params); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(context.Background(), 10, "player", params)
	if !errors.Is(err, errs.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestDelete_ChecksManager(t *testing.T) {
	q, tm := newMocks()
	svc := NewService(q, tm)
	writeup, err := svc.Create(context.Background(), 10, "player", CreateParams{
		GameID: 1,
		TeamID: 2,
		Title:  "writeup",
		URL:    "https://example.com/writeup",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	tm.err = errs.ErrForbidden
	err = svc.Delete(context.Background(), 11, "player", writeup.ID)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
