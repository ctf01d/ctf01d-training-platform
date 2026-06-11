package scoreboard

import (
	"context"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockGameQuerier struct {
	games map[int64]db.Game
}

type mockResultQuerier struct {
	results map[int64][]db.Result
	all     []db.Result
}

type mockFinalResultQuerier struct {
	finalResults map[int64][]db.FinalResult
}

type mockTeamQuerier struct {
	teams map[int64]db.Team
}

func newMocks() (*mockGameQuerier, *mockResultQuerier, *mockFinalResultQuerier, *mockTeamQuerier) {
	gq := &mockGameQuerier{games: make(map[int64]db.Game)}
	rq := &mockResultQuerier{results: make(map[int64][]db.Result)}
	frq := &mockFinalResultQuerier{finalResults: make(map[int64][]db.FinalResult)}
	tq := &mockTeamQuerier{teams: make(map[int64]db.Team)}
	return gq, rq, frq, tq
}

func (m *mockGameQuerier) GetGameByID(_ context.Context, id int64) (db.Game, error) {
	g, ok := m.games[id]
	if !ok {
		return db.Game{}, pgx.ErrNoRows
	}
	return g, nil
}

func (m *mockResultQuerier) ListResultsByGame(_ context.Context, gameID int64) ([]db.Result, error) {
	return m.results[gameID], nil
}

func (m *mockResultQuerier) ListAllResults(_ context.Context) ([]db.Result, error) {
	if m.all != nil {
		return m.all, nil
	}
	var result []db.Result
	for _, rs := range m.results {
		result = append(result, rs...)
	}
	return result, nil
}

func (m *mockFinalResultQuerier) ListFinalResultsByGame(_ context.Context, gameID int64) ([]db.FinalResult, error) {
	return m.finalResults[gameID], nil
}

func (m *mockTeamQuerier) GetTeamByID(_ context.Context, id int64) (db.Team, error) {
	t, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return t, nil
}

func ptrInt32(v int32) *int32 { return &v }

func TestForGame_Finalized(t *testing.T) {
	gq, rq, frq, tq := newMocks()
	svc := NewService(gq, rq, frq, tq)

	gq.games[1] = db.Game{ID: 1, Finalized: true}
	tq.teams[1] = db.Team{ID: 1, Name: "Team A"}
	tq.teams[2] = db.Team{ID: 2, Name: "Team B"}
	frq.finalResults[1] = []db.FinalResult{
		{TeamID: 1, Score: 200, Position: ptrInt32(1)},
		{TeamID: 2, Score: 100, Position: ptrInt32(2)},
	}

	sb, err := svc.ForGame(context.Background(), 1, "admin")
	if err != nil {
		t.Fatalf("ForGame: %v", err)
	}
	if len(sb.Entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(sb.Entries))
	}
	if sb.Entries[0].Score != 200 || sb.Entries[0].Position != 1 {
		t.Errorf("first entry: score=%d pos=%d", sb.Entries[0].Score, sb.Entries[0].Position)
	}
}

func TestForGame_NotFinalized(t *testing.T) {
	gq, rq, frq, tq := newMocks()
	svc := NewService(gq, rq, frq, tq)

	gq.games[1] = db.Game{ID: 1, Finalized: false}
	tq.teams[1] = db.Team{ID: 1, Name: "Team A"}
	s1 := int32(100)
	rq.results[1] = []db.Result{
		{TeamID: 1, Score: &s1},
	}

	sb, err := svc.ForGame(context.Background(), 1, "admin")
	if err != nil {
		t.Fatalf("ForGame: %v", err)
	}
	if len(sb.Entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(sb.Entries))
	}
	if sb.Entries[0].Score != 100 || sb.Entries[0].Position != 1 {
		t.Errorf("entry: score=%d pos=%d", sb.Entries[0].Score, sb.Entries[0].Position)
	}
}

