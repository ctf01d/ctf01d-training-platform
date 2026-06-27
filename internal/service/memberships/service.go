package memberships

import (
	"context"
	"fmt"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type Membership struct {
	ID        int64     `json:"id"`
	TeamID    int64     `json:"team_id"`
	UserID    int64     `json:"user_id"`
	Role      *string   `json:"role"`
	Status    *string   `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MembershipListResult struct {
	Items   []Membership `json:"items"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
	Total   int64        `json:"total"`
}

type MembershipEvent struct {
	ID         int64     `json:"id"`
	TeamID     int64     `json:"team_id"`
	UserID     int64     `json:"user_id"`
	ActorID    *int32    `json:"actor_id"`
	Action     string    `json:"action"`
	FromRole   *string   `json:"from_role"`
	ToRole     *string   `json:"to_role"`
	FromStatus *string   `json:"from_status"`
	ToStatus   *string   `json:"to_status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type EventListResult struct {
	Items   []MembershipEvent `json:"items"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
	Total   int64             `json:"total"`
}

var managingRoles = map[string]bool{
	memRoleOwner:       true,
	memRoleCaptain:     true,
	memRoleViceCaptain: true,
}

const (
	minInt32 = -1 << 31
	maxInt32 = 1<<31 - 1

	memRoleOwner       = "owner"
	memRoleCaptain     = "captain"
	memRoleViceCaptain = "vice_captain"
	memRolePlayer      = "player"
	memRoleGuest       = "guest"

	memStatusPending  = "pending"
	memStatusApproved = "approved"
	memStatusRejected = "rejected"

	memFieldRole = "role"

	memActionInvite   = "invite"
	memActionAccepted = "accepted"
	memActionRemoved  = "removed"

	msgMembershipNotPending = "membership is not pending"
	fieldStatus             = "status"
)

type Querier interface {
	GetTeamMembershipByID(ctx context.Context, id int64) (db.TeamMembership, error)
	ListTeamMemberships(ctx context.Context, arg db.ListTeamMembershipsParams) ([]db.TeamMembership, error)
	ListTeamMembershipsByTeam(ctx context.Context, teamID int64) ([]db.TeamMembership, error)
	ListTeamMembershipsByUser(ctx context.Context, userID int64) ([]db.TeamMembership, error)
	CountTeamMemberships(ctx context.Context) (int64, error)
	UpdateTeamMembership(ctx context.Context, arg db.UpdateTeamMembershipParams) (db.TeamMembership, error)
	UpdateMembershipStatus(ctx context.Context, arg db.UpdateMembershipStatusParams) (db.TeamMembership, error)
	UpdateMembershipRole(ctx context.Context, arg db.UpdateMembershipRoleParams) (db.TeamMembership, error)
	DeleteTeamMembership(ctx context.Context, id int64) error
	GetMembership(ctx context.Context, arg db.GetMembershipParams) (db.TeamMembership, error)
	CountApprovedManagers(ctx context.Context, teamID int64) (int64, error)
	CreateTeamMembership(ctx context.Context, arg db.CreateTeamMembershipParams) (db.TeamMembership, error)
}

type EventQuerier interface {
	CreateEvent(ctx context.Context, arg db.CreateEventParams) (db.TeamMembershipEvent, error)
	ListEventsByTeam(ctx context.Context, arg db.ListEventsByTeamParams) ([]db.TeamMembershipEvent, error)
	CountEventsByTeam(ctx context.Context, teamID int64) (int64, error)
	GetLatestEventForMember(ctx context.Context, arg db.GetLatestEventForMemberParams) (db.TeamMembershipEvent, error)
}

type TeamQuerier interface {
	GetTeamByID(ctx context.Context, id int64) (db.Team, error)
	GetTeamByCaptain(ctx context.Context, captainID *int32) (db.Team, error)
	SetCaptain(ctx context.Context, arg db.SetCaptainParams) (db.Team, error)
	ClearCaptain(ctx context.Context, id int64) (db.Team, error)
}

type TxRunner interface {
	RunInTx(ctx context.Context, fn func(queries *db.Queries) error) error
}

type Service struct {
	q      Querier
	events EventQuerier
	teams  TeamQuerier
	tx     TxRunner
}

func NewService(q Querier, events EventQuerier, teams TeamQuerier, tx TxRunner) *Service {
	return &Service{q: q, events: events, teams: teams, tx: tx}
}

type txQueriers struct {
	q      Querier
	events EventQuerier
	teams  TeamQuerier
}

func (s *Service) txQ(q *db.Queries) *txQueriers {
	if q == nil {
		return &txQueriers{q: s.q, events: s.events, teams: s.teams}
	}
	return &txQueriers{q: q, events: q, teams: q}
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Membership, error) {
	mem, err := s.q.GetTeamMembershipByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	m := fromDB(mem)
	return &m, nil
}

func (s *Service) ListByTeam(ctx context.Context, teamID int64) ([]Membership, error) {
	items, err := s.q.ListTeamMembershipsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	result := make([]Membership, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) ListByUser(ctx context.Context, userID int64) ([]Membership, error) {
	items, err := s.q.ListTeamMembershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]Membership, len(items))
	for i, item := range items {
		result[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, page, perPage int) (*MembershipListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset, err := int32FromInt64(int64(page-1) * int64(perPage))
	if err != nil {
		return nil, err
	}
	limit, err := int32FromInt64(int64(perPage))
	if err != nil {
		return nil, err
	}

	items, err := s.q.ListTeamMemberships(ctx, db.ListTeamMembershipsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	total, err := s.q.CountTeamMemberships(ctx)
	if err != nil {
		return nil, err
	}

	result := &MembershipListResult{
		Items:   make([]Membership, len(items)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, item := range items {
		result.Items[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) ListEvents(ctx context.Context, teamID int64, page, perPage int) (*EventListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset, err := int32FromInt64(int64(page-1) * int64(perPage))
	if err != nil {
		return nil, err
	}
	limit, err := int32FromInt64(int64(perPage))
	if err != nil {
		return nil, err
	}

	items, err := s.events.ListEventsByTeam(ctx, db.ListEventsByTeamParams{
		TeamID: teamID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	total, err := s.events.CountEventsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}

	result := &EventListResult{
		Items:   make([]MembershipEvent, len(items)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, item := range items {
		result.Items[i] = eventFromDB(item)
	}
	return result, nil
}

func (s *Service) CreateDirect(ctx context.Context, arg db.CreateTeamMembershipParams) (*Membership, error) {
	mem, err := s.q.CreateTeamMembership(ctx, arg)
	if err != nil {
		return nil, mapDBError(err)
	}
	m := fromDB(mem)
	return &m, nil
}

func (s *Service) Update(ctx context.Context, id int64, role, status string) (*Membership, error) {
	if role != "" {
		updated, err := s.q.UpdateMembershipRole(ctx, db.UpdateMembershipRoleParams{ID: id, Role: &role})
		if err != nil {
			return nil, mapNotFound(err)
		}
		m := fromDB(updated)
		return &m, nil
	}
	if status != "" {
		updated, err := s.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: id, Status: &status})
		if err != nil {
			return nil, mapNotFound(err)
		}
		m := fromDB(updated)
		return &m, nil
	}

	mem, err := s.q.GetTeamMembershipByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	m := fromDB(mem)
	return &m, nil
}

func (s *Service) Delete(ctx context.Context, id int64, actorID int64, globalRole string) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		mem, err := tq.q.GetTeamMembershipByID(ctx, id)
		if err != nil {
			return mapNotFound(err)
		}
		if err := s.canManageMembership(ctx, mem.TeamID, actorID, globalRole, tq.q); err != nil {
			return err
		}

		// Do not allow removing the last approved owner, mirroring SetRole.
		if mem.Role != nil && *mem.Role == memRoleOwner && mem.Status != nil && *mem.Status == memStatusApproved {
			members, err := tq.q.ListTeamMembershipsByTeam(ctx, mem.TeamID)
			if err != nil {
				return err
			}
			ownerCount := int64(0)
			for _, m := range members {
				if m.Role != nil && *m.Role == memRoleOwner && m.Status != nil && *m.Status == memStatusApproved {
					ownerCount++
				}
			}
			if ownerCount <= 1 {
				return errs.NewValidationError(map[string]string{memFieldRole: "cannot remove the last owner"})
			}
		}

		if err := tq.q.DeleteTeamMembership(ctx, id); err != nil {
			return err
		}

		// Removing an approved captain must clear the team's captain reference,
		// mirroring SetRole; otherwise teams.captain_id dangles to a non-member.
		if mem.Role != nil && *mem.Role == memRoleCaptain && mem.Status != nil && *mem.Status == memStatusApproved {
			if _, err := tq.teams.ClearCaptain(ctx, mem.TeamID); err != nil {
				return err
			}
		}

		actorID32, err := int32Ptr(actorID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    actorID32,
			Action:     memActionRemoved,
			FromRole:   mem.Role,
			ToRole:     mem.Role,
			FromStatus: mem.Status,
			ToStatus:   mem.Status,
		})
		return err
	})
}

func (s *Service) Approve(ctx context.Context, membershipID int64, actorID int64, globalRole string) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		mem, err := tq.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err)
		}
		if mem.Status == nil || *mem.Status != memStatusPending {
			return errs.NewValidationError(map[string]string{fieldStatus: msgMembershipNotPending})
		}
		if err := s.canManageMembership(ctx, mem.TeamID, actorID, globalRole, tq.q); err != nil {
			return err
		}

		approved := memStatusApproved
		updated, err := tq.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &approved})
		if err != nil {
			return err
		}

		actorID32, err := int32Ptr(actorID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    actorID32,
			Action:     memStatusApproved,
			FromRole:   mem.Role,
			ToRole:     updated.Role,
			FromStatus: mem.Status,
			ToStatus:   &approved,
		})
		return err
	})
}

