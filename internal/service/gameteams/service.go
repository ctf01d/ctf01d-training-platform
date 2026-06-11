package gameteams

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type GameTeam struct {
	ID              int64           `json:"id"`
	GameID          int64           `json:"game_id"`
	TeamID          int64           `json:"team_id"`
	IpAddress       *string         `json:"ip_address"`
	Ctf01dID        *string         `json:"ctf01d_id"`
	Ctf01dOverrides json.RawMessage `json:"ctf01d_overrides"`
	TeamType        *string         `json:"team_type"`
	Order           int32           `json:"order"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ReorderItem struct {
	ID    int64 `json:"id"`
	Order int   `json:"order"`
}

type CreateParams struct {
	GameID          int64           `json:"game_id"`
	TeamID          int64           `json:"team_id"`
	IpAddress       *string         `json:"ip_address"`
	Ctf01dID        *string         `json:"ctf01d_id"`
	Ctf01dOverrides json.RawMessage `json:"ctf01d_overrides"`
	TeamType        *string         `json:"team_type"`
	Order           int32           `json:"order"`
}

type UpdateParams struct {
	IpAddress       *string         `json:"ip_address"`
	Ctf01dID        *string         `json:"ctf01d_id"`
	Ctf01dOverrides json.RawMessage `json:"ctf01d_overrides"`
	TeamType        *string         `json:"team_type"`
	Order           int32           `json:"order"`
}

type Querier interface {
	CreateGameTeam(ctx context.Context, arg db.CreateGameTeamParams) (db.GameTeam, error)
	GetGameTeamByID(ctx context.Context, id int64) (db.GameTeam, error)
	ListGameTeamsByGame(ctx context.Context, gameID int64) ([]db.GameTeam, error)
	UpdateGameTeam(ctx context.Context, arg db.UpdateGameTeamParams) (db.GameTeam, error)
	DeleteGameTeam(ctx context.Context, id int64) error
	UpdateGameTeamOrder(ctx context.Context, arg db.UpdateGameTeamOrderParams) error
}

type TxRunner interface {
	RunInTx(ctx context.Context, fn func() error) error
}

type Service struct {
	q  Querier
	tx TxRunner
}

func NewService(q Querier, tx TxRunner) *Service {
	return &Service{q: q, tx: tx}
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*GameTeam, error) {
	if params.Ctf01dOverrides == nil {
		params.Ctf01dOverrides = json.RawMessage("{}")
	}
	dbGT, err := s.q.CreateGameTeam(ctx, db.CreateGameTeamParams{
		GameID:          params.GameID,
		TeamID:          params.TeamID,
		IpAddress:       params.IpAddress,
		Ctf01dID:        params.Ctf01dID,
		Ctf01dOverrides: params.Ctf01dOverrides,
		TeamType:        params.TeamType,
		Order:           params.Order,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	gt := fromDB(dbGT)
	return &gt, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*GameTeam, error) {
	dbGT, err := s.q.GetGameTeamByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "game_team")
	}
	gt := fromDB(dbGT)
	return &gt, nil
}

func (s *Service) ListByGame(ctx context.Context, gameID int64) ([]GameTeam, error) {
	items, err := s.q.ListGameTeamsByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	result := make([]GameTeam, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, id int64, params UpdateParams) (*GameTeam, error) {
	var order int32
	order = params.Order
	dbGT, err := s.q.UpdateGameTeam(ctx, db.UpdateGameTeamParams{
		ID:              id,
		IpAddress:       params.IpAddress,
		Ctf01dID:        params.Ctf01dID,
		Ctf01dOverrides: params.Ctf01dOverrides,
		TeamType:        params.TeamType,
		Order:           order,
	})
	if err != nil {
		return nil, mapNotFound(err, "game_team")
	}
	gt := fromDB(dbGT)
	return &gt, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.q.DeleteGameTeam(ctx, id)
}

func (s *Service) Reorder(ctx context.Context, gameID int64, items []ReorderItem) error {
	return s.tx.RunInTx(ctx, func() error {
		for _, item := range items {
			if err := s.q.UpdateGameTeamOrder(ctx, db.UpdateGameTeamOrderParams{
				ID:    item.ID,
				Order: int32(item.Order),
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func fromDB(gt db.GameTeam) GameTeam {
	return GameTeam{
		ID:              gt.ID,
		GameID:          gt.GameID,
		TeamID:          gt.TeamID,
		IpAddress:       gt.IpAddress,
		Ctf01dID:        gt.Ctf01dID,
		Ctf01dOverrides: gt.Ctf01dOverrides,
		TeamType:        gt.TeamType,
		Order:           gt.Order,
		CreatedAt:       gt.CreatedAt,
		UpdatedAt:       gt.UpdatedAt,
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
