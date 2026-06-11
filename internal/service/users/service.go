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
	ID          int64      `json:"id"`
	UserName    string     `json:"user_name"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Rating      int        `json:"rating"`
	AvatarUrl   *string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UserListResult struct {
	Items []User `json:"items"`
	Page  int    `json:"page"`
	PerPage int  `json:"per_page"`
	Total int64  `json:"total"`
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

var userNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type Querier interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByID(ctx context.Context, id int64) (db.User, error)
	GetUserByUserName(ctx context.Context, userName string) (db.User, error)
	ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.User, error)
	CountUsers(ctx context.Context) (int64, error)
	UpdateUserProfile(ctx context.Context, arg db.UpdateUserProfileParams) (db.User, error)
	UpdateUserRole(ctx context.Context, arg db.UpdateUserRoleParams) (db.User, error)
	UpdateUserRating(ctx context.Context, arg db.UpdateUserRatingParams) (db.User, error)
	DeleteUser(ctx context.Context, id int64) error
}

type Service struct {
	q Querier
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

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
	if len(params.Password) < 6 {
		return nil, errs.NewValidationError(map[string]string{
			"password": "must be at least 6 characters",
		})
	}

	role := params.Role
	if role == "" {
		role = "guest"
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
		return nil, mapDBError(err, "user_name", params.UserName)
	}

	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	dbUser, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "user")
	}
	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) GetByUserName(ctx context.Context, userName string) (*User, error) {
	dbUser, err := s.q.GetUserByUserName(ctx, userName)
	if err != nil {
		return nil, mapNotFound(err, "user")
	}
	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) List(ctx context.Context, page, perPage int) (*UserListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

	items, err := s.q.ListUsers(ctx, db.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	total, err := s.q.CountUsers(ctx)
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
		return nil, mapNotFound(err, "user")
	}

	displayName := existing.DisplayName
	if params.DisplayName != nil {
		displayName = *params.DisplayName
	}

	var passwordDigest *string
	if params.Password != nil {
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
	dbUser, err := s.q.UpdateUserRating(ctx, db.UpdateUserRatingParams{
		ID:     id,
		Rating: int32(rating),
	})
	if err != nil {
		return nil, err
	}
	u := fromDB(dbUser)
	return &u, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
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
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func mapNotFound(err error, entity string) error {
	if repository.IsNoRows(err) {
		return errs.ErrNotFound
	}
	return err
}

func mapDBError(err error, field, value string) error {
	if repository.IsDuplicateKey(err) {
		return errs.ErrConflict
	}
	return err
}
