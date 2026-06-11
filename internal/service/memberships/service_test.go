package memberships

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	"github.com/jackc/pgx/v5"
)

type mockQuerier struct {
	members  map[int64]db.TeamMembership
	nextID   int64
	byTeamUser map[string]int64
}

type mockEventQuerier struct {
	events []db.TeamMembershipEvent
	nextID int64
}

type mockTeamQuerier struct {
	teams     map[int64]db.Team
	byCaptain map[int32]int64
}

type mockTxRunner struct{}

func (m *mockTxRunner) RunInTx(_ context.Context, fn func() error) error {
	return fn()
}

func newMocks() (*mockQuerier, *mockEventQuerier, *mockTeamQuerier, *mockTxRunner) {
	mq := &mockQuerier{members: make(map[int64]db.TeamMembership), nextID: 1, byTeamUser: make(map[string]int64)}
	eq := &mockEventQuerier{nextID: 1}
	tq := &mockTeamQuerier{teams: make(map[int64]db.Team), byCaptain: make(map[int32]int64)}
	return mq, eq, tq, &mockTxRunner{}
}

func memKey(teamID, userID int64) string {
	return fmt.Sprintf("%d:%d", teamID, userID)
}

func (m *mockQuerier) CreateTeamMembership(_ context.Context, arg db.CreateTeamMembershipParams) (db.TeamMembership, error) {
	key := memKey(arg.TeamID, arg.UserID)
	if _, ok := m.byTeamUser[key]; ok {
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
	m.members[id] = mem
	m.byTeamUser[key] = id
	return mem, nil
}

func (m *mockQuerier) GetTeamMembershipByID(_ context.Context, id int64) (db.TeamMembership, error) {
	mem, ok := m.members[id]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	return mem, nil
}

func (m *mockQuerier) GetMembership(_ context.Context, arg db.GetMembershipParams) (db.TeamMembership, error) {
	id, ok := m.byTeamUser[memKey(arg.TeamID, arg.UserID)]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	return m.members[id], nil
}

func (m *mockQuerier) ListTeamMemberships(_ context.Context, arg db.ListTeamMembershipsParams) ([]db.TeamMembership, error) {
	var result []db.TeamMembership
	for i := int32(0); i < arg.Limit; i++ {
		idx := arg.Offset + i + 1
		if mem, ok := m.members[int64(idx)]; ok {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListTeamMembershipsByTeam(_ context.Context, teamID int64) ([]db.TeamMembership, error) {
	var result []db.TeamMembership
	for _, mem := range m.members {
		if mem.TeamID == teamID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListTeamMembershipsByUser(_ context.Context, userID int64) ([]db.TeamMembership, error) {
	var result []db.TeamMembership
	for _, mem := range m.members {
		if mem.UserID == userID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockQuerier) CountTeamMemberships(_ context.Context) (int64, error) {
	return int64(len(m.members)), nil
}

func (m *mockQuerier) UpdateTeamMembership(_ context.Context, arg db.UpdateTeamMembershipParams) (db.TeamMembership, error) {
	mem, ok := m.members[arg.ID]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	if arg.Role != nil {
		mem.Role = arg.Role
	}
	if arg.Status != nil {
		mem.Status = arg.Status
	}
	mem.UpdatedAt = time.Now()
	m.members[arg.ID] = mem
	return mem, nil
}

func (m *mockQuerier) UpdateMembershipStatus(_ context.Context, arg db.UpdateMembershipStatusParams) (db.TeamMembership, error) {
	mem, ok := m.members[arg.ID]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	mem.Status = arg.Status
	mem.UpdatedAt = time.Now()
	m.members[arg.ID] = mem
	key := memKey(mem.TeamID, mem.UserID)
	m.byTeamUser[key] = arg.ID
	return mem, nil
}

func (m *mockQuerier) UpdateMembershipRole(_ context.Context, arg db.UpdateMembershipRoleParams) (db.TeamMembership, error) {
	mem, ok := m.members[arg.ID]
	if !ok {
		return db.TeamMembership{}, pgx.ErrNoRows
	}
	mem.Role = arg.Role
	mem.UpdatedAt = time.Now()
	m.members[arg.ID] = mem
	return mem, nil
}

func (m *mockQuerier) DeleteTeamMembership(_ context.Context, id int64) error {
	if mem, ok := m.members[id]; ok {
		delete(m.byTeamUser, memKey(mem.TeamID, mem.UserID))
		delete(m.members, id)
	}
	return nil
}

func (m *mockQuerier) CountApprovedManagers(_ context.Context, teamID int64) (int64, error) {
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

func (m *mockEventQuerier) ListEventsByTeam(_ context.Context, arg db.ListEventsByTeamParams) ([]db.TeamMembershipEvent, error) {
	var result []db.TeamMembershipEvent
	for _, e := range m.events {
		if e.TeamID == arg.TeamID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockEventQuerier) CountEventsByTeam(_ context.Context, teamID int64) (int64, error) {
	var count int64
	for _, e := range m.events {
		if e.TeamID == teamID {
			count++
		}
	}
	return count, nil
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

func seedOwner(mq *mockQuerier, eq *mockEventQuerier, teamID, userID int64) int64 {
	role := "owner"
	status := "approved"
	mem, _ := mq.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		TeamID: teamID, UserID: userID, Role: &role, Status: &status,
	})
	return mem.ID
}

func seedPendingMember(mq *mockQuerier, eq *mockEventQuerier, teamID, userID int64, role string) int64 {
	status := "pending"
	mem, _ := mq.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		TeamID: teamID, UserID: userID, Role: &role, Status: &status,
	})
	return mem.ID
}

func TestGetByID_Success(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	memID := seedOwner(mq, eq, 1, 1)
	mem, err := svc.GetByID(context.Background(), memID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if mem.TeamID != 1 || mem.UserID != 1 {
		t.Errorf("TeamID=%d UserID=%d", mem.TeamID, mem.UserID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	_, err := svc.GetByID(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestApprove_Success(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	ownerMemID := seedOwner(mq, eq, 1, 10)
	_ = ownerMemID
	pendingMemID := seedPendingMember(mq, eq, 1, 20, "player")

	err := svc.Approve(context.Background(), pendingMemID, 10, "player")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}

	mem, _ := mq.GetTeamMembershipByID(context.Background(), pendingMemID)
	if mem.Status == nil || *mem.Status != "approved" {
		t.Errorf("status = %v, want approved", mem.Status)
	}

	found := false
	for _, e := range eq.events {
		if e.Action == "approved" && e.UserID == 20 {
			found = true
			break
		}
	}
	if !found {
		t.Error("approved event not found")
	}
}

func TestApprove_NotManager(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	seedOwner(mq, eq, 1, 10)
	pendingMemID := seedPendingMember(mq, eq, 1, 20, "player")
	playerRole := "player"
	approvedStatus := "approved"
	mq.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		TeamID: 1, UserID: 30, Role: &playerRole, Status: &approvedStatus,
	})

	err := svc.Approve(context.Background(), pendingMemID, 30, "player")
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestApprove_Admin(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	pendingMemID := seedPendingMember(mq, eq, 1, 20, "player")

	err := svc.Approve(context.Background(), pendingMemID, 99, "admin")
	if err != nil {
		t.Fatalf("Admin Approve: %v", err)
	}
}

func TestReject_Success(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	seedOwner(mq, eq, 1, 10)
	pendingMemID := seedPendingMember(mq, eq, 1, 20, "player")

	err := svc.Reject(context.Background(), pendingMemID, 10, "player")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}

	mem, _ := mq.GetTeamMembershipByID(context.Background(), pendingMemID)
	if mem.Status == nil || *mem.Status != "rejected" {
		t.Errorf("status = %v, want rejected", mem.Status)
	}
}

func TestAccept_Success(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	pendingMemID := seedPendingMember(mq, eq, 1, 20, "player")

	err := svc.Accept(context.Background(), pendingMemID, 20)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}

	mem, _ := mq.GetTeamMembershipByID(context.Background(), pendingMemID)
	if mem.Status == nil || *mem.Status != "approved" {
		t.Errorf("status = %v, want approved", mem.Status)
	}
}

func TestAccept_WrongUser(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	pendingMemID := seedPendingMember(mq, eq, 1, 20, "player")

	err := svc.Accept(context.Background(), pendingMemID, 99)
	if err != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestDecline_Success(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	pendingMemID := seedPendingMember(mq, eq, 1, 20, "player")

	err := svc.Decline(context.Background(), pendingMemID, 20)
	if err != nil {
		t.Fatalf("Decline: %v", err)
	}

	mem, _ := mq.GetTeamMembershipByID(context.Background(), pendingMemID)
	if mem.Status == nil || *mem.Status != "rejected" {
		t.Errorf("status = %v, want rejected", mem.Status)
	}
}

func TestSetRole_ToCaptain(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	tq.teams[1] = db.Team{ID: 1, Name: "Team A", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	seedOwner(mq, eq, 1, 10)
	playerRole := "player"
	approvedStatus := "approved"
	mem, _ := mq.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		TeamID: 1, UserID: 20, Role: &playerRole, Status: &approvedStatus,
	})

	err := svc.SetRole(context.Background(), mem.ID, "captain", 10, "player")
	if err != nil {
		t.Fatalf("SetRole to captain: %v", err)
	}

	updated, _ := mq.GetTeamMembershipByID(context.Background(), mem.ID)
	if updated.Role == nil || *updated.Role != "captain" {
		t.Errorf("role = %v, want captain", updated.Role)
	}

	team, _ := tq.GetTeamByID(context.Background(), 1)
	if team.CaptainID == nil || *team.CaptainID != 20 {
		t.Errorf("team.CaptainID = %v, want 20", team.CaptainID)
	}
}

func TestSetRole_RemoveLastOwner(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	tq.teams[1] = db.Team{ID: 1, Name: "Team A", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	ownerMemID := seedOwner(mq, eq, 1, 10)

	err := svc.SetRole(context.Background(), ownerMemID, "player", 10, "player")
	if err == nil {
		t.Error("expected error when removing last owner")
	}
}

func TestSetRole_CanRemoveOwnerIfMultiple(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	tq.teams[1] = db.Team{ID: 1, Name: "Team A", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	ownerMemID1 := seedOwner(mq, eq, 1, 10)
	seedOwner(mq, eq, 1, 11)

	err := svc.SetRole(context.Background(), ownerMemID1, "player", 11, "player")
	if err != nil {
		t.Fatalf("SetRole: %v", err)
	}

	updated, _ := mq.GetTeamMembershipByID(context.Background(), ownerMemID1)
	if updated.Role == nil || *updated.Role != "player" {
		t.Errorf("role = %v, want player", updated.Role)
	}
}

func TestSetRole_CaptainToOther_ClearsCaptain(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	captainID := int32(20)
	tq.teams[1] = db.Team{ID: 1, Name: "Team A", CaptainID: &captainID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	tq.byCaptain[captainID] = 1

	seedOwner(mq, eq, 1, 10)
	captainRole := "captain"
	approvedStatus := "approved"
	mem, _ := mq.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		TeamID: 1, UserID: 20, Role: &captainRole, Status: &approvedStatus,
	})

	err := svc.SetRole(context.Background(), mem.ID, "player", 10, "player")
	if err != nil {
		t.Fatalf("SetRole captain to player: %v", err)
	}

	team, _ := tq.GetTeamByID(context.Background(), 1)
	if team.CaptainID != nil {
		t.Errorf("team.CaptainID = %v, want nil", team.CaptainID)
	}
}

func TestSetRole_InvalidRole(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	ownerMemID := seedOwner(mq, eq, 1, 10)

	err := svc.SetRole(context.Background(), ownerMemID, "superadmin", 10, "player")
	if _, ok := err.(*errs.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestListByTeam(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	seedOwner(mq, eq, 1, 10)
	seedOwner(mq, eq, 1, 11)

	items, err := svc.ListByTeam(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListByTeam: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}

func TestListEvents(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	seedOwner(mq, eq, 1, 10)
	seedPendingMember(mq, eq, 1, 20, "player")
	svc.Approve(context.Background(), 2, 10, "player")

	result, err := svc.ListEvents(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(result.Items))
	}
	if result.Items[0].Action != "approved" {
		t.Errorf("action = %q, want approved", result.Items[0].Action)
	}
}

func TestDelete(t *testing.T) {
	mq, eq, tq, tx := newMocks()
	svc := NewService(mq, eq, tq, tx)

	memID := seedOwner(mq, eq, 1, 10)
	err := svc.Delete(context.Background(), memID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(context.Background(), memID)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
