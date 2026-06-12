package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

func TestStore_WithTx_Commit(t *testing.T) {
	store := NewTestStore(t)
	TruncateAll(t, store)
	ctx := context.Background()

	var result int32
	err := store.WithTx(ctx, func(q *db.Queries) error {
		var err error
		result, err = q.Ping(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("WithTx commit: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected ping result 1, got %d", result)
	}
}

func TestStore_WithTx_Rollback(t *testing.T) {
	store := NewTestStore(t)
	TruncateAll(t, store)
	ctx := context.Background()

	err := store.WithTx(ctx, func(q *db.Queries) error {
		return errors.New("forced error")
	})
	if err == nil {
		t.Fatal("expected error from WithTx rollback")
	}
}
