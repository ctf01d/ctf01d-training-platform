package results

import (
	"context"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type Result struct {
	ID        int64     `json:"id"`
	GameID    int64     `json:"game_id"`
	TeamID    int64     `json:"team_id"`
	Score     *int32    `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateParams struct {
	GameID int64  `json:"game_id"`
	TeamID int64  `json:"team_id"`
	Score  *int32 `json:"score"`
}

type UpdateParams struct {
	Score *int32 `json:"score"`
}

type GameQuerier interface {
	GetGameByID(ctx context.Context, id int64) (db.Game, error)
}

type Querier interface {
	CreateResult(ctx context.Context, arg db.CreateResultParams) (db.Result, error)
	GetResultByID(ctx context.Context, id int64) (db.Result, error)
	ListResultsByGame(ctx context.Context, gameID int64) ([]db.Result, error)
	ListResultsByTeam(ctx context.Context, teamID int64) ([]db.Result, error)
	ListResultsByGameAndTeam(ctx context.Context, arg db.ListResultsByGameAndTeamParams) ([]db.Result, error)
	ListAllResults(ctx context.Context) ([]db.Result, error)
	UpsertResult(ctx context.Context, arg db.UpsertResultParams) (db.Result, error)
	UpdateResult(ctx context.Context, arg db.UpdateResultParams) (db.Result, error)
	DeleteResult(ctx context.Context, id int64) error
}

type Service struct {
	q     Querier
	games GameQuerier
}

func NewService(q Querier, games GameQuerier) *Service {
	return &Service{q: q, games: games}
}

func (s *Service) checkNotFinalized(ctx context.Context, gameID int64, callerRole string) error {
	if callerRole == "admin" {
		return nil
	}
	game, err := s.games.GetGameByID(ctx, gameID)
	if err != nil {
		return nil
	}
	if game.Finalized {
		return errs.ErrForbidden
	}
	return nil
}

func (s *Service) Create(ctx context.Context, params CreateParams, callerRole string) (*Result, error) {
	if err := s.checkNotFinalized(ctx, params.GameID, callerRole); err != nil {
		return nil, err
	}
	dbResult, err := s.q.CreateResult(ctx, db.CreateResultParams{
		GameID: params.GameID,
		TeamID: params.TeamID,
		Score:  params.Score,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	r := fromDB(dbResult)
	return &r, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Result, error) {
	dbResult, err := s.q.GetResultByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "result")
	}
	r := fromDB(dbResult)
	return &r, nil
}

func (s *Service) ListByGame(ctx context.Context, gameID int64) ([]Result, error) {
	items, err := s.q.ListResultsByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	result := make([]Result, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) ListByTeam(ctx context.Context, teamID int64) ([]Result, error) {
	items, err := s.q.ListResultsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	result := make([]Result, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) ListByGameAndTeam(ctx context.Context, gameID, teamID int64) ([]Result, error) {
	items, err := s.q.ListResultsByGameAndTeam(ctx, db.ListResultsByGameAndTeamParams{GameID: gameID, TeamID: teamID})
	if err != nil {
		return nil, err
	}
	result := make([]Result, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) ListAll(ctx context.Context) ([]Result, error) {
	items, err := s.q.ListAllResults(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Result, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) Upsert(ctx context.Context, gameID, teamID int64, score *int32, callerRole string) (*Result, error) {
	if err := s.checkNotFinalized(ctx, gameID, callerRole); err != nil {
		return nil, err
	}
	dbResult, err := s.q.UpsertResult(ctx, db.UpsertResultParams{
		GameID: gameID,
		TeamID: teamID,
		Score:  score,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	r := fromDB(dbResult)
	return &r, nil
}

func (s *Service) Update(ctx context.Context, id int64, params UpdateParams, callerRole string) (*Result, error) {
	dbResult, err := s.q.GetResultByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "result")
	}
	if err := s.checkNotFinalized(ctx, dbResult.GameID, callerRole); err != nil {
		return nil, err
	}
	dbResult, err = s.q.UpdateResult(ctx, db.UpdateResultParams{
		ID:    id,
		Score: params.Score,
	})
	if err != nil {
		return nil, mapNotFound(err, "result")
	}
	r := fromDB(dbResult)
	return &r, nil
}

func (s *Service) Delete(ctx context.Context, id int64, callerRole string) error {
	dbResult, err := s.q.GetResultByID(ctx, id)
	if err != nil {
		return mapNotFound(err, "result")
	}
	if err := s.checkNotFinalized(ctx, dbResult.GameID, callerRole); err != nil {
		return err
	}
	return s.q.DeleteResult(ctx, id)
}

func fromDB(r db.Result) Result {
	return Result{
		ID:        r.ID,
		GameID:    r.GameID,
		TeamID:    r.TeamID,
		Score:     r.Score,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func mapNotFound(err error, entity string) error {
	if isNoRows(err) {
		return errs.ErrNotFound
	}
	return err
}

func mapDBError(err error) error {
	if isDuplicateKey(err) {
		return errs.ErrConflict
	}
	return err
}

func isNoRows(err error) bool {
	return err != nil && err.Error() == "no rows in result set"
}

func isDuplicateKey(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "violates unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
