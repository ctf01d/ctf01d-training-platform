package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

type mockUserStore struct {
	users map[string]db.User
	byID  map[int64]db.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]db.User), byID: make(map[int64]db.User)}
}

func (m *mockUserStore) GetUserByUserName(_ context.Context, userName string) (db.User, error) {
	u, ok := m.users[userName]
	if !ok {
		return db.User{}, fmt.Errorf("no rows in result set")
	}
	return u, nil
}

func (m *mockUserStore) GetUserByID(_ context.Context, id int64) (db.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return db.User{}, fmt.Errorf("no rows in result set")
	}
	return u, nil
}

type mockJWT struct {
	secret string
}

func (m *mockJWT) Generate(userID int64, role, userName string) (string, error) {
	return "token_" + userName, nil
}

type mockChecker struct{}

func (m *mockChecker) CheckPassword(hash, plain string) bool {
	return hash == "hash_"+plain
}

func addTestUser(store *mockUserStore, id int64, userName, password string) {
	hash := "hash_" + password
	store.users[userName] = db.User{
		ID:             id,
		UserName:       userName,
		DisplayName:    "Display " + userName,
		Role:           "player",
		Rating:         0,
		PasswordDigest: &hash,
	}
	store.byID[id] = store.users[userName]
}

func TestLogin_Success(t *testing.T) {
	store := newMockUserStore()
	addTestUser(store, 1, "alice", "password123")

	svc := NewService(store, &mockJWT{}, &mockChecker{})
	token, user, err := svc.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "token_alice" {
		t.Errorf("expected token_alice, got %s", token)
	}
	if user.UserName != "alice" {
		t.Errorf("expected alice, got %s", user.UserName)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	store := newMockUserStore()
	svc := NewService(store, &mockJWT{}, &mockChecker{})
	_, _, err := svc.Login(context.Background(), "nobody", "pass")
	if err != errs.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	store := newMockUserStore()
	addTestUser(store, 1, "alice", "password123")

	svc := NewService(store, &mockJWT{}, &mockChecker{})
	_, _, err := svc.Login(context.Background(), "alice", "wrong")
	if err != errs.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogin_NilPasswordDigest(t *testing.T) {
	store := newMockUserStore()
	store.users["bob"] = db.User{
		ID: 2, UserName: "bob", DisplayName: "Bob", Role: "guest", Rating: 0, PasswordDigest: nil,
	}
	store.byID[2] = store.users["bob"]

	svc := NewService(store, &mockJWT{}, &mockChecker{})
	_, _, err := svc.Login(context.Background(), "bob", "any")
	if err != errs.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestMe_Success(t *testing.T) {
	store := newMockUserStore()
	addTestUser(store, 1, "alice", "password123")

	svc := NewService(store, &mockJWT{}, &mockChecker{})
	user, err := svc.Me(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.UserName != "alice" {
		t.Errorf("expected alice, got %s", user.UserName)
	}
}

func TestMe_NotFound(t *testing.T) {
	store := newMockUserStore()
	svc := NewService(store, &mockJWT{}, &mockChecker{})
	_, err := svc.Me(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
