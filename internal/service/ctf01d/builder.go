package ctf01d

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type BuilderQuerier interface {
	GetGameByID(ctx context.Context, id int64) (db.Game, error)
	ListGameTeamsByGame(ctx context.Context, gameID int64) ([]db.GameTeam, error)
	ListServicesByGame(ctx context.Context, gameID int64) ([]int64, error)
	GetServiceByID(ctx context.Context, id int64) (db.Service, error)
	GetTeamByID(ctx context.Context, id int64) (db.Team, error)
}

type Builder struct {
	q          BuilderQuerier
	storageDir string
}

func NewBuilder(q BuilderQuerier) *Builder {
	return &Builder{q: q}
}

func (b *Builder) SetStorageDir(dir string) {
	b.storageDir = dir
}

type BuildResult struct {
	Game       GameParams
	Scoreboard ScoreboardParams
	Teams      []TeamParams
	Checkers   []CheckerParams
	Options    Options
	Warnings   []string
}

func (b *Builder) BuildParams(ctx context.Context, gameID int64, req Ctf01dExportRequest) (*BuildResult, error) {
	game, err := b.q.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("get game: %w", err)
	}

	gameTeams, err := b.q.ListGameTeamsByGame(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("list game teams: %w", err)
	}

	serviceIDs, err := b.q.ListServicesByGame(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	opts := buildOptions(req)

	gameParams := buildGameParams(game, req)

	teamParams, teamWarnings := b.buildTeamParams(ctx, gameTeams)
	var warnings []string
	warnings = append(warnings, teamWarnings...)

	checkerParams, checkerWarnings := b.buildCheckerParams(ctx, serviceIDs)
	warnings = append(warnings, checkerWarnings...)

	scoreParams := buildScoreboardParams(req)

	return &BuildResult{
		Game:       gameParams,
		Scoreboard: scoreParams,
		Teams:      teamParams,
		Checkers:   checkerParams,
		Options:    opts,
		Warnings:   warnings,
	}, nil
}

type Ctf01dExportRequest struct {
	Prefix           *string
	IncludeHtml      *bool
	HtmlSourcePath   *string
	IncludeCompose   *bool
	ComposeProject   *string
	Port             *int
	Htmlfolder       *string
	Random           *bool
	FlagTtlMin       *int
	BasicAttackCost  *int
	DefenceCost      *float64
	CoffeeBreakStart *time.Time
	CoffeeBreakEnd   *time.Time
}

type Ctf01dExportOptions struct {
	FlagTtlMin       int
	BasicAttackCost  int
	DefenceCost      float64
	Port             int
	IncludeHtml      bool
	HtmlSourcePath   string
	IncludeCompose   bool
	CoffeeBreakStart *time.Time
	CoffeeBreakEnd   *time.Time
	Warnings         []string
}

func (b *Builder) BuildOptions(ctx context.Context, gameID int64) (*Ctf01dExportOptions, error) {
	game, err := b.q.GetGameByID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("get game: %w", err)
	}

	gameTeams, err := b.q.ListGameTeamsByGame(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("list game teams: %w", err)
	}

	serviceIDs, err := b.q.ListServicesByGame(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	var warnings []string

	for _, gt := range gameTeams {
		if gt.IpAddress == nil || *gt.IpAddress == "" {
			team, terr := b.q.GetTeamByID(ctx, gt.TeamID)
			if terr == nil {
				warnings = append(warnings, fmt.Sprintf("team %q (id=%d) has no ip_address", team.Name, gt.TeamID))
			}
		}
	}

	for _, sid := range serviceIDs {
		svc, serr := b.q.GetServiceByID(ctx, sid)
		if serr == nil {
			if svc.CheckerLocalPath == nil || *svc.CheckerLocalPath == "" {
				warnings = append(warnings, fmt.Sprintf("service %q (id=%d) has no local checker archive", svc.Name, svc.ID))
			}
			if svc.ServiceLocalPath == nil || *svc.ServiceLocalPath == "" {
				warnings = append(warnings, fmt.Sprintf("service %q (id=%d) has no local service archive", svc.Name, svc.ID))
			}
		}
	}

	opts := &Ctf01dExportOptions{
		FlagTtlMin:      10,
		BasicAttackCost: 100,
		DefenceCost:     50.0,
		Port:            8080,
		IncludeHtml:     true,
		HtmlSourcePath:  "",
		IncludeCompose:  false,
		Warnings:        warnings,
	}

	if game.StartsAt.Valid && game.EndsAt.Valid {
		duration := game.EndsAt.Time.Sub(game.StartsAt.Time)
		hours := int(duration.Hours())
		if hours > 0 {
			ttl := hours * 60 / 10
			if ttl < 1 {
				ttl = 1
			}
			if ttl <= 25 {
				opts.FlagTtlMin = ttl
			}
		}
	}

	return opts, nil
}

func buildGameParams(game db.Game, req Ctf01dExportRequest) GameParams {
	gp := GameParams{
		ID:              strconv.FormatInt(game.ID, 10),
		Name:            strOrDefault(game.Name, "unnamed-game"),
		FlagTTLMin:      10,
		BasicAttackCost: 100,
		DefenceCost:     50.0,
	}

	if game.StartsAt.Valid {
		gp.StartUTC = game.StartsAt.Time.UTC()
	}
	if game.EndsAt.Valid {
		gp.EndUTC = game.EndsAt.Time.UTC()
	}

	if req.FlagTtlMin != nil && *req.FlagTtlMin > 0 {
		gp.FlagTTLMin = *req.FlagTtlMin
	}
	if req.BasicAttackCost != nil && *req.BasicAttackCost > 0 {
		gp.BasicAttackCost = *req.BasicAttackCost
	}
	if req.DefenceCost != nil {
		gp.DefenceCost = float64(*req.DefenceCost)
	}
	if req.CoffeeBreakStart != nil {
		gp.CoffeeBreakStartUTC = req.CoffeeBreakStart
	}
	if req.CoffeeBreakEnd != nil {
		gp.CoffeeBreakEndUTC = req.CoffeeBreakEnd
	}

	return gp
}