func (s *Service) Reject(ctx context.Context, membershipID int64, actorID int64, globalRole string) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		mem, err := tq.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err)
		}
		if mem.Status == nil || *mem.Status != memStatusPending {
			return errs.NewValidationError(map[string]string{fieldStatus: msgMembershipNotPending})
		}
		if err := s.canManageMembership(ctx, mem.TeamID, actorID, globalRole, tq.q); err != nil {
			return err
		}

		rejected := memStatusRejected
		updated, err := tq.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &rejected})
		if err != nil {
			return err
		}

		actorID32, err := int32Ptr(actorID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    actorID32,
			Action:     memStatusRejected,
			FromRole:   mem.Role,
			ToRole:     updated.Role,
			FromStatus: mem.Status,
			ToStatus:   &rejected,
		})
		return err
	})
}

func (s *Service) Accept(ctx context.Context, membershipID int64, userID int64) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		mem, err := tq.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err)
		}
		if mem.UserID != userID {
			return errs.ErrForbidden
		}
		if mem.Status == nil || *mem.Status != memStatusPending {
			return errs.NewValidationError(map[string]string{fieldStatus: msgMembershipNotPending})
		}

		evt, err := tq.events.GetLatestEventForMember(ctx, db.GetLatestEventForMemberParams{
			TeamID: mem.TeamID,
			UserID: mem.UserID,
		})
		if err != nil {
			if repository.IsNoRows(err) {
				return errs.ErrForbidden
			}
			return fmt.Errorf("checking membership source: %w", err)
		}
		if evt.Action != memActionInvite {
			return errs.ErrForbidden
		}

		approved := memStatusApproved
		updated, err := tq.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &approved})
		if err != nil {
			return err
		}

		userID32, err := int32Ptr(userID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    userID32,
			Action:     memActionAccepted,
			FromRole:   mem.Role,
			ToRole:     updated.Role,
			FromStatus: mem.Status,
			ToStatus:   &approved,
		})
		return err
	})
}

