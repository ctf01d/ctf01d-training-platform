package ctf01d

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockBuilderQuerier struct {
	game       db.Game
	gameTeams  []db.GameTeam
	serviceIDs []int64
	services   map[int64]db.Service
	teams      map[int64]db.Team
}

func (m *mockBuilderQuerier) GetGameByID(_ context.Context, id int64) (db.Game, error) {
	return m.game, nil
}

func (m *mockBuilderQuerier) ListGameTeamsByGame(_ context.Context, gameID int64) ([]db.GameTeam, error) {
	return m.gameTeams, nil
}

func (m *mockBuilderQuerier) ListServicesByGame(_ context.Context, gameID int64) ([]int64, error) {
	return m.serviceIDs, nil
}

func (m *mockBuilderQuerier) GetServiceByID(_ context.Context, id int64) (db.Service, error) {
	return m.services[id], nil
}

func (m *mockBuilderQuerier) GetTeamByID(_ context.Context, id int64) (db.Team, error) {
	return m.teams[id], nil
}

func strPtr(s string) *string { return &s }

func makeMockQ() *mockBuilderQuerier {
	return &mockBuilderQuerier{
		game: db.Game{
			ID:       1,
			Name:     strPtr("TestGame"),
			StartsAt: pgtype.Timestamptz{Time: time.Date(2025, 10, 1, 9, 0, 0, 0, time.UTC), Valid: true},
			EndsAt:   pgtype.Timestamptz{Time: time.Date(2025, 10, 1, 19, 0, 0, 0, time.UTC), Valid: true},
		},
		gameTeams: []db.GameTeam{
			{
				ID:        10,
				GameID:    1,
				TeamID:    100,
				IpAddress: strPtr("10.0.1.1"),
				Ctf01dID:  strPtr("team_alpha"),
			},
			{
				ID:        11,
				GameID:    1,
				TeamID:    101,
				IpAddress: strPtr("10.0.2.1"),
			},
		},
		serviceIDs: []int64{200, 201},
		services: map[int64]db.Service{
			200: {
				ID:               200,
				Name:             "Web Service",
				CheckerLocalPath: strPtr("/tmp/checker_web.zip"),
				ServiceLocalPath: strPtr("/tmp/service_web.zip"),
				Ctf01dTraining:   json.RawMessage(`{"script_wait": 15, "round_sleep": 45}`),
			},
			201: {
				ID:               201,
				Name:             "Crypto Service",
				CheckerLocalPath: nil,
				ServiceLocalPath: nil,
				Ctf01dTraining:   json.RawMessage(`{}`),
			},
		},
		teams: map[int64]db.Team{
			100: {ID: 100, Name: "Alpha", AvatarUrl: strPtr("http://example.com/logo.png")},
			101: {ID: 101, Name: "Beta", AvatarUrl: nil},
		},
	}
}

