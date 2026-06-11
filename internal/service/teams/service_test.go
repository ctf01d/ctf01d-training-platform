package teams

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5"
)

type mockTeamQuerier struct {
	teams     map[int64]db.Team
	nextID    int64
	byCaptain map[int32]int64
}

type mockMembershipQuerier struct {
	members map[string]db.TeamMembership
	nextID  int64
}

type mockEventQuerier struct {
	events []db.TeamMembershipEvent
	nextID int64
}

type mockTxRunner struct {
	teamQ  *mockTeamQuerier
	memQ   *mockMembershipQuerier
	eventQ *mockEventQuerier
}

func (m *mockTxRunner) RunInTx(_ context.Context, fn func() error) error {
	return fn()
}

func newMocks() (*mockTeamQuerier, *mockMembershipQuerier, *mockEventQuerier, *mockTxRunner) {
	tq := &mockTeamQuerier{teams: make(map[int64]db.Team), nextID: 1, byCaptain: make(map[int32]int64)}
	mq := &mockMembershipQuerier{members: make(map[string]db.TeamMembership), nextID: 1}
	eq := &mockEventQuerier{nextID: 1}
	tx := &mockTxRunner{teamQ: tq, memQ: mq, eventQ: eq}
	return tq, mq, eq, tx
}

func (m *mockTeamQuerier) CreateTeam(_ context.Context, arg db.CreateTeamParams) (db.Team, error) {
	id := m.nextID
	m.nextID++
	now := time.Now()
	t := db.Team{
		ID: id, Name: arg.Name, Description: arg.Description,
		Website: arg.Website, AvatarUrl: arg.AvatarUrl,
		UniversityID: arg.UniversityID, CreatedAt: now, UpdatedAt: now,
	}
	m.teams[id] = t
	return t, nil
}

func (m *mockTeamQuerier) GetTeamByID(_ context.Context, id int64) (db.Team, error) {
	t, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return t, nil
}

func (m *mockTeamQuerier) GetTeamByCaptain(_ context.Context, captainID *int32) (db.Team, error) {
	if captainID == nil {
		return db.Team{}, pgx.ErrNoRows
	}
	id, ok := m.byCaptain[*captainID]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	return m.teams[id], nil
}