func (s *Service) Decline(ctx context.Context, membershipID int64, userID int64) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		mem, err := tq.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err)
		}
		if mem.UserID != userID {
			return errs.ErrForbidden
		}
		if mem.Status == nil || *mem.Status != memStatusPending {
			return errs.NewValidationError(map[string]string{fieldStatus: msgMembershipNotPending})
		}

		evt, err := tq.events.GetLatestEventForMember(ctx, db.GetLatestEventForMemberParams{
			TeamID: mem.TeamID,
			UserID: mem.UserID,
		})
		if err != nil {
			if repository.IsNoRows(err) {
				return errs.ErrForbidden
			}
			return fmt.Errorf("checking membership source: %w", err)
		}
		if evt.Action != memActionInvite {
			return errs.ErrForbidden
		}

		rejected := memStatusRejected
		updated, err := tq.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &rejected})
		if err != nil {
			return err
		}

		userID32, err := int32Ptr(userID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    userID32,
			Action:     "declined",
			FromRole:   mem.Role,
			ToRole:     updated.Role,
			FromStatus: mem.Status,
			ToStatus:   &rejected,
		})
		return err
	})
}

func (s *Service) SetRole(ctx context.Context, membershipID int64, newRole string, actorID int64, globalRole string) error {
	if !managingRoles[newRole] && newRole != memRolePlayer && newRole != memRoleGuest {
		return errs.NewValidationError(map[string]string{memFieldRole: "invalid role"})
	}

	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		mem, err := tq.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err)
		}
		if err := s.canManageMembership(ctx, mem.TeamID, actorID, globalRole, tq.q); err != nil {
			return err
		}

		oldRole := ""
		if mem.Role != nil {
			oldRole = *mem.Role
		}

		if oldRole == memRoleOwner && newRole != memRoleOwner {
			ownerCount := int64(0)
			members, err := tq.q.ListTeamMembershipsByTeam(ctx, mem.TeamID)
			if err != nil {
				return err
			}
			for _, m := range members {
				if m.Role != nil && *m.Role == memRoleOwner && m.Status != nil && *m.Status == memStatusApproved {
					ownerCount++
				}
			}
			if ownerCount <= 1 {
				return errs.NewValidationError(map[string]string{memFieldRole: "cannot remove the last owner"})
			}
		}

		updated, err := tq.q.UpdateMembershipRole(ctx, db.UpdateMembershipRoleParams{ID: membershipID, Role: &newRole})
		if err != nil {
			return err
		}

		actorID32, err := int32Ptr(actorID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    actorID32,
			Action:     "set_role",
			FromRole:   &oldRole,
			ToRole:     &newRole,
			FromStatus: mem.Status,
			ToStatus:   updated.Status,
		})
		if err != nil {
			return err
		}

		if newRole == memRoleCaptain {
			captainID, err := int32FromInt64(mem.UserID)
			if err != nil {
				return err
			}
			_, existingTeam, err := s.getTeamByCaptainSafe(ctx, &captainID, tq.teams)
			if err != nil {
				return err
			}
			if existingTeam != nil && existingTeam.ID != mem.TeamID {
				return fmt.Errorf("%w: user %d is already captain of team %d", errs.ErrConflict, mem.UserID, existingTeam.ID)
			}
			if existingTeam == nil || existingTeam.ID == mem.TeamID {
				_, err = tq.teams.SetCaptain(ctx, db.SetCaptainParams{ID: mem.TeamID, CaptainID: &captainID})
				if err != nil {
					return err
				}
			}
		}

		if oldRole == memRoleCaptain && newRole != memRoleCaptain {
			_, err = tq.teams.ClearCaptain(ctx, mem.TeamID)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) canManageMembership(ctx context.Context, teamID, actorID int64, globalRole string, q Querier) error {
	if globalRole == "admin" {
		return nil
	}
	mem, err := q.GetMembership(ctx, db.GetMembershipParams{TeamID: teamID, UserID: actorID})
	if err != nil {
		return errs.ErrForbidden
	}
	if mem.Status == nil || *mem.Status != memStatusApproved {
		return errs.ErrForbidden
	}
	if mem.Role == nil || !managingRoles[*mem.Role] {
		return errs.ErrForbidden
	}
	return nil
}

