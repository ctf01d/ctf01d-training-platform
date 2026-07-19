package gameteams

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

const maxHostnameLen = 253

// hostnameRe matches a DNS hostname (ASCII labels). Go's regexp has no
// lookahead, so the overall length is checked separately.
var hostnameRe = regexp.MustCompile(
	`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`,
)

// ipv4ShapeRe matches a dotted-decimal IPv4 shape so values like "10.10.1.999"
// are rejected as bad IPs instead of slipping through as a numeric hostname.
var ipv4ShapeRe = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)

// normalizeIPAddress trims and validates a game team's address. It goes into
// config.yml as ip-or-host, so it must be empty (clears it), a valid IPv4/IPv6
// address, or a DNS hostname. Returns the trimmed value on success.
func normalizeIPAddress(ip *string) (*string, error) {
	if ip == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*ip)
	if v == "" {
		return &v, nil
	}
	if net.ParseIP(v) != nil {
		return &v, nil // valid IPv4 or IPv6
	}
	// Not a valid IP: reject anything IPv4-shaped (bad octets), else allow hostname.
	if !ipv4ShapeRe.MatchString(v) && len(v) <= maxHostnameLen && hostnameRe.MatchString(v) {
		return &v, nil
	}
	return nil, errs.NewValidationError(map[string]string{
		"ip_address": "must be a valid IP address or hostname",
	})
}

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
	IpAddress       *string          `json:"ip_address"`
	Ctf01dID        *string          `json:"ctf01d_id"`
	Ctf01dOverrides *json.RawMessage `json:"ctf01d_overrides"`
	TeamType        *string          `json:"team_type"`
	Order           *int32           `json:"order"`
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
	RunInTx(ctx context.Context, fn func(queries *db.Queries) error) error
}

type Service struct {
	q  Querier
	tx TxRunner
}

func NewService(q Querier, tx TxRunner) *Service {
	return &Service{q: q, tx: tx}
}

func (s *Service) txQ(q *db.Queries) Querier {
	if q == nil {
		return s.q
	}
	return q
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*GameTeam, error) {
	normIP, err := normalizeIPAddress(params.IpAddress)
	if err != nil {
		return nil, err
	}
	params.IpAddress = normIP
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
		return nil, mapNotFound(err)
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
	normIP, err := normalizeIPAddress(params.IpAddress)
	if err != nil {
		return nil, err
	}
	params.IpAddress = normIP
	var overrides []byte
	if params.Ctf01dOverrides != nil {
		overrides = []byte(*params.Ctf01dOverrides)
	}
	dbGT, err := s.q.UpdateGameTeam(ctx, db.UpdateGameTeamParams{
		ID:              id,
		IpAddress:       params.IpAddress,
		Ctf01dID:        params.Ctf01dID,
		Ctf01dOverrides: overrides,
		TeamType:        params.TeamType,
		Order:           params.Order,
	})
	if err != nil {
		return nil, mapNotFound(err)
	}
	gt := fromDB(dbGT)
	return &gt, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.q.DeleteGameTeam(ctx, id)
}

func (s *Service) Reorder(ctx context.Context, gameID int64, items []ReorderItem) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		existing, err := tq.ListGameTeamsByGame(ctx, gameID)
		if err != nil {
			return err
		}
		allowed := make(map[int64]bool, len(existing))
		for _, gt := range existing {
			allowed[gt.ID] = true
		}
		for _, item := range items {
			if !allowed[item.ID] {
				return errs.ErrForbidden
			}
			order, err := int32FromInt(item.Order)
			if err != nil {
				return err
			}
			if err := tq.UpdateGameTeamOrder(ctx, db.UpdateGameTeamOrderParams{
				ID:    item.ID,
				Order: order,
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
	return err
}

const (
	minInt32 = -1 << 31
	maxInt32 = 1<<31 - 1
)

func int32FromInt(v int) (int32, error) {
	if v < minInt32 || v > maxInt32 {
		return 0, errs.NewValidationError(map[string]string{"order": "must fit int32"})
	}
	return int32(v), nil
}