func (m *mockTeamQuerier) ListTeams(_ context.Context, arg db.ListTeamsParams) ([]db.Team, error) {
	var result []db.Team
	for i := int32(0); i < arg.Limit; i++ {
		idx := arg.Offset + i + 1
		if t, ok := m.teams[int64(idx)]; ok {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTeamQuerier) CountTeams(_ context.Context) (int64, error) {
	return int64(len(m.teams)), nil
}

func (m *mockTeamQuerier) UpdateTeam(_ context.Context, arg db.UpdateTeamParams) (db.Team, error) {
	t, ok := m.teams[arg.ID]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	t.Name = arg.Name
	if arg.Description != nil {
		t.Description = arg.Description
	}
	if arg.Website != nil {
		t.Website = arg.Website
	}
	if arg.AvatarUrl != nil {
		t.AvatarUrl = arg.AvatarUrl
	}
	if arg.UniversityID != nil {
		t.UniversityID = arg.UniversityID
	}
	t.UpdatedAt = time.Now()
	m.teams[arg.ID] = t
	return t, nil
}

func (m *mockTeamQuerier) SetCaptain(_ context.Context, arg db.SetCaptainParams) (db.Team, error) {
	t, ok := m.teams[arg.ID]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	if arg.CaptainID != nil {
		m.byCaptain[*arg.CaptainID] = arg.ID
	}
	t.CaptainID = arg.CaptainID
	t.UpdatedAt = time.Now()
	m.teams[arg.ID] = t
	return t, nil
}

func (m *mockTeamQuerier) ClearCaptain(_ context.Context, id int64) (db.Team, error) {
	t, ok := m.teams[id]
	if !ok {
		return db.Team{}, pgx.ErrNoRows
	}
	if t.CaptainID != nil {
		delete(m.byCaptain, *t.CaptainID)
	}
	t.CaptainID = nil
	t.UpdatedAt = time.Now()
	m.teams[id] = t
	return t, nil
}

func (m *mockTeamQuerier) DeleteTeam(_ context.Context, id int64) error {
	if t, ok := m.teams[id]; ok {
		if t.CaptainID != nil {
			delete(m.byCaptain, *t.CaptainID)
		}
		delete(m.teams, id)
	}
	return nil
}

func memKey(teamID, userID int64) string {
	return fmt.Sprintf("%d:%d", teamID, userID)
}

func (m *mockMembershipQuerier) CreateTeamMembership(_ context.Context, arg db.CreateTeamMembershipParams) (db.TeamMembership, error) {
	key := memKey(arg.TeamID, arg.UserID)
	if _, ok := m.members[key]; ok {
		return db.TeamMembership{}, fmt.Errorf("duplicate key value violates unique constraint")
	}
	id := m.nextID
	m.nextID++
	now := time.Now()
	mem := db.TeamMembership{
		ID: id, TeamID: arg.TeamID, UserID: arg.UserID,
		Role: arg.Role, Status: arg.Status,
		CreatedAt: now, UpdatedAt: now,
	}
	m.members[key] = mem
	return mem, nil
}

func (m *mockMembershipQuerier) GetMembership(_ context.Context, arg db.GetMembershipParams) (db.TeamMembership, error) {
	mem, ok := m.members[memKey(arg.TeamID, arg.UserID)]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	return mem, nil
}

func (m *mockMembershipQuerier) CountApprovedManagers(_ context.Context, teamID int64) (int64, error) {
	var count int64
	for _, mem := range m.members {
		if mem.TeamID == teamID && mem.Status != nil && *mem.Status == "approved" && mem.Role != nil {
			if managingRoles[*mem.Role] {
				count++
			}
		}
	}
	return count, nil
}

func (m *mockEventQuerier) CreateEvent(_ context.Context, arg db.CreateEventParams) (db.TeamMembershipEvent, error) {
	id := m.nextID
	m.nextID++
	now := time.Now()
	e := db.TeamMembershipEvent{
		ID: id, TeamID: arg.TeamID, UserID: arg.UserID,
		ActorID: arg.ActorID, Action: arg.Action,
		FromRole: arg.FromRole, ToRole: arg.ToRole,
		FromStatus: arg.FromStatus, ToStatus: arg.ToStatus,
		CreatedAt: now, UpdatedAt: now,
	}
	m.events = append(m.events, e)
	return e, nil
}

func TestCreate_Success(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	team, err := svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if team.ID != 1 {
		t.Errorf("ID = %d, want 1", team.ID)
	}
	if team.Name != "Team Alpha" {
		t.Errorf("Name = %q, want %q", team.Name, "Team Alpha")
	}

	mem, err := mq.GetMembership(context.Background(), db.GetMembershipParams{TeamID: 1, UserID: 1})
	if err != nil {
		t.Fatalf("membership not found: %v", err)
	}
	if mem.Role == nil || *mem.Role != "owner" {
		t.Errorf("membership role = %v, want owner", mem.Role)
	}
	if mem.Status == nil || *mem.Status != "approved" {
		t.Errorf("membership status = %v, want approved", mem.Status)
	}

	if len(eq.events) != 1 {
		t.Fatalf("events = %d, want 1", len(eq.events))
	}
	if eq.events[0].Action != "created" {
		t.Errorf("event action = %q, want created", eq.events[0].Action)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	_, err := svc.Create(context.Background(), 1, CreateParams{Name: ""})
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	team, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if team.Name != "Team Alpha" {
		t.Errorf("Name = %q, want Team Alpha", team.Name)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	_, err := svc.GetByID(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	for i := 0; i < 5; i++ {
		svc.Create(context.Background(), int64(i+1), CreateParams{Name: "Team " + string(rune('A'+i))})
	}

	result, err := svc.List(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(result.Items))
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
}

func TestCanManage_Admin(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	err := svc.CanManage(context.Background(), 1, 999, "admin")
	if err != nil {
		t.Errorf("admin should be able to manage, got %v", err)
	}
}

func TestCanManage_Owner(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	err := svc.CanManage(context.Background(), 1, 1, "player")
	if err != nil {
		t.Errorf("owner should be able to manage, got %v", err)
	}
}

func TestCanManage_NotMember(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	err := svc.CanManage(context.Background(), 1, 999, "player")
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestCanManage_PlayerRole(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})
	role := "player"
	status := "approved"
	mq.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		TeamID: 1, UserID: 2, Role: &role, Status: &status,
	})

	err := svc.CanManage(context.Background(), 1, 2, "player")
	if err != errs.ErrForbidden {
		t.Errorf("player role should not manage, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	desc := "Updated description"
	name := "Team Beta"
	team, err := svc.Update(context.Background(), 1, UpdateParams{
		Name:        &name,
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if team.Name != "Team Beta" {
		t.Errorf("Name = %q, want Team Beta", team.Name)
	}
}

func TestDelete(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(context.Background(), 1)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRequestJoin(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	err := svc.RequestJoin(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("RequestJoin: %v", err)
	}

	mem, err := mq.GetMembership(context.Background(), db.GetMembershipParams{TeamID: 1, UserID: 2})
	if err != nil {
		t.Fatalf("membership not found: %v", err)
	}
	if mem.Role == nil || *mem.Role != "guest" {
		t.Errorf("role = %v, want guest", mem.Role)
	}
	if mem.Status == nil || *mem.Status != "pending" {
		t.Errorf("status = %v, want pending", mem.Status)
	}

	found := false
	for _, e := range eq.events {
		if e.Action == "join_request" && e.UserID == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("join_request event not found")
	}
}

func TestRequestJoin_Duplicate(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})
	svc.RequestJoin(context.Background(), 1, 2)

	err := svc.RequestJoin(context.Background(), 1, 2)
	if err != errs.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestInvite_Success(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})

	err := svc.Invite(context.Background(), 1, 1, 2)
	if err != nil {
		t.Fatalf("Invite: %v", err)
	}

	mem, err := mq.GetMembership(context.Background(), db.GetMembershipParams{TeamID: 1, UserID: 2})
	if err != nil {
		t.Fatalf("membership not found: %v", err)
	}
	if mem.Role == nil || *mem.Role != "player" {
		t.Errorf("role = %v, want player", mem.Role)
	}
	if mem.Status == nil || *mem.Status != "pending" {
		t.Errorf("status = %v, want pending", mem.Status)
	}
}

func TestInvite_NonManager(t *testing.T) {
	tq, mq, eq, tx := newMocks()
	svc := NewService(tq, mq, eq, tx)

	svc.Create(context.Background(), 1, CreateParams{Name: "Team Alpha"})
	role := "player"
	status := "approved"
	mq2 := tx.memQ
	mq2.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		TeamID: 1, UserID: 2, Role: &role, Status: &status,
	})

	err := svc.Invite(context.Background(), 1, 2, 3)
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}
