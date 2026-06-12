package teams

import (
	"context"
	"fmt"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type Team struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Website      *string   `json:"website"`
	AvatarUrl    *string   `json:"avatar_url"`
	CaptainID    *int32    `json:"captain_id"`
	UniversityID *int64    `json:"university_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TeamListResult struct {
	Items   []Team `json:"items"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Total   int64  `json:"total"`
}

type CreateParams struct {
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Website      *string `json:"website"`
	AvatarUrl    *string `json:"avatar_url"`
	UniversityID *int64  `json:"university_id"`
}

type UpdateParams struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Website      *string `json:"website"`
	AvatarUrl    *string `json:"avatar_url"`
	UniversityID *int64  `json:"university_id"`
}

var managingRoles = map[string]bool{
	"owner":        true,
	"captain":      true,
	"vice_captain": true,
}

type TeamQuerier interface {
	CreateTeam(ctx context.Context, arg db.CreateTeamParams) (db.Team, error)
	GetTeamByID(ctx context.Context, id int64) (db.Team, error)
	GetTeamByCaptain(ctx context.Context, captainID *int32) (db.Team, error)
	ListTeams(ctx context.Context, arg db.ListTeamsParams) ([]db.Team, error)
	CountTeams(ctx context.Context) (int64, error)
	UpdateTeam(ctx context.Context, arg db.UpdateTeamParams) (db.Team, error)
	SetCaptain(ctx context.Context, arg db.SetCaptainParams) (db.Team, error)
	ClearCaptain(ctx context.Context, id int64) (db.Team, error)
	DeleteTeam(ctx context.Context, id int64) error
}

type MembershipQuerier interface {
	CreateTeamMembership(ctx context.Context, arg db.CreateTeamMembershipParams) (db.TeamMembership, error)
	GetMembership(ctx context.Context, arg db.GetMembershipParams) (db.TeamMembership, error)
	UpdateMembershipStatus(ctx context.Context, arg db.UpdateMembershipStatusParams) (db.TeamMembership, error)
	CountApprovedManagers(ctx context.Context, teamID int64) (int64, error)
}

type EventQuerier interface {
	CreateEvent(ctx context.Context, arg db.CreateEventParams) (db.TeamMembershipEvent, error)
}

type TxRunner interface {
	RunInTx(ctx context.Context, fn func(queries *db.Queries) error) error
}

type Service struct {
	teams   TeamQuerier
	members MembershipQuerier
	events  EventQuerier
	tx      TxRunner
}

func NewService(teams TeamQuerier, members MembershipQuerier, events EventQuerier, tx TxRunner) *Service {
	return &Service{teams: teams, members: members, events: events, tx: tx}
}

type txQueriers struct {
	teams   TeamQuerier
	members MembershipQuerier
	events  EventQuerier
}

func (s *Service) txQ(q *db.Queries) *txQueriers {
	if q == nil {
		return &txQueriers{teams: s.teams, members: s.members, events: s.events}
	}
	return &txQueriers{teams: q, members: q, events: q}
}

func (s *Service) Create(ctx context.Context, creatorID int64, params CreateParams) (*Team, error) {
	if params.Name == "" {
		return nil, errs.NewValidationError(map[string]string{"name": "is required"})
	}

	var result *Team
	err := s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		team, err := tq.teams.CreateTeam(ctx, db.CreateTeamParams{
			Name:         params.Name,
			Description:  params.Description,
			Website:      params.Website,
			AvatarUrl:    params.AvatarUrl,
			UniversityID: params.UniversityID,
		})
		if err != nil {
			return mapDBError(err)
		}

		ownerRole := "owner"
		approved := "approved"
		_, err = tq.members.CreateTeamMembership(ctx, db.CreateTeamMembershipParams{
			TeamID: team.ID,
			UserID: creatorID,
			Role:   &ownerRole,
			Status: &approved,
		})
		if err != nil {
			return err
		}

		actorID, err := int32PtrFromInt64(creatorID)
		if err != nil {
			return err
		}
		action := "created"
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:     team.ID,
			UserID:     creatorID,
			ActorID:    actorID,
			Action:     action,
			FromRole:   nil,
			ToRole:     &ownerRole,
			FromStatus: nil,
			ToStatus:   &approved,
		})
		if err != nil {
			return err
		}

		t := fromDB(team)
		result = &t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Team, error) {
	team, err := s.teams.GetTeamByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	t := fromDB(team)
	return &t, nil
}

