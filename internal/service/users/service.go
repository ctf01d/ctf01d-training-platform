package users

import (
	"context"
	"regexp"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type User struct {
	ID          int64     `json:"id"`
	UserName    string    `json:"user_name"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	Rating      int       `json:"rating"`
	AvatarUrl   *string   `json:"avatar_url,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	Telegram    *string   `json:"telegram,omitempty"`
	Github      *string   `json:"github,omitempty"`
	Email       *string   `json:"email,omitempty"`
	IsBlocked   bool      `json:"is_blocked"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserListResult struct {
	Items   []User `json:"items"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Total   int64  `json:"total"`
}

type CreateParams struct {
	UserName    string  `json:"user_name"`
	DisplayName string  `json:"display_name"`
	Password    string  `json:"password"`
	Role        string  `json:"role"`
	AvatarUrl   *string `json:"avatar_url,omitempty"`
}

type UpdateParams struct {
	DisplayName *string `json:"display_name,omitempty"`
	AvatarUrl   *string `json:"avatar_url,omitempty"`
	Password    *string `json:"password,omitempty"`
}

// AdminUpdateParams carries the full set of fields an admin may edit on the
// user management page. Optional profile fields are set verbatim (a nil pointer
// clears the column).
type AdminUpdateParams struct {
	DisplayName *string `json:"display_name,omitempty"`
	Password    *string `json:"password,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	Telegram    *string `json:"telegram,omitempty"`
	Github      *string `json:"github,omitempty"`
	Email       *string `json:"email,omitempty"`
}

var userNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

const defaultRole = "guest"

type Querier interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByID(ctx context.Context, id int64) (db.User, error)
	GetUserByUserName(ctx context.Context, userName string) (db.User, error)
	ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.User, error)
	CountUsers(ctx context.Context, searchQuery *string) (int64, error)
	UpdateUserProfile(ctx context.Context, arg db.UpdateUserProfileParams) (db.User, error)
	UpdateUserProfileAdmin(ctx context.Context, arg db.UpdateUserProfileAdminParams) (db.User, error)
	UpdateUserRole(ctx context.Context, arg db.UpdateUserRoleParams) (db.User, error)
	UpdateUserRating(ctx context.Context, arg db.UpdateUserRatingParams) (db.User, error)
	SetUserAvatar(ctx context.Context, arg db.SetUserAvatarParams) (db.User, error)
	SetUserBlocked(ctx context.Context, arg db.SetUserBlockedParams) (db.User, error)
	ClearUserTeamCaptaincy(ctx context.Context, captainID *int32) error
	DeleteUser(ctx context.Context, id int64) error
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

const minPasswordLength = 6

func (s *Service) Create(ctx context.Context, params CreateParams) (*User, error) {
	if params.UserName == "" || !userNameRegex.MatchString(params.UserName) {
		return nil, errs.NewValidationError(map[string]string{
			"user_name": "must match ^[a-zA-Z0-9_]+$ and be non-empty",
		})
	}
	if params.DisplayName == "" {
		return nil, errs.NewValidationError(map[string]string{
			"display_name": "is required",
		})
	}
	if len(params.Password) < minPasswordLength {
		return nil, errs.NewValidationError(map[string]string{
			"password": "must be at least 6 characters",
		})
	}

	role := params.Role
	if role == "" {
		role = defaultRole
	}

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	dbUser, err := s.q.CreateUser(ctx, db.CreateUserParams{
		UserName:       params.UserName,
		DisplayName:    params.DisplayName,
		Role:           role,
		Rating:         0,
		AvatarUrl:      params.AvatarUrl,
		PasswordDigest: &hash,
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	dbUser, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) GetByUserName(ctx context.Context, userName string) (*User, error) {
	dbUser, err := s.q.GetUserByUserName(ctx, userName)
	if err != nil {
		return nil, mapNotFound(err)
	}
	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) List(ctx context.Context, page, perPage int, query *string) (*UserListResult, error) {
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

	items, err := s.q.ListUsers(ctx, db.ListUsersParams{
		Limit:       limit,
		Offset:      offset,
		SearchQuery: query,
	})
	if err != nil {
		return nil, err
	}

	total, err := s.q.CountUsers(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &UserListResult{
		Items:   make([]User, len(items)),
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}
	for i, dbUser := range items {
		result.Items[i] = fromDB(dbUser)
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, id int64, params UpdateParams) (*User, error) {
	existing, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}

	displayName := existing.DisplayName
	if params.DisplayName != nil {
		displayName = *params.DisplayName
	}

	var passwordDigest *string
	if params.Password != nil {
		if *params.Password == "" || len(*params.Password) < 6 {
			return nil, errs.NewValidationError(map[string]string{"password": "password must be at least 6 characters"})
		}
		hash, err := auth.HashPassword(*params.Password)
		if err != nil {
			return nil, err
		}
		passwordDigest = &hash
	}

	dbUser, err := s.q.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:             id,
		DisplayName:    displayName,
		AvatarUrl:      params.AvatarUrl,
		PasswordDigest: passwordDigest,
	})
	if err != nil {
		return nil, err
	}

	u := fromDB(dbUser)
	return &u, nil
}

// UpdateAdmin applies an admin's edits, setting optional profile fields
// verbatim. display_name falls back to the existing value when omitted.
func (s *Service) UpdateAdmin(ctx context.Context, id int64, params AdminUpdateParams) (*User, error) {
	existing, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err)
	}

	displayName := existing.DisplayName
	if params.DisplayName != nil {
		if *params.DisplayName == "" {
			return nil, errs.NewValidationError(map[string]string{"display_name": "must not be empty"})
		}
		displayName = *params.DisplayName
	}

	var passwordDigest *string
	if params.Password != nil {
		if len(*params.Password) < minPasswordLength {
			return nil, errs.NewValidationError(map[string]string{"password": "must be at least 6 characters"})
		}
		hash, err := auth.HashPassword(*params.Password)
		if err != nil {
			return nil, err
		}
		passwordDigest = &hash
	}

	dbUser, err := s.q.UpdateUserProfileAdmin(ctx, db.UpdateUserProfileAdminParams{
		ID:             id,
		DisplayName:    displayName,
		AvatarUrl:      nil, // avatar managed via dedicated upload endpoint
		PasswordDigest: passwordDigest,
		Bio:            normalizeOptional(params.Bio),
		Telegram:       normalizeOptional(params.Telegram),
		Github:         normalizeOptional(params.Github),
		Email:          normalizeOptional(params.Email),
	})
	if err != nil {
		return nil, err
	}

	u := fromDB(dbUser)
	return &u, nil
}

// SetAvatar updates the stored avatar URL for a user.
func (s *Service) SetAvatar(ctx context.Context, id int64, url *string) (*User, error) {
	dbUser, err := s.q.SetUserAvatar(ctx, db.SetUserAvatarParams{ID: id, AvatarUrl: url})
	if err != nil {
		return nil, mapNotFound(err)
	}
	u := fromDB(dbUser)
	return &u, nil
}

// SetBlocked toggles the blocked flag for a user. Revoking active sessions is
// the caller's responsibility (handled in the auth service).
func (s *Service) SetBlocked(ctx context.Context, id int64, blocked bool) (*User, error) {
	dbUser, err := s.q.SetUserBlocked(ctx, db.SetUserBlockedParams{ID: id, IsBlocked: blocked})
	if err != nil {
		return nil, mapNotFound(err)
	}
	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) UpdateRole(ctx context.Context, id int64, role string) (*User, error) {
	dbUser, err := s.q.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:   id,
		Role: role,
	})
	if err != nil {
		return nil, err
	}
	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) UpdateRating(ctx context.Context, id int64, rating int) (*User, error) {
	dbRating, err := int32FromInt64(int64(rating))
	if err != nil {
		return nil, err
	}
	dbUser, err := s.q.UpdateUserRating(ctx, db.UpdateUserRatingParams{
		ID:     id,
		Rating: dbRating,
	})
	if err != nil {
		return nil, err
	}
	u := fromDB(dbUser)
	return &u, nil
}

// Delete removes a user along with references that would otherwise dangle.
// Team memberships and membership events cascade via foreign keys; team
// captaincy has no FK so it is cleared explicitly first.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if captainID, err := int32FromInt64(id); err == nil {
		cid := captainID
		if err := s.q.ClearUserTeamCaptaincy(ctx, &cid); err != nil {
			return err
		}
	}
	return s.q.DeleteUser(ctx, id)
}

func fromDB(u db.User) User {
	return User{
		ID:          u.ID,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Rating:      int(u.Rating),
		AvatarUrl:   u.AvatarUrl,
		Bio:         u.Bio,
		Telegram:    u.Telegram,
		Github:      u.Github,
		Email:       u.Email,
		IsBlocked:   u.IsBlocked,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// normalizeOptional treats empty strings as cleared (nil) values so the DB
// stores NULL rather than an empty string for optional profile fields.
func normalizeOptional(v *string) *string {
	if v == nil || *v == "" {
		return nil
	}
	return v
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
		return 0, errs.NewValidationError(map[string]string{"value": "must fit int32"})
	}
	return int32(v), nil
}
