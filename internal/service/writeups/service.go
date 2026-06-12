package writeups

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type Writeup struct {
	ID        int64     `json:"id"`
	GameID    int64     `json:"game_id"`
	TeamID    int64     `json:"team_id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateParams struct {
	GameID int64  `json:"game_id"`
	TeamID int64  `json:"team_id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type Querier interface {
	CreateWriteup(ctx context.Context, arg db.CreateWriteupParams) (db.Writeup, error)
	GetWriteupByID(ctx context.Context, id int64) (db.Writeup, error)
	ListWriteupsByGame(ctx context.Context, gameID int64) ([]db.Writeup, error)
	ListWriteupsByTeam(ctx context.Context, teamID int64) ([]db.Writeup, error)
	ListWriteupsByGameAndTeam(ctx context.Context, arg db.ListWriteupsByGameAndTeamParams) ([]db.Writeup, error)
	ListAllWriteups(ctx context.Context) ([]db.Writeup, error)
	DeleteWriteup(ctx context.Context, id int64) error
}

type TeamManager interface {
	CanManage(ctx context.Context, teamID, userID int64, globalRole string) error
}

type Service struct {
	q     Querier
	teams TeamManager
}

const (
	requiredFieldMessage = "is required"
	maxTitleLength       = 255
)

func NewService(q Querier, teams TeamManager) *Service {
	return &Service{q: q, teams: teams}
}

func (s *Service) Create(ctx context.Context, actorID int64, actorRole string, params CreateParams) (*Writeup, error) {
	if err := validateCreate(params); err != nil {
		return nil, err
	}
	if err := s.teams.CanManage(ctx, params.TeamID, actorID, actorRole); err != nil {
		return nil, err
	}

	dbWriteup, err := s.q.CreateWriteup(ctx, db.CreateWriteupParams{
		GameID: params.GameID,
		TeamID: params.TeamID,
		Title:  strings.TrimSpace(params.Title),
		Url:    strings.TrimSpace(params.URL),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	w := fromDB(dbWriteup)
	return &w, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Writeup, error) {
	dbWriteup, err := s.q.GetWriteupByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	w := fromDB(dbWriteup)
	return &w, nil
}

func (s *Service) ListByGame(ctx context.Context, gameID int64) ([]Writeup, error) {
	items, err := s.q.ListWriteupsByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	return fromDBList(items), nil
}

func (s *Service) ListByTeam(ctx context.Context, teamID int64) ([]Writeup, error) {
	items, err := s.q.ListWriteupsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return fromDBList(items), nil
}

func (s *Service) ListByGameAndTeam(ctx context.Context, gameID, teamID int64) ([]Writeup, error) {
	items, err := s.q.ListWriteupsByGameAndTeam(ctx, db.ListWriteupsByGameAndTeamParams{GameID: gameID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return fromDBList(items), nil
}

func (s *Service) ListAll(ctx context.Context) ([]Writeup, error) {
	items, err := s.q.ListAllWriteups(ctx)
	if err != nil {
		return nil, err
	}
	return fromDBList(items), nil
}

func (s *Service) Delete(ctx context.Context, actorID int64, actorRole string, id int64) error {
	dbWriteup, err := s.q.GetWriteupByID(ctx, id)
	if err != nil {
		return mapNotFound(err)
	}
	if err := s.teams.CanManage(ctx, dbWriteup.TeamID, actorID, actorRole); err != nil {
		return err
	}
	if err := s.q.DeleteWriteup(ctx, id); err != nil {
		return mapNotFound(err)
	}
	return nil
}

func validateCreate(params CreateParams) error {
	fields := make(map[string]string)
	title := strings.TrimSpace(params.Title)
	rawURL := strings.TrimSpace(params.URL)

	if params.GameID <= 0 {
		fields["game_id"] = requiredFieldMessage
	}
	if params.TeamID <= 0 {
		fields["team_id"] = requiredFieldMessage
	}
	if title == "" {
		fields["title"] = requiredFieldMessage
	} else if len(title) > maxTitleLength {
		fields["title"] = "must be at most 255 characters"
	}
	if rawURL == "" {
		fields["url"] = requiredFieldMessage
	} else if !validHTTPURL(rawURL) {
		fields["url"] = "must be a valid http(s):// URL"
	}

	if len(fields) > 0 {
		return errs.NewValidationError(fields)
	}
	return nil
}

func validHTTPURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

func fromDBList(items []db.Writeup) []Writeup {
	result := make([]Writeup, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result
}

func fromDB(w db.Writeup) Writeup {
	return Writeup{
		ID:        w.ID,
		GameID:    w.GameID,
		TeamID:    w.TeamID,
		Title:     w.Title,
		URL:       w.Url,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func mapNotFound(err error) error {
	if repository.IsNoRows(err) {
		return errs.ErrNotFound
	}
	return err
}

func mapDBError(err error) error {
	if repository.IsDuplicateKey(err) {
		return errs.ErrConflict
	}
	if repository.IsForeignKeyViolation(err) {
		return errs.ErrNotFound
	}
	return err
}
