package results

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type mockGameQuerier struct {
	games map[int64]db.Game
}

type mockQuerier struct {
	results map[int64]db.Result
	nextID  int64
	byGame  map[int64][]db.Result
}

func newMocks() (*mockGameQuerier, *mockQuerier) {
	gq := &mockGameQuerier{games: make(map[int64]db.Game)}
	q := &mockQuerier{results: make(map[int64]db.Result), nextID: 1, byGame: make(map[int64][]db.Result)}
	return gq, q
}

func (m *mockGameQuerier) GetGameByID(_ context.Context, id int64) (db.Game, error) {
	g, ok := m.games[id]
	if !ok {
		return db.Game{}, pgx.ErrNoRows
	}
	return g, nil
}

func (m *mockQuerier) CreateResult(_ context.Context, arg db.CreateResultParams) (db.Result, error) {
	for _, r := range m.results {
		if r.GameID == arg.GameID && r.TeamID == arg.TeamID {
			return db.Result{}, &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
		}
	}
	id := m.nextID
	m.nextID++
	now := time.Now()
	r := db.Result{ID: id, GameID: arg.GameID, TeamID: arg.TeamID, Score: arg.Score, CreatedAt: now, UpdatedAt: now}
	m.results[id] = r
	m.byGame[arg.GameID] = append(m.byGame[arg.GameID], r)
	return r, nil
}

func (m *mockQuerier) GetResultByID(_ context.Context, id int64) (db.Result, error) {
	r, ok := m.results[id]
	if !ok {
		return db.Result{}, pgx.ErrNoRows
	}
	return r, nil
}

func (m *mockQuerier) ListResultsByGame(_ context.Context, gameID int64) ([]db.Result, error) {
	return m.byGame[gameID], nil
}

func (m *mockQuerier) ListResultsByTeam(_ context.Context, teamID int64) ([]db.Result, error) {
	var result []db.Result
	for _, r := range m.results {
		if r.TeamID == teamID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListResultsByGameAndTeam(_ context.Context, arg db.ListResultsByGameAndTeamParams) ([]db.Result, error) {
	var result []db.Result
	for _, r := range m.results {
		if r.GameID == arg.GameID && r.TeamID == arg.TeamID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListAllResults(_ context.Context) ([]db.Result, error) {
	var result []db.Result
	for _, r := range m.results {
		result = append(result, r)
	}
	return result, nil
}

func (m *mockQuerier) UpsertResult(_ context.Context, arg db.UpsertResultParams) (db.Result, error) {
	for id, r := range m.results {
		if r.GameID == arg.GameID && r.TeamID == arg.TeamID {
			r.Score = arg.Score
			r.UpdatedAt = time.Now()
			m.results[id] = r
			return r, nil
		}
	}
	return m.CreateResult(context.Background(), db.CreateResultParams(arg))
}

func (m *mockQuerier) UpdateResult(_ context.Context, arg db.UpdateResultParams) (db.Result, error) {
	r, ok := m.results[arg.ID]
	if !ok {
		return db.Result{}, pgx.ErrNoRows
	}
	if arg.Score != nil {
		r.Score = arg.Score
	}
	r.UpdatedAt = time.Now()
	m.results[arg.ID] = r
	return r, nil
}

func (m *mockQuerier) DeleteResult(_ context.Context, id int64) error {
	delete(m.results, id)
	return nil
}

func ptrInt32(v int32) *int32 { return &v }

func TestCreate_Success(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	score := int32(100)
	r, err := svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &score}, "player")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID != 1 {
		t.Errorf("ID = %d, want 1", r.ID)
	}
}

func TestCreate_FinalizedForbidden(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)

	gq.games[1] = db.Game{ID: 1, Finalized: true}

	score := int32(100)
	_, err := svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &score}, "player")
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestCreate_FinalizedAdmin(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: true}

	score := int32(100)
	_, err := svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &score}, "admin")
	if err != nil {
		t.Fatalf("admin should be able to create on finalized game, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	score := int32(100)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &score}, "player")

	r, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if *r.Score != 100 {
		t.Errorf("Score = %v, want 100", r.Score)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)

	_, err := svc.GetByID(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListByGame(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	s1 := int32(100)
	s2 := int32(200)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &s1}, "player")
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 2, Score: &s2}, "player")

	items, err := svc.ListByGame(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}

func TestUpsert(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	s1 := int32(100)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &s1}, "player")

	s2 := int32(150)
	r, err := svc.Upsert(context.Background(), 1, 1, &s2, "player")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if *r.Score != 150 {
		t.Errorf("Score = %d, want 150", *r.Score)
	}
}

func TestUpdate(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	s1 := int32(100)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &s1}, "player")

	s2 := int32(200)
	r, err := svc.Update(context.Background(), 1, UpdateParams{Score: &s2}, "player")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if *r.Score != 200 {
		t.Errorf("Score = %d, want 200", *r.Score)
	}
}

func TestUpdate_FinalizedForbidden(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	s1 := int32(100)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &s1}, "player")

	gq.games[1] = db.Game{ID: 1, Finalized: true}

	s2 := int32(200)
	_, err := svc.Update(context.Background(), 1, UpdateParams{Score: &s2}, "player")
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	s1 := int32(100)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &s1}, "player")

	err := svc.Delete(context.Background(), 1, "player")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(context.Background(), 1)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_FinalizedForbidden(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}

	s1 := int32(100)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &s1}, "player")

	gq.games[1] = db.Game{ID: 1, Finalized: true}

	err := svc.Delete(context.Background(), 1, "player")
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestListAll(t *testing.T) {
	gq, q := newMocks()
	svc := NewService(q, gq)
	gq.games[1] = db.Game{ID: 1, Finalized: false}
	gq.games[2] = db.Game{ID: 2, Finalized: false}

	s1 := int32(100)
	s2 := int32(200)
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Score: &s1}, "player")
	svc.Create(context.Background(), CreateParams{GameID: 2, TeamID: 1, Score: &s2}, "player")

	items, err := svc.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}
