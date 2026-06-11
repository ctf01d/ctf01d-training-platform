package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/config"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/ctf01d/ctf01d-training-platform/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "import-rails error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	railsDBURL := os.Getenv("RAILS_DATABASE_URL")
	if railsDBURL == "" {
		return fmt.Errorf("RAILS_DATABASE_URL is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log, err := logger.New(cfg.Env, cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}
	defer logger.Sync(log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	railsStore, err := repository.NewStore(ctx, railsDBURL)
	if err != nil {
		return fmt.Errorf("connecting to Rails database: %w", err)
	}
	defer railsStore.Close()

	goStore, err := repository.NewStore(ctx, cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("connecting to Go database: %w", err)
	}
	defer goStore.Close()

	log.Info("importing users from Rails database")
	count, err := importUsers(ctx, railsStore.Queries, goStore.Queries, log)
	if err != nil {
		return fmt.Errorf("importing users: %w", err)
	}
	log.Info("users imported", zap.Int("count", count))

	log.Info("import completed successfully")
	log.Info("TODO: implement import for universities, teams, team_memberships, " +
		"team_membership_events, games, game_teams, services, results, final_results, writeups, games_services")
	return nil
}

type RailsUser struct {
	ID             int64
	UserName       string
	DisplayName    string
	Role           string
	Rating         int32
	AvatarUrl      *string
	PasswordDigest *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func MapRailsUserToParams(ru RailsUser) db.CreateUserParams {
	return db.CreateUserParams{
		UserName:       ru.UserName,
		DisplayName:    ru.DisplayName,
		Role:           ru.Role,
		Rating:         ru.Rating,
		AvatarUrl:      ru.AvatarUrl,
		PasswordDigest: ru.PasswordDigest,
	}
}

func importUsers(ctx context.Context, src *db.Queries, dst *db.Queries, log *zap.Logger) (int, error) {
	const batchSize = 100
	offset := 0
	total := 0

	for {
		users, err := src.ListUsers(ctx, db.ListUsersParams{
			Limit:  batchSize,
			Offset: int32(offset),
		})
		if err != nil {
			return total, fmt.Errorf("listing users at offset %d: %w", offset, err)
		}

		if len(users) == 0 {
			break
		}

		for _, u := range users {
			existing, err := dst.GetUserByUserName(ctx, u.UserName)
			if err == nil {
				log.Debug("user already exists, skipping",
					zap.String("user_name", u.UserName),
					zap.Int64("existing_id", existing.ID))
				continue
			}

			_, err = dst.CreateUser(ctx, db.CreateUserParams{
				UserName:       u.UserName,
				DisplayName:    u.DisplayName,
				Role:           u.Role,
				Rating:         u.Rating,
				AvatarUrl:      u.AvatarUrl,
				PasswordDigest: u.PasswordDigest,
			})
			if err != nil {
				return total, fmt.Errorf("creating user %s: %w", u.UserName, err)
			}
			total++
		}

		offset += batchSize
	}

	return total, nil
}
