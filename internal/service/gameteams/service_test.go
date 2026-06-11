package gameteams

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5"
)

type mockQuerier struct {
	items  map[int64]db.GameTeam
	nextID int64
}

type mockTxRunner struct{}

func newMocks() (*mockQuerier, *mockTxRunner) {
	q := &mockQuerier{items: make(map[int64]db.GameTeam), nextID: 1}
	tx := &mockTxRunner{}
	return q, tx
}

func (m *mockTxRunner) RunInTx(_ context.Context, fn func(*db.Queries) error) error {
	return fn(nil)
}

func (m *mockQuerier) CreateGameTeam(_ context.Context, arg db.CreateGameTeamParams) (db.GameTeam, error) {
	id := m.nextID
	m.nextID++
	now := time.Now()
	gt := db.GameTeam{
		ID: id, GameID: arg.GameID, TeamID: arg.TeamID,
		IpAddress: arg.IpAddress, Ctf01dID: arg.Ctf01dID,
		Ctf01dOverrides: arg.Ctf01dOverrides, TeamType: arg.TeamType,
		Order: arg.Order, CreatedAt: now, UpdatedAt: now,
	}
	m.items[id] = gt
	return gt, nil
}

func (m *mockQuerier) GetGameTeamByID(_ context.Context, id int64) (db.GameTeam, error) {
	gt, ok := m.items[id]
	if !ok {
		return db.GameTeam{}, pgx.ErrNoRows
	}
	return gt, nil
}

func (m *mockQuerier) ListGameTeamsByGame(_ context.Context, gameID int64) ([]db.GameTeam, error) {
	var result []db.GameTeam
	for _, gt := range m.items {
		if gt.GameID == gameID {
			result = append(result, gt)
		}
	}
	return result, nil
}

func (m *mockQuerier) UpdateGameTeam(_ context.Context, arg db.UpdateGameTeamParams) (db.GameTeam, error) {
	gt, ok := m.items[arg.ID]
	if !ok {
		return db.GameTeam{}, pgx.ErrNoRows
	}
	if arg.IpAddress != nil {
		gt.IpAddress = arg.IpAddress
	}
	if arg.Ctf01dID != nil {
		gt.Ctf01dID = arg.Ctf01dID
	}
	if arg.Order != nil {
		gt.Order = *arg.Order
	}
	gt.UpdatedAt = time.Now()
	m.items[arg.ID] = gt
	return gt, nil
}

func (m *mockQuerier) DeleteGameTeam(_ context.Context, id int64) error {
	delete(m.items, id)
	return nil
}

func (m *mockQuerier) UpdateGameTeamOrder(_ context.Context, arg db.UpdateGameTeamOrderParams) error {
	gt, ok := m.items[arg.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	gt.Order = arg.Order
	m.items[arg.ID] = gt
	return nil
}

func ptrStr(v string) *string { return &v }

func TestCreate_Success(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	ip := "10.0.0.1"
	gt, err := svc.Create(context.Background(), CreateParams{
		GameID:    1,
		TeamID:    1,
		IpAddress: &ip,
		Order:     1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gt.ID != 1 {
		t.Errorf("ID = %d, want 1", gt.ID)
	}
}

func TestCreate_DefaultOverrides(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	gt, err := svc.Create(context.Background(), CreateParams{
		GameID: 1, TeamID: 1, Order: 0,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if string(gt.Ctf01dOverrides) != "{}" {
		t.Errorf("Ctf01dOverrides = %s, want {}", string(gt.Ctf01dOverrides))
	}
}

func TestGetByID_Success(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Order: 0})

	gt, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if gt.GameID != 1 {
		t.Errorf("GameID = %d, want 1", gt.GameID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	_, err := svc.GetByID(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListByGame(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Order: 1})
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 2, Order: 2})
	svc.Create(context.Background(), CreateParams{GameID: 2, TeamID: 3, Order: 1})

	items, err := svc.ListByGame(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListByGame: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}

func TestUpdate(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Order: 0})

	newIP := "10.0.0.2"
	orderVal := int32(5)
	gt, err := svc.Update(context.Background(), 1, UpdateParams{IpAddress: &newIP, Order: &orderVal})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if *gt.IpAddress != "10.0.0.2" {
		t.Errorf("IpAddress = %v, want 10.0.0.2", gt.IpAddress)
	}
}

func TestDelete(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Order: 0})

	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(context.Background(), 1)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestReorder(t *testing.T) {
	q, tx := newMocks()
	svc := NewService(q, tx)

	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 1, Order: 1})
	svc.Create(context.Background(), CreateParams{GameID: 1, TeamID: 2, Order: 2})

	err := svc.Reorder(context.Background(), 1, []ReorderItem{
		{ID: 1, Order: 2},
		{ID: 2, Order: 1},
	})
	if err != nil {
		t.Fatalf("Reorder: %v", err)
	}

	gt1, _ := svc.GetByID(context.Background(), 1)
	gt2, _ := svc.GetByID(context.Background(), 2)
	if gt1.Order != 2 {
		t.Errorf("gt1.Order = %d, want 2", gt1.Order)
	}
	if gt2.Order != 1 {
		t.Errorf("gt2.Order = %d, want 1", gt2.Order)
	}
}

var _ = json.RawMessage{}