func (s *Service) getTeamByCaptainSafe(ctx context.Context, captainID *int32, teams TeamQuerier) (found bool, team *db.Team, err error) {
	t, e := teams.GetTeamByCaptain(ctx, captainID)
	if e != nil {
		if isNoRows(e) {
			return false, nil, nil
		}
		return false, nil, e
	}
	return true, &t, nil
}

func fromDB(m db.TeamMembership) Membership {
	return Membership{
		ID:        m.ID,
		TeamID:    m.TeamID,
		UserID:    m.UserID,
		Role:      m.Role,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func eventFromDB(e db.TeamMembershipEvent) MembershipEvent {
	return MembershipEvent{
		ID:         e.ID,
		TeamID:     e.TeamID,
		UserID:     e.UserID,
		ActorID:    e.ActorID,
		Action:     e.Action,
		FromRole:   e.FromRole,
		ToRole:     e.ToRole,
		FromStatus: e.FromStatus,
		ToStatus:   e.ToStatus,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

func int32FromInt64(v int64) (int32, error) {
	if v < minInt32 || v > maxInt32 {
		return 0, errs.NewValidationError(map[string]string{"id": "must fit int32"})
	}
	return int32(v), nil
}

func int32Ptr(v int64) (*int32, error) {
	i, err := int32FromInt64(v)
	if err != nil {
		return nil, err
	}
	return &i, nil
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

func isNoRows(err error) bool {
	return repository.IsNoRows(err)
}
