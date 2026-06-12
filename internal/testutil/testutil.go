package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx database/sql driver for goose
	"github.com/pressly/goose/v3"

	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
)

func NewTestStore(t *testing.T) *repository.Store {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	store, err := repository.NewStore(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}

	migrateDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		store.Close()
		t.Fatalf("opening migration connection: %v", err)
	}
	defer migrateDB.Close()

	if err := goose.Up(migrateDB, "../../migrations"); err != nil {
		store.Close()
		t.Fatalf("applying goose migrations: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store
}

func TruncateAll(t *testing.T, store *repository.Store) {
	t.Helper()

	tables := []string{
		"writeups",
		"final_results",
		"results",
		"game_teams",
		"games_services",
		"services",
		"games",
		"team_membership_events",
		"team_memberships",
		"teams",
		"universities",
		"users",
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)
		if _, err := store.Pool.Exec(context.Background(), query); err != nil {
			t.Fatalf("truncating table %s: %v", table, err)
		}
	}
}
