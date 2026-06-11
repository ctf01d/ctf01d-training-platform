package games

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5"
)

type mockGameQuerier struct {
	games  map[int64]db.Game
	nextID int64
}

type mockGamesServiceQuerier struct {
	pairs map[string]bool
}

type mockResultQuerier struct {
	results map[int64][]db.Result
}

type mockFinalResultQuerier struct {
	finalResults map[int64][]db.FinalResult
}

type mockTxRunner struct{}

func newMocks() (*mockGameQuerier, *mockGamesServiceQuerier, *mockResultQuerier, *mockFinalResultQuerier, *mockTxRunner) {
	gq := &mockGameQuerier{games: make(map[int64]db.Game), nextID: 1}
	gsq := &mockGamesServiceQuerier{pairs: make(map[string]bool)}
	rq := &mockResultQuerier{results: make(map[int64][]db.Result)}
	frq := &mockFinalResultQuerier{finalResults: make(map[int64][]db.FinalResult)}
	tx := &mockTxRunner{}
	return gq, gsq, rq, frq, tx
}

func (m *mockTxRunner) RunInTx(_ context.Context, fn func(*db.Queries) error) error {
	return fn(nil)
}

func (m *mockGameQuerier) CreateGame(_ context.Context, arg db.CreateGameParams) (db.Game, error) {
	id := m.nextID
	m.nextID++
	now := time.Now()
	g := db.Game{
		ID: id, Name: arg.Name, Organizer: arg.Organizer,
		StartsAt: arg.StartsAt, EndsAt: arg.EndsAt,
		CreatedAt: now, UpdatedAt: now,
		AvatarUrl: arg.AvatarUrl, SiteUrl: arg.SiteUrl, CtftimeUrl: arg.CtftimeUrl,
		Finalized: arg.Finalized, FinalizedAt: arg.FinalizedAt,
		RegistrationOpensAt: arg.RegistrationOpensAt, RegistrationClosesAt: arg.RegistrationClosesAt,
		ScoreboardOpensAt: arg.ScoreboardOpensAt, ScoreboardClosesAt: arg.ScoreboardClosesAt,
		VpnUrl: arg.VpnUrl, VpnConfigUrl: arg.VpnConfigUrl,
		AccessInstructions: arg.AccessInstructions, AccessSecret: arg.AccessSecret,
	}
	m.games[id] = g
	return g, nil
}

func (m *mockGameQuerier) GetGameByID(_ context.Context, id int64) (db.Game, error) {
	g, ok := m.games[id]
	if !ok {
		return db.Game{}, pgx.ErrNoRows
	}
	return g, nil
}