func TestBuildParams_Basic(t *testing.T) {
	b := NewBuilder(makeMockQ())
	req := Ctf01dExportRequest{}
	result, err := b.BuildParams(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	if result.Game.ID != "1" {
		t.Errorf("Game.ID = %q, want 1", result.Game.ID)
	}
	if result.Game.Name != "TestGame" {
		t.Errorf("Game.Name = %q, want TestGame", result.Game.Name)
	}
	if result.Game.FlagTTLMin != 10 {
		t.Errorf("Game.FlagTTLMin = %d, want 10", result.Game.FlagTTLMin)
	}
	if result.Game.BasicAttackCost != 100 {
		t.Errorf("Game.BasicAttackCost = %d, want 100", result.Game.BasicAttackCost)
	}

	if len(result.Teams) != 2 {
		t.Fatalf("len(Teams) = %d, want 2", len(result.Teams))
	}
	if result.Teams[0].ID != "team_alpha" {
		t.Errorf("Teams[0].ID = %q, want team_alpha", result.Teams[0].ID)
	}
	if result.Teams[0].IPAddress != "10.0.1.1" {
		t.Errorf("Teams[0].IPAddress = %q, want 10.0.1.1", result.Teams[0].IPAddress)
	}
	if result.Teams[1].ID != "team_101" {
		t.Errorf("Teams[1].ID = %q, want team_101 (fallback)", result.Teams[1].ID)
	}

	if len(result.Checkers) != 2 {
		t.Fatalf("len(Checkers) = %d, want 2", len(result.Checkers))
	}
	if result.Checkers[0].Name != "Web Service" {
		t.Errorf("Checkers[0].Name = %q, want Web Service", result.Checkers[0].Name)
	}
	if result.Checkers[0].ScriptWait != 15 {
		t.Errorf("Checkers[0].ScriptWait = %d, want 15", result.Checkers[0].ScriptWait)
	}
	if result.Checkers[0].RoundSleep != 45 {
		t.Errorf("Checkers[0].RoundSleep = %d, want 45", result.Checkers[0].RoundSleep)
	}
	if !result.Checkers[0].CheckerFromBundle {
		t.Error("Checkers[0].CheckerFromBundle = false, want true")
	}
	if result.Checkers[1].CheckerFromBundle {
		t.Error("Checkers[1].CheckerFromBundle = true, want false (no checker archive)")
	}

	if result.Scoreboard.Port != 8080 {
		t.Errorf("Scoreboard.Port = %d, want 8080", result.Scoreboard.Port)
	}

	if result.Options.Prefix != "ctf01d_package" {
		t.Errorf("Options.Prefix = %q, want ctf01d_package", result.Options.Prefix)
	}
}

func TestBuildParams_WithRequestOverrides(t *testing.T) {
	b := NewBuilder(makeMockQ())
	port := 9090
	prefix := "my_export"
	flagTTL := 5
	attackCost := 200
	defenceCost := 75.0
	includeHTML := false
	includeCompose := true

	req := Ctf01dExportRequest{
		Prefix:          &prefix,
		Port:            &port,
		FlagTtlMin:      &flagTTL,
		BasicAttackCost: &attackCost,
		DefenceCost:     &defenceCost,
		IncludeHtml:     &includeHTML,
		IncludeCompose:  &includeCompose,
	}
	result, err := b.BuildParams(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	if result.Game.FlagTTLMin != 5 {
		t.Errorf("FlagTTLMin = %d, want 5", result.Game.FlagTTLMin)
	}
	if result.Game.BasicAttackCost != 200 {
		t.Errorf("BasicAttackCost = %d, want 200", result.Game.BasicAttackCost)
	}
	if result.Game.DefenceCost != 75.0 {
		t.Errorf("DefenceCost = %f, want 75.0", result.Game.DefenceCost)
	}
	if result.Scoreboard.Port != 9090 {
		t.Errorf("Port = %d, want 9090", result.Scoreboard.Port)
	}
	if result.Options.Prefix != "my_export" {
		t.Errorf("Prefix = %q, want my_export", result.Options.Prefix)
	}
	if result.Options.IncludeHTML {
		t.Error("IncludeHTML = true, want false")
	}
	if !result.Options.IncludeCompose {
		t.Error("IncludeCompose = false, want true")
	}
}

func TestBuildParams_Warnings(t *testing.T) {
	b := NewBuilder(makeMockQ())
	req := Ctf01dExportRequest{}
	result, err := b.BuildParams(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	found := false
	for _, w := range result.Warnings {
		if w == `service "Crypto Service" has no local checker archive` {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about Crypto Service missing checker, got warnings: %v", result.Warnings)
	}
}

func TestBuildParams_Ctf01dOverrides(t *testing.T) {
	mq := makeMockQ()
	mq.gameTeams[0].Ctf01dOverrides = json.RawMessage(`{"ctf01d_custom_field": "custom_value"}`)
	b := NewBuilder(mq)

	result, err := b.BuildParams(context.Background(), 1, Ctf01dExportRequest{})
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	if result.Teams[0].Ctf01dExtra == nil {
		t.Fatal("Ctf01dExtra is nil, expected overrides")
	}
	if result.Teams[0].Ctf01dExtra["ctf01d_custom_field"] != "custom_value" {
		t.Errorf("Ctf01dExtra = %v, want custom_value", result.Teams[0].Ctf01dExtra)
	}
}

func TestBuildParams_ServiceTrainingOverrides(t *testing.T) {
	mq := makeMockQ()
	mq.services[200] = db.Service{
		ID:               200,
		Name:             "Trained",
		CheckerLocalPath: strPtr("/tmp/checker.zip"),
		Ctf01dTraining:   json.RawMessage(`{"script_rel": "./check.sh", "enabled": false, "script_wait": 20, "round_sleep": 60}`),
	}
	b := NewBuilder(mq)

	result, err := b.BuildParams(context.Background(), 1, Ctf01dExportRequest{})
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}

	if result.Checkers[0].ScriptRel != "./check.sh" {
		t.Errorf("ScriptRel = %q, want ./check.sh", result.Checkers[0].ScriptRel)
	}
	if result.Checkers[0].Enabled {
		t.Error("Enabled = true, want false")
	}
	if result.Checkers[0].ScriptWait != 20 {
		t.Errorf("ScriptWait = %d, want 20", result.Checkers[0].ScriptWait)
	}
	if result.Checkers[0].RoundSleep != 60 {
		t.Errorf("RoundSleep = %d, want 60", result.Checkers[0].RoundSleep)
	}
}

func TestBuildOptions_Defaults(t *testing.T) {
	b := NewBuilder(makeMockQ())
	opts, err := b.BuildOptions(context.Background(), 1)
	if err != nil {
		t.Fatalf("BuildOptions: %v", err)
	}

	if opts.FlagTtlMin != 10 {
		t.Errorf("FlagTtlMin = %d, want 10 (default, auto-calc capped)", opts.FlagTtlMin)
	}
	if opts.BasicAttackCost != 100 {
		t.Errorf("BasicAttackCost = %d, want 100", opts.BasicAttackCost)
	}
	if opts.Port != 8080 {
		t.Errorf("Port = %d, want 8080", opts.Port)
	}
	if !opts.IncludeHtml {
		t.Error("IncludeHtml = false, want true")
	}
	if opts.IncludeCompose {
		t.Error("IncludeCompose = true, want false")
	}
}

func TestBuildOptions_Warnings(t *testing.T) {
	b := NewBuilder(makeMockQ())
	opts, err := b.BuildOptions(context.Background(), 1)
	if err != nil {
		t.Fatalf("BuildOptions: %v", err)
	}

	hasNoIP := false
	hasNoChecker := false
	for _, w := range opts.Warnings {
		if w == `team "Alpha" (id=100) has no ip_address` {
			hasNoIP = true
		}
		if w == `service "Crypto Service" (id=201) has no local checker archive` {
			hasNoChecker = true
		}
	}

	if hasNoIP {
		t.Error("Alpha team has ip_address but got warning")
	}

	if !hasNoChecker {
		t.Errorf("expected warning about Crypto Service missing checker, got: %v", opts.Warnings)
	}
}

func TestBuildOptions_MissingTeamIP(t *testing.T) {
	mq := makeMockQ()
	mq.gameTeams[0].IpAddress = nil
	b := NewBuilder(mq)

	opts, err := b.BuildOptions(context.Background(), 1)
	if err != nil {
		t.Fatalf("BuildOptions: %v", err)
	}

	found := false
	for _, w := range opts.Warnings {
		if w == `team "Alpha" (id=100) has no ip_address` {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about missing ip_address, got: %v", opts.Warnings)
	}
}

func TestBuildParams_CoffeeBreak(t *testing.T) {
	b := NewBuilder(makeMockQ())
	start := time.Date(2025, 10, 1, 12, 0, 0, 0, time.UTC)
	end := time.Date(2025, 10, 1, 13, 0, 0, 0, time.UTC)
	req := Ctf01dExportRequest{
		CoffeeBreakStart: &start,
		CoffeeBreakEnd:   &end,
	}
	result, err := b.BuildParams(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("BuildParams: %v", err)
	}
	if result.Game.CoffeeBreakStartUTC == nil || !result.Game.CoffeeBreakStartUTC.Equal(start) {
		t.Errorf("CoffeeBreakStartUTC = %v, want %v", result.Game.CoffeeBreakStartUTC, start)
	}
	if result.Game.CoffeeBreakEndUTC == nil || !result.Game.CoffeeBreakEndUTC.Equal(end) {
		t.Errorf("CoffeeBreakEndUTC = %v, want %v", result.Game.CoffeeBreakEndUTC, end)
	}
}
