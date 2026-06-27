package users

import (
	"context"
	"regexp"
	"strings"
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
	Language    string     `json:"language"`
	Theme       string     `json:"theme"`
	Role        string     `json:"role"`
	Rating      int        `json:"rating"`
	AvatarUrl   *string    `json:"avatar_url,omitempty"`
	Bio         *string    `json:"bio,omitempty"`
	Telegram    *string    `json:"telegram,omitempty"`
	Github      *string    `json:"github,omitempty"`
	Email       *string    `json:"email,omitempty"`
	LastLoginIp *string    `json:"last_login_ip,omitempty"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	IsBlocked   bool       `json:"is_blocked"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
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

// ProfileUpdateParams carries the full set of fields editable on profile
// screens. Optional profile fields are set verbatim (a nil pointer clears the
// column).
type ProfileUpdateParams struct {
	DisplayName *string `json:"display_name,omitempty"`
	Password    *string `json:"password,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	Telegram    *string `json:"telegram,omitempty"`
	Github      *string `json:"github,omitempty"`
	Email       *string `json:"email,omitempty"`
	Language    *string `json:"language,omitempty"`
	Theme       *string `json:"theme,omitempty"`
}

// AdminUpdateParams carries the profile fields an admin may edit from the user
// management page. Keep it separate from ProfileUpdateParams so future admin-only
// fields or validation do not silently become self-service.
type AdminUpdateParams struct {
	DisplayName *string `json:"display_name,omitempty"`
	Password    *string `json:"password,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	Telegram    *string `json:"telegram,omitempty"`
	Github      *string `json:"github,omitempty"`
	Email       *string `json:"email,omitempty"`
	Language    *string `json:"language,omitempty"`
	Theme       *string `json:"theme,omitempty"`
}

var userNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Selectable UI themes, kept in sync with the `theme` enum in the OpenAPI spec
// and the frontend theme catalog.
const (
	themeClassic  = "classic"
	themeIndigo   = "indigo"
	themeDark     = "dark"
	themeMidnight = "midnight"
)

const (
	defaultRole     = "guest"
	defaultLanguage = "en"
	defaultTheme    = themeClassic
)

var validThemes = map[string]struct{}{
	themeClassic:  {},
	themeIndigo:   {},
	themeDark:     {},
	themeMidnight: {},
}

// fieldPassword is the validation-error field key for the password input.
const fieldPassword = "password"

type Querier interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByID(ctx context.Context, id int64) (db.User, error)
	GetUserByUserName(ctx context.Context, userName string) (db.User, error)
	ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.User, error)
	CountUsers(ctx context.Context, searchQuery *string) (int64, error)
	UpdateUserProfile(ctx context.Context, arg db.UpdateUserProfileParams) (db.User, error)
	UpdateUserProfileAdmin(ctx context.Context, arg db.UpdateUserProfileAdminParams) (db.User, error)
	UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) (db.User, error)
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

const (
	minPasswordLength       = 6
	passwordTooShortMessage = "must be at least 6 characters"
)

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
			fieldPassword: passwordTooShortMessage,
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
		if len(*params.Password) < minPasswordLength {
			return nil, errs.NewValidationError(map[string]string{fieldPassword: passwordTooShortMessage})
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

// UpdateProfile applies profile edits, setting optional profile fields verbatim.
// display_name falls back to the existing value when omitted.
func (s *Service) UpdateProfile(ctx context.Context, id int64, params ProfileUpdateParams) (*User, error) {
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

	language := userLanguage(existing.Language)
	if params.Language != nil {
		normalizedLanguage, ok := normalizeLanguage(*params.Language)
		if !ok {
			return nil, errs.NewValidationError(map[string]string{"language": "must be one of: en, ru"})
		}
		language = normalizedLanguage
	}

	theme := userTheme(existing.Theme)
	if params.Theme != nil {
		normalizedTheme, ok := normalizeTheme(*params.Theme)
		if !ok {
			return nil, errs.NewValidationError(map[string]string{"theme": "must be one of: classic, indigo, dark, midnight"})
		}
		theme = normalizedTheme
	}

	var passwordDigest *string
	if params.Password != nil {
		if len(*params.Password) < minPasswordLength {
			return nil, errs.NewValidationError(map[string]string{fieldPassword: passwordTooShortMessage})
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
		Language:       language,
		Theme:          theme,
	})
	if err != nil {
		return nil, err
	}

	u := fromDB(dbUser)
	return &u, nil
}

// ChangePassword sets a new password for a user, leaving every other field
// untouched. It is intentionally separate from the profile update so changing a
// password — a critical action — can never clobber other profile data.
func (s *Service) ChangePassword(ctx context.Context, id int64, password string) (*User, error) {
	if len(password) < minPasswordLength {
		return nil, errs.NewValidationError(map[string]string{fieldPassword: passwordTooShortMessage})
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	dbUser, err := s.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:             id,
		PasswordDigest: &hash,
	})
	if err != nil {
		return nil, mapNotFound(err)
	}
	u := fromDB(dbUser)
	return &u, nil
}

// UpdateAdmin applies the same profile edits from the admin user-management page.
func (s *Service) UpdateAdmin(ctx context.Context, id int64, params AdminUpdateParams) (*User, error) {
	return s.UpdateProfile(ctx, id, ProfileUpdateParams(params))
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
	// teams.captain_id is an int32 column; converting first surfaces an error
	// rather than silently leaving a dangling captain reference behind.
	captainID, err := int32FromInt64(id)
	if err != nil {
		return err
	}
	if err := s.q.ClearUserTeamCaptaincy(ctx, &captainID); err != nil {
		return err
	}
	return s.q.DeleteUser(ctx, id)
}

func fromDB(u db.User) User {
	var lastLoginAt *time.Time
	if u.LastLoginAt.Valid {
		lastLoginAt = &u.LastLoginAt.Time
	}
	return User{
		ID:          u.ID,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		Language:    userLanguage(u.Language),
		Theme:       userTheme(u.Theme),
		Role:        u.Role,
		Rating:      int(u.Rating),
		AvatarUrl:   u.AvatarUrl,
		Bio:         u.Bio,
		Telegram:    u.Telegram,
		Github:      u.Github,
		Email:       u.Email,
		LastLoginIp: u.LastLoginIp,
		LastLoginAt: lastLoginAt,
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

func normalizeLanguage(value string) (string, bool) {
	language := strings.ToLower(strings.TrimSpace(value))
	switch language {
	case "en", "ru":
		return language, true
	default:
		return "", false
	}
}

func userLanguage(value string) string {
	if normalized, ok := normalizeLanguage(value); ok {
		return normalized
	}
	return defaultLanguage
}

func normalizeTheme(value string) (string, bool) {
	theme := strings.ToLower(strings.TrimSpace(value))
	if _, ok := validThemes[theme]; ok {
		return theme, true
	}
	return "", false
}

func userTheme(value string) string {
	if normalized, ok := normalizeTheme(value); ok {
		return normalized
	}
	return defaultTheme
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