func (s *Service) List(ctx context.Context, page, perPage int) (*TeamListResult, error) {
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

	items, err := s.teams.ListTeams(ctx, db.ListTeamsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	total, err := s.teams.CountTeams(ctx)
	if err != nil {
		return nil, err
	}

	result := &TeamListResult{
		Items:   make([]Team, len(items)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, item := range items {
		result.Items[i] = fromDB(item)
	}
	return result, nil
}

func (s *Service) CanManage(ctx context.Context, teamID, userID int64, globalRole string) error {
	return s.canManageWithQuerier(ctx, teamID, userID, globalRole, s.members)
}

func (s *Service) canManageWithQuerier(ctx context.Context, teamID, userID int64, globalRole string, members MembershipQuerier) error {
	if globalRole == "admin" {
		return nil
	}
	membership, err := members.GetMembership(ctx, db.GetMembershipParams{
		TeamID: teamID,
		UserID: userID,
	})
	if err != nil {
		return errs.ErrForbidden
	}
	if membership.Status == nil || *membership.Status != "approved" {
		return errs.ErrForbidden
	}
	if membership.Role == nil || !managingRoles[*membership.Role] {
		return errs.ErrForbidden
	}
	return nil
}

func (s *Service) Update(ctx context.Context, id int64, params UpdateParams) (*Team, error) {
	if params.Name != nil && *params.Name == "" {
		return nil, errs.NewValidationError(map[string]string{"name": "must be non-empty"})
	}
	name := ""
	if params.Name != nil {
		name = *params.Name
	} else {
		existing, err := s.teams.GetTeamByID(ctx, id)
		if err != nil {
			return nil, mapNotFound(err)
		}
		name = existing.Name
	}
	team, err := s.teams.UpdateTeam(ctx, db.UpdateTeamParams{
		ID:           id,
		Name:         name,
		Description:  params.Description,
		Website:      params.Website,
		AvatarUrl:    params.AvatarUrl,
		UniversityID: params.UniversityID,
	})
	if err != nil {
		return nil, mapNotFound(err)
	}
	t := fromDB(team)
	return &t, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.teams.DeleteTeam(ctx, id)
}

func (s *Service) RequestJoin(ctx context.Context, teamID, userID int64) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)

		existing, err := tq.members.GetMembership(ctx, db.GetMembershipParams{
			TeamID: teamID,
			UserID: userID,
		})
		if err == nil && existing.Status != nil && *existing.Status != "rejected" {
			return errs.ErrConflict
		}

		role := "guest"
		status := "pending"
		action := "join_request"

		if err == nil {
			_, err = tq.members.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{
				ID:     existing.ID,
				Status: &status,
			})
		} else {
			_, err = tq.members.CreateTeamMembership(ctx, db.CreateTeamMembershipParams{
				TeamID: teamID,
				UserID: userID,
				Role:   &role,
				Status: &status,
			})
		}
		if err != nil {
			return mapDBError(err)
		}

		actorID, err := int32PtrFromInt64(userID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:   teamID,
			UserID:   userID,
			ActorID:  actorID,
			Action:   action,
			ToRole:   &role,
			ToStatus: &status,
		})
		return err
	})
}

func (s *Service) Invite(ctx context.Context, teamID, inviterID, inviteeID int64, globalRole string) error {
	return s.tx.RunInTx(ctx, func(q *db.Queries) error {
		tq := s.txQ(q)
		if err := s.canManageWithQuerier(ctx, teamID, inviterID, globalRole, tq.members); err != nil {
			return err
		}
		role := "player"
		status := "pending"
		action := "invite"

		existing, err := tq.members.GetMembership(ctx, db.GetMembershipParams{
			TeamID: teamID,
			UserID: inviteeID,
		})
		if err == nil {
			if existing.Status != nil && *existing.Status != "rejected" {
				return errs.ErrConflict
			}
			_, err = tq.members.UpdateMembershipStatus(ctx, db.UpdateMembershipStatusParams{
				ID:     existing.ID,
				Status: &status,
			})
		} else {
			_, err = tq.members.CreateTeamMembership(ctx, db.CreateTeamMembershipParams{
				TeamID: teamID,
				UserID: inviteeID,
				Role:   &role,
				Status: &status,
			})
		}
		if err != nil {
			return mapDBError(err)
		}

		actorID, err := int32PtrFromInt64(inviterID)
		if err != nil {
			return err
		}
		_, err = tq.events.CreateEvent(ctx, db.CreateEventParams{
			TeamID:   teamID,
			UserID:   inviteeID,
			ActorID:  actorID,
			Action:   action,
			ToRole:   &role,
			ToStatus: &status,
		})
		return err
	})
}

func fromDB(t db.Team) Team {
	return Team{
		ID:           t.ID,
		Name:         t.Name,
		Description:  t.Description,
		Website:      t.Website,
		AvatarUrl:    t.AvatarUrl,
		CaptainID:    t.CaptainID,
		UniversityID: t.UniversityID,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

func int32PtrFromInt64(v int64) (*int32, error) {
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

const (
	minInt32 = -1 << 31
	maxInt32 = 1<<31 - 1
)

func int32FromInt64(v int64) (int32, error) {
	if v < minInt32 || v > maxInt32 {
		return 0, errs.NewValidationError(map[string]string{"id": "must fit int32"})
	}
	return int32(v), nil
}

func EnsureCaptainUnique(ctx context.Context, q TeamQuerier, userID int64) error {
	captainID, err := int32FromInt64(userID)
	if err != nil {
		return err
	}
	_, err = q.GetTeamByCaptain(ctx, &captainID)
	if err == nil {
		return fmt.Errorf("%w: user %d is already captain of another team", errs.ErrConflict, userID)
	}
	return nil
}
