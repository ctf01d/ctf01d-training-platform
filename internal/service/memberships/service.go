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
	"owner":        true,
	"captain":      true,
	"vice_captain": true,
}

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
}

type TeamQuerier interface {
	GetTeamByID(ctx context.Context, id int64) (db.Team, error)
	GetTeamByCaptain(ctx context.Context, captainID *int32) (db.Team, error)
	SetCaptain(ctx context.Context, arg db.SetCaptainParams) (db.Team, error)
	ClearCaptain(ctx context.Context, id int64) (db.Team, error)
}

type TxRunner interface {
	RunInTx(ctx context.Context, fn func() error) error
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

func (s *Service) GetByID(ctx context.Context, id int64) (*Membership, error) {
	mem, err := s.q.GetTeamMembershipByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "membership")
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

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

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

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

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
			return nil, mapNotFound(err, "membership")
		}
		m := fromDB(updated)
		return &m, nil
	}
	if status != "" {
		updated, err := s.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: id, Status: &status})
		if err != nil {
			return nil, mapNotFound(err, "membership")
		}
		m := fromDB(updated)
		return &m, nil
	}

	mem, err := s.q.GetTeamMembershipByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "membership")
	}
	m := fromDB(mem)
	return &m, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.q.DeleteTeamMembership(ctx, id)
}

func (s *Service) Approve(ctx context.Context, membershipID int64, actorID int64, globalRole string) error {
	return s.tx.RunInTx(ctx, func() error {
		mem, err := s.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err, "membership")
		}
		if mem.Status == nil || *mem.Status != "pending" {
			return errs.NewValidationError(map[string]string{"status": "membership is not pending"})
		}
		if err := s.canManageMembership(ctx, mem.TeamID, actorID, globalRole); err != nil {
			return err
		}

		approved := "approved"
		updated, err := s.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &approved})
		if err != nil {
			return err
		}

		_, err = s.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    int32Ptr(actorID),
			Action:     "approved",
			FromRole:   mem.Role,
			ToRole:     updated.Role,
			FromStatus: mem.Status,
			ToStatus:   &approved,
		})
		return err
	})
}

func (s *Service) Reject(ctx context.Context, membershipID int64, actorID int64, globalRole string) error {
	return s.tx.RunInTx(ctx, func() error {
		mem, err := s.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err, "membership")
		}
		if mem.Status == nil || *mem.Status != "pending" {
			return errs.NewValidationError(map[string]string{"status": "membership is not pending"})
		}
		if err := s.canManageMembership(ctx, mem.TeamID, actorID, globalRole); err != nil {
			return err
		}

		rejected := "rejected"
		updated, err := s.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &rejected})
		if err != nil {
			return err
		}

		_, err = s.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    int32Ptr(actorID),
			Action:     "rejected",
			FromRole:   mem.Role,
			ToRole:     updated.Role,
			FromStatus: mem.Status,
			ToStatus:   &rejected,
		})
		return err
	})
}

func (s *Service) Accept(ctx context.Context, membershipID int64, userID int64) error {
	return s.tx.RunInTx(ctx, func() error {
		mem, err := s.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err, "membership")
		}
		if mem.UserID != userID {
			return errs.ErrForbidden
		}
		if mem.Status == nil || *mem.Status != "pending" {
			return errs.NewValidationError(map[string]string{"status": "membership is not pending"})
		}

		approved := "approved"
		updated, err := s.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &approved})
		if err != nil {
			return err
		}

		_, err = s.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    int32Ptr(userID),
			Action:     "accepted",
			FromRole:   mem.Role,
			ToRole:     updated.Role,
			FromStatus: mem.Status,
			ToStatus:   &approved,
		})
		return err
	})
}

func (s *Service) Decline(ctx context.Context, membershipID int64, userID int64) error {
	return s.tx.RunInTx(ctx, func() error {
		mem, err := s.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err, "membership")
		}
		if mem.UserID != userID {
			return errs.ErrForbidden
		}
		if mem.Status == nil || *mem.Status != "pending" {
			return errs.NewValidationError(map[string]string{"status": "membership is not pending"})
		}

		rejected := "rejected"
		updated, err := s.q.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{ID: membershipID, Status: &rejected})
		if err != nil {
			return err
		}

		_, err = s.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    int32Ptr(userID),
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
	if !managingRoles[newRole] && newRole != "player" && newRole != "guest" {
		return errs.NewValidationError(map[string]string{"role": "invalid role"})
	}

	return s.tx.RunInTx(ctx, func() error {
		mem, err := s.q.GetTeamMembershipByID(ctx, membershipID)
		if err != nil {
			return mapNotFound(err, "membership")
		}
		if err := s.canManageMembership(ctx, mem.TeamID, actorID, globalRole); err != nil {
			return err
		}

		oldRole := ""
		if mem.Role != nil {
			oldRole = *mem.Role
		}

		if oldRole == "owner" && newRole != "owner" {
			ownerCount := int64(0)
			members, err := s.q.ListTeamMembershipsByTeam(ctx, mem.TeamID)
			if err != nil {
				return err
			}
			for _, m := range members {
				if m.Role != nil && *m.Role == "owner" && m.Status != nil && *m.Status == "approved" {
					ownerCount++
				}
			}
			if ownerCount <= 1 {
				return errs.NewValidationError(map[string]string{"role": "cannot remove the last owner"})
			}
		}

		updated, err := s.q.UpdateMembershipRole(ctx, db.UpdateMembershipRoleParams{ID: membershipID, Role: &newRole})
		if err != nil {
			return err
		}

		_, err = s.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     mem.TeamID,
			UserID:     mem.UserID,
			ActorID:    int32Ptr(actorID),
			Action:     "set_role",
			FromRole:   &oldRole,
			ToRole:     &newRole,
			FromStatus: mem.Status,
			ToStatus:   updated.Status,
		})
		if err != nil {
			return err
		}

		if newRole == "captain" {
			captainID := int32(mem.UserID)
			_, existingTeam, err := s.getTeamByCaptainSafe(ctx, &captainID)
			if err != nil {
				return err
			}
			if existingTeam != nil && existingTeam.ID != mem.TeamID {
				return fmt.Errorf("%w: user %d is already captain of team %d", errs.ErrConflict, mem.UserID, existingTeam.ID)
			}
			if existingTeam == nil || existingTeam.ID == mem.TeamID {
				_, err = s.teams.SetCaptain(ctx, db.SetCaptainParams{ID: mem.TeamID, CaptainID: &captainID})
				if err != nil {
					return err
				}
			}
		}

		if oldRole == "captain" && newRole != "captain" {
			_, err = s.teams.ClearCaptain(ctx, mem.TeamID)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) canManageMembership(ctx context.Context, teamID, actorID int64, globalRole string) error {
	if globalRole == "admin" {
		return nil
	}
	mem, err := s.q.GetMembership(ctx, db.GetMembershipParams{TeamID: teamID, UserID: actorID})
	if err != nil {
		return errs.ErrForbidden
	}
	if mem.Status == nil || *mem.Status != "approved" {
		return errs.ErrForbidden
	}
	if mem.Role == nil || !managingRoles[*mem.Role] {
		return errs.ErrForbidden
	}
	return nil
}

func (s *Service) getTeamByCaptainSafe(ctx context.Context, captainID *int32) (found bool, team *db.Team, err error) {
	t, e := s.teams.GetTeamByCaptain(ctx, captainID)
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

func int32Ptr(v int64) *int32 {
	i := int32(v)
	return &i
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

func isNoRows(err error) bool {
	return repository.IsNoRows(err)
}
