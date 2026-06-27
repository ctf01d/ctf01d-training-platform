package users

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type mockQuerier struct {
	users  map[int64]db.User
	nextID int64
	byName map[string]int64
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		users:  make(map[int64]db.User),
		nextID: 1,
		byName: make(map[string]int64),
	}
}

func mustCreateUser(t *testing.T, svc *Service, params CreateParams) *User {
	t.Helper()
	user, err := svc.Create(context.Background(), params)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return user
}

func (m *mockQuerier) CreateUser(_ context.Context, arg db.CreateUserParams) (db.User, error) {
	if _, exists := m.byName[arg.UserName]; exists {
		return db.User{}, &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	}
	id := m.nextID
	m.nextID++
	now := time.Now()
	u := db.User{
		ID:             id,
		UserName:       arg.UserName,
		DisplayName:    arg.DisplayName,
		Language:       defaultLanguage,
		Role:           arg.Role,
		Rating:         arg.Rating,
		AvatarUrl:      arg.AvatarUrl,
		PasswordDigest: arg.PasswordDigest,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.users[id] = u
	m.byName[arg.UserName] = id
	return u, nil
}

func (m *mockQuerier) GetUserByID(_ context.Context, id int64) (db.User, error) {
	u, ok := m.users[id]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *mockQuerier) GetUserByUserName(_ context.Context, userName string) (db.User, error) {
	id, ok := m.byName[userName]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return m.users[id], nil
}

func (m *mockQuerier) ListUsers(_ context.Context, arg db.ListUsersParams) ([]db.User, error) {
	var result []db.User
	for i := int32(0); i < arg.Limit; i++ {
		idx := arg.Offset + i + 1
		if u, ok := m.users[int64(idx)]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockQuerier) CountUsers(_ context.Context, _ *string) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockQuerier) UpdateUserProfile(_ context.Context, arg db.UpdateUserProfileParams) (db.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	u.DisplayName = arg.DisplayName
	if arg.AvatarUrl != nil {
		u.AvatarUrl = arg.AvatarUrl
	}
	if arg.PasswordDigest != nil {
		u.PasswordDigest = arg.PasswordDigest
	}
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) UpdateUserProfileAdmin(_ context.Context, arg db.UpdateUserProfileAdminParams) (db.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	u.DisplayName = arg.DisplayName
	if arg.AvatarUrl != nil {
		u.AvatarUrl = arg.AvatarUrl
	}
	if arg.PasswordDigest != nil {
		u.PasswordDigest = arg.PasswordDigest
	}
	u.Bio = arg.Bio
	u.Telegram = arg.Telegram
	u.Github = arg.Github
	u.Email = arg.Email
	u.Language = arg.Language
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) UpdateUserPassword(_ context.Context, arg db.UpdateUserPasswordParams) (db.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	u.PasswordDigest = arg.PasswordDigest
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) SetUserAvatar(_ context.Context, arg db.SetUserAvatarParams) (db.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	u.AvatarUrl = arg.AvatarUrl
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) SetUserBlocked(_ context.Context, arg db.SetUserBlockedParams) (db.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	u.IsBlocked = arg.IsBlocked
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) ClearUserTeamCaptaincy(_ context.Context, _ *int32) error {
	return nil
}

func (m *mockQuerier) UpdateUserRole(_ context.Context, arg db.UpdateUserRoleParams) (db.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	u.Role = arg.Role
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) UpdateUserRating(_ context.Context, arg db.UpdateUserRatingParams) (db.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	u.Rating = arg.Rating
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) DeleteUser(_ context.Context, id int64) error {
	if u, ok := m.users[id]; ok {
		delete(m.byName, u.UserName)
		delete(m.users, id)
	}
	return nil
}

func TestCreate_Success(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	avatar := "https://example.com/avatar.png"
	u, err := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test User",
		Password:    "secret123",
		Role:        "guest",
		AvatarUrl:   &avatar,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID != 1 {
		t.Errorf("ID = %d, want 1", u.ID)
	}
	if u.UserName != "testuser" {
		t.Errorf("UserName = %q, want %q", u.UserName, "testuser")
	}
	if u.Role != "guest" {
		t.Errorf("Role = %q, want %q", u.Role, "guest")
	}
	if u.Language != defaultLanguage {
		t.Errorf("Language = %q, want %q", u.Language, defaultLanguage)
	}
}

func TestCreate_DuplicateUserName(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test User",
		Password:    "secret123",
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Another User",
		Password:    "secret456",
	})
	if !isErr(err, errs.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_InvalidUserName(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Create(context.Background(), CreateParams{
		UserName:    "invalid user!",
		DisplayName: "Test",
		Password:    "secret123",
	})
	var ve *errs.ValidationError
	if !isValidation(err) {
		t.Errorf("expected ValidationError, got %v", err)
	}
	_ = ve
}

func TestCreate_ShortPassword(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "abc",
	})
	if !isValidation(err) {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestCreate_DefaultRole(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	u, err := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.Role != "guest" {
		t.Errorf("Role = %q, want %q", u.Role, "guest")
	}
}

func TestGetByID_Success(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created, _ := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	u, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if u.UserName != "testuser" {
		t.Errorf("UserName = %q, want %q", u.UserName, "testuser")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.GetByID(context.Background(), 999)
	if !isErr(err, errs.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByUserName_Success(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	mustCreateUser(t, svc, CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	u, err := svc.GetByUserName(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("GetByUserName: %v", err)
	}
	if u.UserName != "testuser" {
		t.Errorf("UserName = %q, want %q", u.UserName, "testuser")
	}
}

func TestList(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	for i := 0; i < 5; i++ {
		mustCreateUser(t, svc, CreateParams{
			UserName:    "user" + string(rune('0'+i)),
			DisplayName: "User " + string(rune('0'+i)),
			Password:    "secret123",
		})
	}

	result, err := svc.List(context.Background(), 1, 3, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(result.Items))
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
	if result.Page != 1 || result.PerPage != 3 {
		t.Errorf("Page=%d PerPage=%d", result.Page, result.PerPage)
	}
}

func TestUpdate(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created, _ := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Old Name",
		Password:    "secret123",
	})

	newName := "New Name"
	u, err := svc.Update(context.Background(), created.ID, UpdateParams{
		DisplayName: &newName,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if u.DisplayName != "New Name" {
		t.Errorf("DisplayName = %q, want %q", u.DisplayName, "New Name")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	newName := "New Name"
	_, err := svc.Update(context.Background(), 999, UpdateParams{
		DisplayName: &newName,
	})
	if !isErr(err, errs.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateRole(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created, _ := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	u, err := svc.UpdateRole(context.Background(), created.ID, "admin")
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if u.Role != "admin" {
		t.Errorf("Role = %q, want %q", u.Role, "admin")
	}
}

func TestDelete(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created, _ := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	err := svc.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.GetByID(context.Background(), created.ID)
	if !isErr(err, errs.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestUpdate_PasswordChange(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created, _ := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	newPass := "newpassword"
	_, err := svc.Update(context.Background(), created.ID, UpdateParams{
		Password: &newPass,
	})
	if err != nil {
		t.Fatalf("Update password: %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created, _ := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})
	// Seed an existing profile field to prove the password change leaves it alone.
	bio := "hello"
	stored := q.users[created.ID]
	stored.Bio = &bio
	q.users[created.ID] = stored
	oldDigest := stored.PasswordDigest

	updated, err := svc.ChangePassword(context.Background(), created.ID, "newpassword")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if got := q.users[created.ID].PasswordDigest; got == oldDigest || got == nil {
		t.Fatalf("password digest was not updated")
	}
	// Other profile fields must be left untouched.
	if updated.Bio == nil || *updated.Bio != "hello" {
		t.Fatalf("bio changed unexpectedly: %v", updated.Bio)
	}
	if updated.DisplayName != "Test" {
		t.Fatalf("display_name changed unexpectedly: %q", updated.DisplayName)
	}
}

func TestChangePassword_ShortPassword(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created, _ := svc.Create(context.Background(), CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	_, err := svc.ChangePassword(context.Background(), created.ID, "short")
	if !isValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestChangePassword_NotFound(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	_, err := svc.ChangePassword(context.Background(), 999, "newpassword")
	if !isErr(err, errs.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestUpdateProfile_LanguageChange(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created := mustCreateUser(t, svc, CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	language := "ru"
	updated, err := svc.UpdateProfile(context.Background(), created.ID, ProfileUpdateParams{
		Language: &language,
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if updated.Language != "ru" {
		t.Fatalf("Language = %q, want %q", updated.Language, "ru")
	}
	if q.users[created.ID].Language != "ru" {
		t.Fatalf("stored language = %q, want %q", q.users[created.ID].Language, "ru")
	}
}

func TestUpdateProfile_InvalidLanguage(t *testing.T) {
	q := newMockQuerier()
	svc := NewService(q)

	created := mustCreateUser(t, svc, CreateParams{
		UserName:    "testuser",
		DisplayName: "Test",
		Password:    "secret123",
	})

	language := "de"
	_, err := svc.UpdateProfile(context.Background(), created.ID, ProfileUpdateParams{
		Language: &language,
	})
	if !isValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func isErr(err error, target error) bool {
	return err != nil && err.Error() == target.Error()
}

func isValidation(err error) bool {
	_, ok := err.(*errs.ValidationError)
	return ok
}