func (m *mockGameQuerier) ListGames(_ context.Context, arg db.ListGamesParams) ([]db.Game, error) {
	var result []db.Game
	for i := int32(0); i < arg.Limit; i++ {
		idx := arg.Offset + i + 1
		if g, ok := m.games[int64(idx)]; ok {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *mockGameQuerier) CountGames(_ context.Context) (int64, error) {
	return int64(len(m.games)), nil
}

func (m *mockGameQuerier) UpdateGame(_ context.Context, arg db.UpdateGameParams) (db.Game, error) {
	g, ok := m.games[arg.ID]
	if !ok {
		return db.Game{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		g.Name = arg.Name
	}
	if arg.Organizer != nil {
		g.Organizer = arg.Organizer
	}
	if arg.SiteUrl != nil {
		g.SiteUrl = arg.SiteUrl
	}
	g.UpdatedAt = time.Now()
	m.games[arg.ID] = g
	return g, nil
}

func (m *mockGameQuerier) DeleteGame(_ context.Context, id int64) error {
	delete(m.games, id)
	return nil
}

func (m *mockGameQuerier) SetFinalized(_ context.Context, arg db.SetFinalizedParams) (db.Game, error) {
	g, ok := m.games[arg.ID]
	if !ok {
		return db.Game{}, pgx.ErrNoRows
	}
	g.Finalized = arg.Finalized
	g.FinalizedAt = arg.FinalizedAt
	g.UpdatedAt = time.Now()
	m.games[arg.ID] = g
	return g, nil
}

func (m *mockGamesServiceQuerier) AddService(_ context.Context, arg db.AddServiceParams) error {
	key := svcKey(arg.GameID, arg.ServiceID)
	m.pairs[key] = true
	return nil
}

func (m *mockGamesServiceQuerier) RemoveService(_ context.Context, arg db.RemoveServiceParams) error {
	key := svcKey(arg.GameID, arg.ServiceID)
	delete(m.pairs, key)
	return nil
}

func (m *mockGamesServiceQuerier) ListServicesByGame(_ context.Context, gameID int64) ([]int64, error) {
	var result []int64
	return result, nil
}

func svcKey(gameID, serviceID int64) string {
	return fmt.Sprintf("%d:%d", gameID, serviceID)
}

func (m *mockResultQuerier) ListResultsByGame(_ context.Context, gameID int64) ([]db.Result, error) {
	return m.results[gameID], nil
}

func (m *mockFinalResultQuerier) DeleteFinalResultsByGame(_ context.Context, gameID int64) error {
	delete(m.finalResults, gameID)
	return nil
}

func (m *mockFinalResultQuerier) InsertFinalResult(_ context.Context, arg db.InsertFinalResultParams) (db.FinalResult, error) {
	fr := db.FinalResult{
		GameID: arg.GameID, TeamID: arg.TeamID,
		Score: arg.Score, Position: arg.Position,
	}
	m.finalResults[arg.GameID] = append(m.finalResults[arg.GameID], fr)
	return fr, nil
}

func (m *mockFinalResultQuerier) ListFinalResultsByGame(_ context.Context, gameID int64) ([]db.FinalResult, error) {
	return m.finalResults[gameID], nil
}

func ptrStr(v string) *string { return &v }

func TestCreate_Success(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	game, err := svc.Create(context.Background(), CreateParams{Name: &name})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if game.ID != 1 {
		t.Errorf("ID = %d, want 1", game.ID)
	}
}

func TestCreate_InvalidURL(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	badURL := "not-a-url"
	_, err := svc.Create(context.Background(), CreateParams{
		Name:    ptrStr("Test"),
		SiteUrl: &badURL,
	})
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	svc.Create(context.Background(), CreateParams{Name: &name})

	game, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if *game.Name != "Test Game" {
		t.Errorf("Name = %v, want Test Game", game.Name)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	_, err := svc.GetByID(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	for i := 0; i < 5; i++ {
		n := fmt.Sprintf("Game %d", i)
		svc.Create(context.Background(), CreateParams{Name: &n})
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
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	svc.Create(context.Background(), CreateParams{Name: &name})

	newName := "Updated Game"
	game, err := svc.Update(context.Background(), 1, UpdateParams{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if *game.Name != "Updated Game" {
		t.Errorf("Name = %v, want Updated Game", game.Name)
	}
}

func TestDelete(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	svc.Create(context.Background(), CreateParams{Name: &name})

	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(context.Background(), 1)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFinalize_Success(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	svc.Create(context.Background(), CreateParams{Name: &name})

	score1 := int32(100)
	score2 := int32(200)
	rq.results[1] = []db.Result{
		{ID: 1, GameID: 1, TeamID: 1, Score: &score1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 2, GameID: 1, TeamID: 2, Score: &score2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	game, err := svc.Finalize(context.Background(), 1)
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if !game.Finalized {
		t.Error("expected game to be finalized")
	}

	fr, _ := frq.ListFinalResultsByGame(context.Background(), 1)
	if len(fr) != 2 {
		t.Fatalf("expected 2 final results, got %d", len(fr))
	}
	if fr[0].Score != 100 || *fr[0].Position != 1 {
		t.Errorf("first result: score=%d pos=%d, want 100/1", fr[0].Score, *fr[0].Position)
	}
	if fr[1].Score != 200 || *fr[1].Position != 2 {
		t.Errorf("second result: score=%d pos=%d, want 200/2", fr[1].Score, *fr[1].Position)
	}
}

func TestFinalize_AlreadyFinalized(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	svc.Create(context.Background(), CreateParams{Name: &name})
	svc.Finalize(context.Background(), 1)

	_, err := svc.Finalize(context.Background(), 1)
	if err != errs.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestUnfinalize_Success(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	svc.Create(context.Background(), CreateParams{Name: &name})
	svc.Finalize(context.Background(), 1)

	game, err := svc.Unfinalize(context.Background(), 1)
	if err != nil {
		t.Fatalf("Unfinalize: %v", err)
	}
	if game.Finalized {
		t.Error("expected game to be unfinalized")
	}
}

func TestUnfinalize_NotFinalized(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	svc.Create(context.Background(), CreateParams{Name: &name})

	_, err := svc.Unfinalize(context.Background(), 1)
	if err != errs.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestAddService(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	err := svc.AddService(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("AddService: %v", err)
	}
	if !gsq.pairs[svcKey(1, 10)] {
		t.Error("expected service to be added")
	}
}

func TestRemoveService(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	svc.AddService(context.Background(), 1, 10)
	err := svc.RemoveService(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("RemoveService: %v", err)
	}
	if gsq.pairs[svcKey(1, 10)] {
		t.Error("expected service to be removed")
	}
}

func TestValidHTTPURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"ftp://example.com", false},
		{"not-a-url", false},
		{"", false},
	}
	for _, tt := range tests {
		got := validHTTPURL(tt.input)
		if got != tt.want {
			t.Errorf("validHTTPURL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCreate_ValidURLs(t *testing.T) {
	gq, gsq, rq, frq, tx := newMocks()
	svc := NewService(gq, gsq, rq, frq, tx)

	name := "Test Game"
	siteUrl := "https://example.com"
	ctftimeUrl := "https://ctftime.org"
	game, err := svc.Create(context.Background(), CreateParams{
		Name:       &name,
		SiteUrl:    &siteUrl,
		CtftimeUrl: &ctftimeUrl,
	})
	if err != nil {
		t.Fatalf("Create with valid URLs: %v", err)
	}
	if game.ID == 0 {
		t.Error("expected non-zero ID")
	}
}
