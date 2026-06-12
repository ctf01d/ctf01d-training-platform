package games

import (
	"context"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type Game struct {
	ID                   int64              `json:"id"`
	Name                 *string            `json:"name"`
	Organizer            *string            `json:"organizer"`
	StartsAt             *time.Time         `json:"starts_at"`
	EndsAt               *time.Time         `json:"ends_at"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
	AvatarUrl            *string            `json:"avatar_url"`
	SiteUrl              *string            `json:"site_url"`
	CtftimeUrl           *string            `json:"ctftime_url"`
	Finalized            bool               `json:"finalized"`
	FinalizedAt          *time.Time         `json:"finalized_at"`
	RegistrationOpensAt  *time.Time         `json:"registration_opens_at"`
	RegistrationClosesAt *time.Time         `json:"registration_closes_at"`
	ScoreboardOpensAt    *time.Time         `json:"scoreboard_opens_at"`
	ScoreboardClosesAt   *time.Time         `json:"scoreboard_closes_at"`
	VpnUrl               *string            `json:"vpn_url"`
	VpnConfigUrl         *string            `json:"vpn_config_url"`
	AccessInstructions   *string            `json:"access_instructions"`
	AccessSecret         *string            `json:"access_secret"`
	Status               GameStatus         `json:"status"`
	RegistrationStatus   RegistrationStatus `json:"registration_status"`
	ScoreboardStatusVal  ScoreboardStatus   `json:"scoreboard_status"`
}

type GameListResult struct {
	Items   []Game `json:"items"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Total   int64  `json:"total"`
}

type CreateParams struct {
	Name                 *string    `json:"name"`
	Organizer            *string    `json:"organizer"`
	StartsAt             *time.Time `json:"starts_at"`
	EndsAt               *time.Time `json:"ends_at"`
	AvatarUrl            *string    `json:"avatar_url"`
	SiteUrl              *string    `json:"site_url"`
	CtftimeUrl           *string    `json:"ctftime_url"`
	RegistrationOpensAt  *time.Time `json:"registration_opens_at"`
	RegistrationClosesAt *time.Time `json:"registration_closes_at"`
	ScoreboardOpensAt    *time.Time `json:"scoreboard_opens_at"`
	ScoreboardClosesAt   *time.Time `json:"scoreboard_closes_at"`
	VpnUrl               *string    `json:"vpn_url"`
	VpnConfigUrl         *string    `json:"vpn_config_url"`
	AccessInstructions   *string    `json:"access_instructions"`
	AccessSecret         *string    `json:"access_secret"`
}

type UpdateParams struct {
	Name                 *string    `json:"name"`
	Organizer            *string    `json:"organizer"`
	StartsAt             *time.Time `json:"starts_at"`
	EndsAt               *time.Time `json:"ends_at"`
	AvatarUrl            *string    `json:"avatar_url"`
	SiteUrl              *string    `json:"site_url"`
	CtftimeUrl           *string    `json:"ctftime_url"`
	RegistrationOpensAt  *time.Time `json:"registration_opens_at"`
	RegistrationClosesAt *time.Time `json:"registration_closes_at"`
	ScoreboardOpensAt    *time.Time `json:"scoreboard_opens_at"`
	ScoreboardClosesAt   *time.Time `json:"scoreboard_closes_at"`
	VpnUrl               *string    `json:"vpn_url"`
	VpnConfigUrl         *string    `json:"vpn_config_url"`
	AccessInstructions   *string    `json:"access_instructions"`
	AccessSecret         *string    `json:"access_secret"`
}

type GameQuerier interface {
	CreateGame(ctx context.Context, arg db.CreateGameParams) (db.Game, error)
	GetGameByID(ctx context.Context, id int64) (db.Game, error)
	ListGames(ctx context.Context, arg db.ListGamesParams) ([]db.Game, error)
	CountGames(ctx context.Context) (int64, error)
	UpdateGame(ctx context.Context, arg db.UpdateGameParams) (db.Game, error)
	DeleteGame(ctx context.Context, id int64) error
	SetFinalized(ctx context.Context, arg db.SetFinalizedParams) (db.Game, error)
}

type GamesServiceQuerier interface {
	AddService(ctx context.Context, arg db.AddServiceParams) error
	RemoveService(ctx context.Context, arg db.RemoveServiceParams) error
	ListServicesByGame(ctx context.Context, gameID int64) ([]int64, error)
}

type ResultQuerier interface {
	ListResultsByGame(ctx context.Context, gameID int64) ([]db.Result, error)
}

type FinalResultQuerier interface {
	DeleteFinalResultsByGame(ctx context.Context, gameID int64) error
	InsertFinalResult(ctx context.Context, arg db.InsertFinalResultParams) (db.FinalResult, error)
	ListFinalResultsByGame(ctx context.Context, gameID int64) ([]db.FinalResult, error)
}

type TxRunner interface {
	RunInTx(ctx context.Context, fn func(queries *db.Queries) error) error
}

type Service struct {
	games        GameQuerier
	gamesSvc     GamesServiceQuerier
	results      ResultQuerier
	finalResults FinalResultQuerier
	tx           TxRunner
}

func NewService(games GameQuerier, gamesSvc GamesServiceQuerier, results ResultQuerier, finalResults FinalResultQuerier, tx TxRunner) *Service {
	return &Service{games: games, gamesSvc: gamesSvc, results: results, finalResults: finalResults, tx: tx}
}

type txQueriers struct {
	games        GameQuerier
	gamesSvc     GamesServiceQuerier
	results      ResultQuerier
	finalResults FinalResultQuerier
}

func (s *Service) txQ(q *db.Queries) *txQueriers {
	if q == nil {
		return &txQueriers{games: s.games, gamesSvc: s.gamesSvc, results: s.results, finalResults: s.finalResults}
	}
	return &txQueriers{games: q, gamesSvc: q, results: q, finalResults: q}
}

func validHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func validateURLs(params interface{ GetURLs() map[string]*string }) error {
	fields := make(map[string]string)
	urls := params.GetURLs()
	for field, val := range urls {
		if val != nil && *val != "" && !validHTTPURL(*val) {
			fields[field] = "must be a valid http(s) URL"
		}
	}
	if len(fields) > 0 {
		return errs.NewValidationError(fields)
	}
	return nil
}

type urlValidatable struct {
	siteUrl      *string
	ctftimeUrl   *string
	vpnUrl       *string
	vpnConfigUrl *string
}

func (u urlValidatable) GetURLs() map[string]*string {
	return map[string]*string{
		"site_url":       u.siteUrl,
		"ctftime_url":    u.ctftimeUrl,
		"vpn_url":        u.vpnUrl,
		"vpn_config_url": u.vpnConfigUrl,
	}
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*Game, error) {
	if params.Name == nil || *params.Name == "" {
		return nil, errs.NewValidationError(map[string]string{"name": "name is required"})
	}
	if err := validateURLs(urlValidatable{
		siteUrl:      params.SiteUrl,
		ctftimeUrl:   params.CtftimeUrl,
		vpnUrl:       params.VpnUrl,
		vpnConfigUrl: params.VpnConfigUrl,
	}); err != nil {
		return nil, err
	}
	if params.StartsAt != nil && params.EndsAt != nil && !params.EndsAt.After(*params.StartsAt) {
		return nil, errs.NewValidationError(map[string]string{"ends_at": "must be after starts_at"})
	}

	dbGame, err := s.games.CreateGame(ctx, db.CreateGameParams{
		Name:                 params.Name,
		Organizer:            params.Organizer,
		StartsAt:             timeToTimestamptz(params.StartsAt),
		EndsAt:               timeToTimestamptz(params.EndsAt),
		AvatarUrl:            params.AvatarUrl,
		SiteUrl:              params.SiteUrl,
		CtftimeUrl:           params.CtftimeUrl,
		Finalized:            false,
		FinalizedAt:          pgtype.Timestamptz{},
		RegistrationOpensAt:  timeToTimestamptz(params.RegistrationOpensAt),
		RegistrationClosesAt: timeToTimestamptz(params.RegistrationClosesAt),
		ScoreboardOpensAt:    timeToTimestamptz(params.ScoreboardOpensAt),
		ScoreboardClosesAt:   timeToTimestamptz(params.ScoreboardClosesAt),
		VpnUrl:               params.VpnUrl,
		VpnConfigUrl:         params.VpnConfigUrl,
		AccessInstructions:   params.AccessInstructions,
		AccessSecret:         params.AccessSecret,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	g := fromDB(dbGame)
	return &g, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Game, error) {
	dbGame, err := s.games.GetGameByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "game")
	}
	g := fromDB(dbGame)
	return &g, nil
}

func (s *Service) List(ctx context.Context, page, perPage int) (*GameListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

	items, err := s.games.ListGames(ctx, db.ListGamesParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	total, err := s.games.CountGames(ctx)
	if err != nil {
		return nil, err
	}

	result := &GameListResult{
		Items:   make([]Game, len(items)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, item := range items {
		result.Items[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, id int64, params UpdateParams) (*Game, error) {
	if err := validateURLs(urlValidatable{
		siteUrl:      params.SiteUrl,
		ctftimeUrl:   params.CtftimeUrl,
		vpnUrl:       params.VpnUrl,
		vpnConfigUrl: params.VpnConfigUrl,
	}); err != nil {
		return nil, err
	}
	existing, err := s.games.GetGameByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "game")
	}

	effectiveStartsAt := params.StartsAt
	if effectiveStartsAt == nil && existing.StartsAt.Valid {
		effectiveStartsAt = &existing.StartsAt.Time
	}
	effectiveEndsAt := params.EndsAt
	if effectiveEndsAt == nil && existing.EndsAt.Valid {
		effectiveEndsAt = &existing.EndsAt.Time
	}
	if effectiveStartsAt != nil && effectiveEndsAt != nil && !effectiveEndsAt.After(*effectiveStartsAt) {
		return nil, errs.NewValidationError(map[string]string{"ends_at": "must be after starts_at"})
	}

	dbGame, err := s.games.UpdateGame(ctx, db.UpdateGameParams{
		ID:                   id,
		Name:                 params.Name,
		Organizer:            params.Organizer,
		StartsAt:             timeToTimestamptz(params.StartsAt),
		EndsAt:               timeToTimestamptz(params.EndsAt),
		AvatarUrl:            params.AvatarUrl,
		SiteUrl:              params.SiteUrl,
		CtftimeUrl:           params.CtftimeUrl,
		RegistrationOpensAt:  timeToTimestamptz(params.RegistrationOpensAt),
		RegistrationClosesAt: timeToTimestamptz(params.RegistrationClosesAt),
		ScoreboardOpensAt:    timeToTimestamptz(params.ScoreboardOpensAt),
		ScoreboardClosesAt:   timeToTimestamptz(params.ScoreboardClosesAt),
		VpnUrl:               params.VpnUrl,
		VpnConfigUrl:         params.VpnConfigUrl,
		AccessInstructions:   params.AccessInstructions,
		AccessSecret:         params.AccessSecret,
	})
	if err != nil {
		return nil, mapNotFound(err, "game")
	}
	g := fromDB(dbGame)
	return &g, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.games.DeleteGame(ctx, id)
}

func (s *Service) AddService(ctx context.Context, gameID, serviceID int64) error {
	return s.gamesSvc.AddService(ctx, db.AddServiceParams{GameID: gameID, ServiceID: serviceID})
}

func (s *Service) RemoveService(ctx context.Context, gameID, serviceID int64) error {
	return s.gamesSvc.RemoveService(ctx, db.RemoveServiceParams{GameID: gameID, ServiceID: serviceID})
}

func (s *Service) ListServices(ctx context.Context, gameID int64) ([]int64, error) {
	return s.gamesSvc.ListServicesByGame(ctx, gameID)
}

func (s *Service) Finalize(ctx context.Context, gameID int64) (*Game, error) {
	game, err := s.games.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, mapNotFound(err, "game")
	}
	if game.Finalized {
		return nil, errs.ErrConflict
	}

	err = s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		_, err := tq.games.SetFinalized(ctx, db.SetFinalizedParams{ID: gameID, Finalized: true, FinalizedAt: now})
		if err != nil {
			return err
		}

		if err := tq.finalResults.DeleteFinalResultsByGame(ctx, gameID); err != nil {
			return err
		}

		results, err := tq.results.ListResultsByGame(ctx, gameID)
		if err != nil {
			return err
		}

		curRank := int32(1)
		var prevScore int32
		for i, r := range results {
			score := int32(0)
			if r.Score != nil {
				score = *r.Score
			}
			if i > 0 && score != prevScore {
				curRank = int32(i + 1)
			}
			prevScore = score
			pos := curRank
			_, err := tq.finalResults.InsertFinalResult(ctx, db.InsertFinalResultParams{
				GameID:   gameID,
				TeamID:   r.TeamID,
				Score:    score,
				Position: &pos,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.games.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	g := fromDB(updated)
	return &g, nil
}

func (s *Service) Unfinalize(ctx context.Context, gameID int64) (*Game, error) {
	game, err := s.games.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, mapNotFound(err, "game")
	}
	if !game.Finalized {
		return nil, errs.ErrConflict
	}

	err = s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		_, err := tq.games.SetFinalized(ctx, db.SetFinalizedParams{ID: gameID, Finalized: false, FinalizedAt: pgtype.Timestamptz{}})
		if err != nil {
			return err
		}
		return tq.finalResults.DeleteFinalResultsByGame(ctx, gameID)
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.games.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	g := fromDB(updated)
	return &g, nil
}

func fromDB(g db.Game) Game {
	now := time.Now()
	var startsAt *time.Time
	if g.StartsAt.Valid {
		startsAt = &g.StartsAt.Time
	}
	var endsAt *time.Time
	if g.EndsAt.Valid {
		endsAt = &g.EndsAt.Time
	}
	var finalizedAt *time.Time
	if g.FinalizedAt.Valid {
		finalizedAt = &g.FinalizedAt.Time
	}
	var regOpensAt *time.Time
	if g.RegistrationOpensAt.Valid {
		regOpensAt = &g.RegistrationOpensAt.Time
	}
	var regClosesAt *time.Time
	if g.RegistrationClosesAt.Valid {
		regClosesAt = &g.RegistrationClosesAt.Time
	}
	var scOpensAt *time.Time
	if g.ScoreboardOpensAt.Valid {
		scOpensAt = &g.ScoreboardOpensAt.Time
	}
	var scClosesAt *time.Time
	if g.ScoreboardClosesAt.Valid {
		scClosesAt = &g.ScoreboardClosesAt.Time
	}

	return Game{
		ID:                   g.ID,
		Name:                 g.Name,
		Organizer:            g.Organizer,
		StartsAt:             startsAt,
		EndsAt:               endsAt,
		CreatedAt:            g.CreatedAt,
		UpdatedAt:            g.UpdatedAt,
		AvatarUrl:            g.AvatarUrl,
		SiteUrl:              g.SiteUrl,
		CtftimeUrl:           g.CtftimeUrl,
		Finalized:            g.Finalized,
		FinalizedAt:          finalizedAt,
		RegistrationOpensAt:  regOpensAt,
		RegistrationClosesAt: regClosesAt,
		ScoreboardOpensAt:    scOpensAt,
		ScoreboardClosesAt:   scClosesAt,
		VpnUrl:               g.VpnUrl,
		VpnConfigUrl:         g.VpnConfigUrl,
		AccessInstructions:   g.AccessInstructions,
		AccessSecret:         g.AccessSecret,
		Status:               ComputeStatus(startsAt, endsAt, now),
		RegistrationStatus:   ComputeRegistrationStatus(regOpensAt, regClosesAt, now),
		ScoreboardStatusVal:  ComputeScoreboardStatus(scOpensAt, scClosesAt, now),
	}
}

func timeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func mapNotFound(err error, entity string) error {
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