func TestForGame_NotFound(t *testing.T) {
	gq, rq, frq, tq := newMocks()
	svc := NewService(gq, rq, frq, tq)

	_, err := svc.ForGame(context.Background(), 999, "admin")
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestForGame_ClosedScoreboard(t *testing.T) {
	gq, rq, frq, tq := newMocks()
	svc := NewService(gq, rq, frq, tq)

	future := time.Now().Add(24 * time.Hour)
	gq.games[1] = db.Game{
		ID: 1, Finalized: false,
		ScoreboardOpensAt:  pgtype.Timestamptz{Time: future, Valid: true},
		ScoreboardClosesAt: pgtype.Timestamptz{Time: future.Add(2 * time.Hour), Valid: true},
	}

	_, err := svc.ForGame(context.Background(), 1, "player")
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden for closed scoreboard, got %v", err)
	}
}

func TestForGame_AdminCanSeeClosed(t *testing.T) {
	gq, rq, frq, tq := newMocks()
	svc := NewService(gq, rq, frq, tq)

	future := time.Now().Add(24 * time.Hour)
	gq.games[1] = db.Game{
		ID: 1, Finalized: false,
		ScoreboardOpensAt: pgtype.Timestamptz{Time: future, Valid: true},
	}

	sb, err := svc.ForGame(context.Background(), 1, "admin")
	if err != nil {
		t.Fatalf("admin should see closed scoreboard, got %v", err)
	}
	if sb == nil {
		t.Error("expected scoreboard")
	}
}

func TestForGame_EmptyEntries(t *testing.T) {
	gq, rq, frq, tq := newMocks()
	svc := NewService(gq, rq, frq, tq)

	gq.games[1] = db.Game{ID: 1, Finalized: false}

	sb, err := svc.ForGame(context.Background(), 1, "admin")
	if err != nil {
		t.Fatalf("ForGame: %v", err)
	}
	if sb.Entries == nil {
		t.Error("expected non-nil entries slice")
	}
}

func TestGlobal(t *testing.T) {
	gq, rq, frq, tq := newMocks()
	svc := NewService(gq, rq, frq, tq)

	tq.teams[1] = db.Team{ID: 1, Name: "Team A"}
	tq.teams[2] = db.Team{ID: 2, Name: "Team B"}

	s1 := int32(100)
	s2 := int32(200)
	s3 := int32(50)
	rq.all = []db.Result{
		{GameID: 1, TeamID: 1, Score: &s1},
		{GameID: 1, TeamID: 2, Score: &s2},
		{GameID: 2, TeamID: 1, Score: &s3},
	}

	gs, err := svc.Global(context.Background())
	if err != nil {
		t.Fatalf("Global: %v", err)
	}
	if len(gs.Entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(gs.Entries))
	}

	foundA := false
	foundB := false
	for _, e := range gs.Entries {
		if e.TeamID == 1 {
			foundA = true
			if e.TotalScore != 150 {
				t.Errorf("Team A TotalScore = %d, want 150", e.TotalScore)
			}
		}
		if e.TeamID == 2 {
			foundB = true
			if e.TotalScore != 200 {
				t.Errorf("Team B TotalScore = %d, want 200", e.TotalScore)
			}
		}
	}
	if !foundA || !foundB {
		t.Error("missing teams in global scoreboard")
	}
}

func TestComputeScoreboardStatus(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		opensAt  *time.Time
		closesAt *time.Time
		want     string
	}{
		{"always", nil, nil, "always"},
		{"upcoming", ptrTime(now.Add(1 * time.Hour)), nil, "upcoming"},
		{"open", nil, ptrTime(now.Add(1 * time.Hour)), "open"},
		{"closed", ptrTime(now.Add(-2 * time.Hour)), ptrTime(now.Add(-1 * time.Hour)), "closed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeScoreboardStatus(tt.opensAt, tt.closesAt, now)
			if got != tt.want {
				t.Errorf("computeScoreboardStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptrTime(v time.Time) *time.Time { return &v }

var _ = pgtype.Timestamptz{}
