package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/pkg/logger"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func main() {
	if err := seed(); err != nil {
		fmt.Fprintf(os.Stderr, "seed error: %v\n", err)
		os.Exit(1)
	}
}

func seed() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log, err := logger.New(cfg.Env, cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}
	defer logger.Sync(log)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := repository.NewStore(ctx, cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer store.Close()

	q := store.Queries

	adminPassword := os.Getenv("SEED_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin12345"
	}

	adminUser, created, err := seedAdmin(ctx, q, adminPassword)
	if err != nil {
		return fmt.Errorf("seeding admin: %w", err)
	}
	logSeed(log, "admin user", adminUser.UserName, created)

	player1, created, err := seedUser(ctx, q, "player1", "Player One", "player", "player123")
	if err != nil {
		return fmt.Errorf("seeding player1: %w", err)
	}
	logSeed(log, "user", player1.UserName, created)

	player2, created, err := seedUser(ctx, q, "player2", "Player Two", "player", "player123")
	if err != nil {
		return fmt.Errorf("seeding player2: %w", err)
	}
	logSeed(log, "user", player2.UserName, created)

	uni1, created, err := seedUniversity(ctx, q, "MSU", "https://msu.ru")
	if err != nil {
		return fmt.Errorf("seeding university MSU: %w", err)
	}
	logSeed(log, "university", ptrStr(uni1.Name), created)

	uni2, created, err := seedUniversity(ctx, q, "MIPT", "https://mipt.ru")
	if err != nil {
		return fmt.Errorf("seeding university MIPT: %w", err)
	}
	logSeed(log, "university", ptrStr(uni2.Name), created)

	team1, created, err := seedTeam(ctx, q, "Team Alpha", "First test team", uni1.ID)
	if err != nil {
		return fmt.Errorf("seeding team Alpha: %w", err)
	}
	logSeed(log, "team", team1.Name, created)

	team2, created, err := seedTeam(ctx, q, "Team Beta", "Second test team", uni2.ID)
	if err != nil {
		return fmt.Errorf("seeding team Beta: %w", err)
	}
	logSeed(log, "team", team2.Name, created)

	if created {
		_, err = q.CreateTeamMembership(ctx, db.CreateTeamMembershipParams{
			TeamID: team1.ID,
			UserID: player1.ID,
			Role:   strPtr("owner"),
			Status: strPtr("approved"),
		})
		if err != nil {
			log.Warn("failed to create team1 membership", zap.Error(err))
		}

		_, err = q.CreateTeamMembership(ctx, db.CreateTeamMembershipParams{
			TeamID: team2.ID,
			UserID: player2.ID,
			Role:   strPtr("owner"),
			Status: strPtr("approved"),
		})
		if err != nil {
			log.Warn("failed to create team2 membership", zap.Error(err))
		}
	}

	game1, created, err := seedGame(ctx, q, "Test CTF 2026", "CTF01D",
		time.Now().Add(24*time.Hour), time.Now().Add(48*time.Hour))
	if err != nil {
		return fmt.Errorf("seeding game: %w", err)
	}
	logSeed(log, "game", ptrStr(game1.Name), created)

	log.Info("seed completed successfully")
	return nil
}

func seedAdmin(ctx context.Context, q *db.Queries, password string) (db.User, bool, error) {
	existing, err := q.GetUserByUserName(ctx, "admin")
	if err == nil {
		return existing, false, nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.User{}, false, fmt.Errorf("hashing password: %w", err)
	}

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		UserName:       "admin",
		DisplayName:    "Administrator",
		Role:           "admin",
		Rating:         0,
		PasswordDigest: &hash,
	})
	if err != nil {
		return db.User{}, false, err
	}
	return user, true, nil
}

func seedUser(ctx context.Context, q *db.Queries, userName, displayName, role, password string) (db.User, bool, error) {
	existing, err := q.GetUserByUserName(ctx, userName)
	if err == nil {
		return existing, false, nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.User{}, false, fmt.Errorf("hashing password: %w", err)
	}

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		UserName:       userName,
		DisplayName:    displayName,
		Role:           role,
		Rating:         0,
		PasswordDigest: &hash,
	})
	if err != nil {
		return db.User{}, false, err
	}
	return user, true, nil
}

func seedUniversity(ctx context.Context, q *db.Queries, name, siteURL string) (db.University, bool, error) {
	unis, err := q.ListUniversities(ctx, db.ListUniversitiesParams{Limit: 100, Offset: 0})
	if err != nil {
		return db.University{}, false, err
	}
	for _, u := range unis {
		if u.Name != nil && *u.Name == name {
			return u, false, nil
		}
	}

	uni, err := q.CreateUniversity(ctx, db.CreateUniversityParams{
		Name:    &name,
		SiteUrl: &siteURL,
	})
	if err != nil {
		return db.University{}, false, err
	}
	return uni, true, nil
}

func seedTeam(ctx context.Context, q *db.Queries, name, description string, universityID int64) (db.Team, bool, error) {
	teams, err := q.ListTeams(ctx, db.ListTeamsParams{Limit: 100, Offset: 0})
	if err != nil {
		return db.Team{}, false, err
	}
	for _, t := range teams {
		if t.Name == name {
			return t, false, nil
		}
	}

	team, err := q.CreateTeam(ctx, db.CreateTeamParams{
		Name:         name,
		Description:  &description,
		UniversityID: &universityID,
	})
	if err != nil {
		return db.Team{}, false, err
	}
	return team, true, nil
}

func seedGame(ctx context.Context, q *db.Queries, name, organizer string, startsAt, endsAt time.Time) (db.Game, bool, error) {
	games, err := q.ListGames(ctx, db.ListGamesParams{Limit: 100, Offset: 0})
	if err != nil {
		return db.Game{}, false, err
	}
	for _, g := range games {
		if g.Name != nil && *g.Name == name {
			return g, false, nil
		}
	}

	game, err := q.CreateGame(ctx, db.CreateGameParams{
		Name:      &name,
		Organizer: &organizer,
		StartsAt:  pgTz(startsAt),
		EndsAt:    pgTz(endsAt),
	})
	if err != nil {
		return db.Game{}, false, err
	}
	return game, true, nil
}

func logSeed(log *zap.Logger, entity, name string, created bool) {
	if created {
		log.Info("created", zap.String("entity", entity), zap.String("name", name))
	} else {
		log.Info("already exists", zap.String("entity", entity), zap.String("name", name))
	}
}

func strPtr(s string) *string  { return &s }
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func pgTz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}