func (b *Builder) buildTeamParams(ctx context.Context, gameTeams []db.GameTeam) ([]TeamParams, []string) {
	var teams []TeamParams
	var warnings []string

	for _, gt := range gameTeams {
		team, err := b.q.GetTeamByID(ctx, gt.TeamID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("team id=%d not found", gt.TeamID))
			continue
		}

		tp := TeamParams{
			ID:      teamIDFromGameTeam(gt),
			Name:    team.Name,
			Active:  true,
			LogoURL: strPtrOrDefault(team.AvatarUrl, ""),
			LogoSrc: "",
		}

		if gt.IpAddress != nil {
			tp.IPAddress = *gt.IpAddress
		} else {
			warnings = append(warnings, fmt.Sprintf("team %q has no ip_address", team.Name))
		}

		if gt.Ctf01dOverrides != nil && string(gt.Ctf01dOverrides) != "{}" {
			var overrides map[string]string
			if err := json.Unmarshal(gt.Ctf01dOverrides, &overrides); err == nil {
				tp.Ctf01dExtra = overrides
			}
		}

		teams = append(teams, tp)
	}

	return teams, warnings
}

func (b *Builder) buildCheckerParams(ctx context.Context, serviceIDs []int64) ([]CheckerParams, []string) {
	var checkers []CheckerParams
	var warnings []string

	for _, sid := range serviceIDs {
		svc, err := b.q.GetServiceByID(ctx, sid)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("service id=%d not found", sid))
			continue
		}

		cp := CheckerParams{
			ID:                normalizeID(svc.Name),
			Name:              svc.Name,
			Enabled:           true,
			ScriptWait:        10,
			RoundSleep:        30,
			ScriptRel:         "./checker.py",
			BundlePath:        "",
			CheckerFromBundle: false,
		}

		if svc.CheckerLocalPath != nil && *svc.CheckerLocalPath != "" {
			cp.BundlePath = b.resolveStoragePath(*svc.CheckerLocalPath)
			cp.CheckerFromBundle = true
		} else {
			warnings = append(warnings, fmt.Sprintf("service %q has no local checker archive", svc.Name))
		}

		if svc.ServiceLocalPath != nil && *svc.ServiceLocalPath != "" {
			if cp.BundlePath == "" {
				cp.BundlePath = b.resolveStoragePath(*svc.ServiceLocalPath)
			}
		}

		if svc.Ctf01dTraining != nil && string(svc.Ctf01dTraining) != "{}" {
			var training map[string]interface{}
			if err := json.Unmarshal(svc.Ctf01dTraining, &training); err == nil {
				if v, ok := training["script_wait"]; ok {
					if f, ok := v.(float64); ok && f > 0 {
						cp.ScriptWait = int(f)
					}
				}
				if v, ok := training["round_sleep"]; ok {
					if f, ok := v.(float64); ok && f > 0 {
						cp.RoundSleep = int(f)
					}
				}
				if v, ok := training["script_rel"]; ok {
					if s, ok := v.(string); ok && s != "" {
						cp.ScriptRel = s
					}
				}
				if v, ok := training["enabled"]; ok {
					if b, ok := v.(bool); ok {
						cp.Enabled = b
					}
				}
			}
		}

		checkers = append(checkers, cp)
	}

	return checkers, warnings
}

func (b *Builder) resolveStoragePath(key string) string {
	if b.storageDir == "" {
		return key
	}
	return filepath.Join(b.storageDir, key)
}

func buildScoreboardParams(req Ctf01dExportRequest) ScoreboardParams {
	sp := ScoreboardParams{
		Port:       8080,
		HtmlFolder: "./html",
		Random:     false,
	}
	if req.Port != nil && *req.Port > 0 {
		sp.Port = *req.Port
	}
	if req.Htmlfolder != nil && *req.Htmlfolder != "" {
		sp.HtmlFolder = *req.Htmlfolder
	}
	if req.Random != nil {
		sp.Random = *req.Random
	}
	return sp
}

func buildOptions(req Ctf01dExportRequest) Options {
	opts := Options{
		Prefix:         "ctf01d_package",
		IncludeHTML:    true,
		HtmlSourcePath: "",
		IncludeCompose: false,
		ComposeProject: "ctf01d",
	}
	if req.Prefix != nil && *req.Prefix != "" {
		opts.Prefix = *req.Prefix
	}
	if req.IncludeHtml != nil {
		opts.IncludeHTML = *req.IncludeHtml
	}
	if req.HtmlSourcePath != nil {
		opts.HtmlSourcePath = *req.HtmlSourcePath
	}
	if req.IncludeCompose != nil {
		opts.IncludeCompose = *req.IncludeCompose
	}
	if req.ComposeProject != nil && *req.ComposeProject != "" {
		opts.ComposeProject = *req.ComposeProject
	}
	return opts
}

func teamIDFromGameTeam(gt db.GameTeam) string {
	if gt.Ctf01dID != nil && *gt.Ctf01dID != "" {
		return *gt.Ctf01dID
	}
	return fmt.Sprintf("team_%d", gt.TeamID)
}

func strOrDefault(s *string, def string) string {
	if s != nil && *s != "" {
		return *s
	}
	return def
}

func strPtrOrDefault(s *string, def string) string {
	if s != nil && *s != "" {
		return *s
	}
	return def
}
