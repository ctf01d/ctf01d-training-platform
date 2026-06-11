package scoreboard

import (
	"context"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type ScoreboardEntry struct {
	TeamID   int64  `json:"team_id"`
	TeamName string `json:"team_name"`
	Score    int    `json:"score"`
	Position int    `json:"position"`
}

type Scoreboard struct {
	GameID  int64             `json:"game_id"`
	Status  string            `json:"status"`
	Entries []ScoreboardEntry `json:"entries"`
}

type GlobalEntry struct {
	TeamID     int64  `json:"team_id"`
	TeamName   string `json:"team_name"`
	TotalScore int    `json:"total_score"`
}

type GlobalScoreboard struct {
	Entries []GlobalEntry `json:"entries"`
}

type GameQuerier interface {
	GetGameByID(ctx context.Context, id int64) (db.Game, error)
}

type ResultQuerier interface {
	ListResultsByGame(ctx context.Context, gameID int64) ([]db.Result, error)
	ListAllResults(ctx context.Context) ([]db.Result, error)
}

type FinalResultQuerier interface {
	ListFinalResultsByGame(ctx context.Context, gameID int64) ([]db.FinalResult, error)
}

type TeamQuerier interface {
	GetTeamByID(ctx context.Context, id int64) (db.Team, error)
}

type Service struct {
	games        GameQuerier
	results      ResultQuerier
	finalResults FinalResultQuerier
	teams        TeamQuerier
}

func NewService(games GameQuerier, results ResultQuerier, finalResults FinalResultQuerier, teams TeamQuerier) *Service {
	return &Service{games: games, results: results, finalResults: finalResults, teams: teams}
}

func (s *Service) ForGame(ctx context.Context, gameID int64, viewerRole string) (*Scoreboard, error) {
	game, err := s.games.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, errs.ErrNotFound
	}

	now := time.Now()
	var scOpensAt *time.Time
	if game.ScoreboardOpensAt.Valid {
		scOpensAt = &game.ScoreboardOpensAt.Time
	}
	var scClosesAt *time.Time
	if game.ScoreboardClosesAt.Valid {
		scClosesAt = &game.ScoreboardClosesAt.Time
	}

	sbStatus := computeScoreboardStatus(scOpensAt, scClosesAt, now)

	if viewerRole != "admin" && (sbStatus == "closed" || sbStatus == "upcoming") {
		return nil, errs.ErrForbidden
	}

	var entries []ScoreboardEntry

	if game.Finalized {
		finalResults, err := s.finalResults.ListFinalResultsByGame(ctx, gameID)
		if err != nil {
			return nil, err
		}
		for _, fr := range finalResults {
			team, err := s.teams.GetTeamByID(ctx, fr.TeamID)
			if err != nil {
				continue
			}
			pos := 0
			if fr.Position != nil {
				pos = int(*fr.Position)
			}
			entries = append(entries, ScoreboardEntry{
				TeamID:   fr.TeamID,
				TeamName: team.Name,
				Score:    int(fr.Score),
				Position: pos,
			})
		}
	} else {
		results, err := s.results.ListResultsByGame(ctx, gameID)
		if err != nil {
			return nil, err
		}
		for i, r := range results {
			team, err := s.teams.GetTeamByID(ctx, r.TeamID)
			if err != nil {
				continue
			}
			score := 0
			if r.Score != nil {
				score = int(*r.Score)
			}
			entries = append(entries, ScoreboardEntry{
				TeamID:   r.TeamID,
				TeamName: team.Name,
				Score:    score,
				Position: i + 1,
			})
		}
	}

	if entries == nil {
		entries = []ScoreboardEntry{}
	}

	return &Scoreboard{
		GameID:  gameID,
		Status:  sbStatus,
		Entries: entries,
	}, nil
}

func (s *Service) Global(ctx context.Context) (*GlobalScoreboard, error) {
	allResults, err := s.results.ListAllResults(ctx)
	if err != nil {
		return nil, err
	}

	teamScores := make(map[int64]int)
	for _, r := range allResults {
		score := 0
		if r.Score != nil {
			score = int(*r.Score)
		}
		teamScores[r.TeamID] += score
	}

	type teamScore struct {
		TeamID int64
		Score  int
	}
	var sorted []teamScore
	for tid, sc := range teamScores {
		sorted = append(sorted, teamScore{TeamID: tid, Score: sc})
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Score > sorted[i].Score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	entries := make([]GlobalEntry, 0, len(sorted))
	for _, ts := range sorted {
		team, err := s.teams.GetTeamByID(ctx, ts.TeamID)
		if err != nil {
			continue
		}
		entries = append(entries, GlobalEntry{
			TeamID:     ts.TeamID,
			TeamName:   team.Name,
			TotalScore: ts.Score,
		})
	}

	return &GlobalScoreboard{Entries: entries}, nil
}

func computeScoreboardStatus(opensAt, closesAt *time.Time, now time.Time) string {
	if opensAt == nil && closesAt == nil {
		return "always"
	}
	if opensAt != nil && now.Before(*opensAt) {
		return "upcoming"
	}
	afterOpen := opensAt == nil || !now.Before(*opensAt)
	beforeClose := closesAt == nil || !now.After(*closesAt)
	if afterOpen && beforeClose {
		return "open"
	}
	return "closed"
}

// re-export for testing convenience
var _ = pgtype.Timestamptz{}
