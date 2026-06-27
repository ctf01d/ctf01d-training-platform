package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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
		return db.User{}, errors.New("no rows in result set")
	}
	return u, nil
}

func (m *mockUserStore) GetUserByID(_ context.Context, id int64) (db.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return db.User{}, errors.New("no rows in result set")
	}
	return u, nil
}

func (m *mockUserStore) SetUserLastLogin(_ context.Context, arg db.SetUserLastLoginParams) (db.User, error) {
	u, ok := m.byID[arg.ID]
	if !ok {
		return db.User{}, errors.New("no rows in result set")
	}
	u.LastLoginIp = arg.LastLoginIp
	u.LastLoginAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	m.byID[arg.ID] = u
	m.users[u.UserName] = u
	return u, nil
}

type mockSessionStore struct {
	created    []db.CreateSessionParams
	authRow    db.GetSessionForAuthRow
	authErr    error
	touchCalls int
}

func (m *mockSessionStore) CreateSession(_ context.Context, arg db.CreateSessionParams) (db.UserSession, error) {
	m.created = append(m.created, arg)
	return db.UserSession{ID: int64(len(m.created)), Jti: arg.Jti, UserID: arg.UserID}, nil
}

func (m *mockSessionStore) GetSessionForAuth(_ context.Context, _ string) (db.GetSessionForAuthRow, error) {
	if m.authErr != nil {
		return db.GetSessionForAuthRow{}, m.authErr
	}
	return m.authRow, nil
}

func (m *mockSessionStore) ListActiveSessionsByUser(_ context.Context, _ int64) ([]db.UserSession, error) {
	return nil, nil
}

func (m *mockSessionStore) TouchSession(_ context.Context, _ db.TouchSessionParams) error {
	m.touchCalls++
	return nil
}

func (m *mockSessionStore) RevokeSession(_ context.Context, _ string) error { return nil }

func (m *mockSessionStore) RevokeSessionByID(_ context.Context, _ db.RevokeSessionByIDParams) error {
	return nil
}

func (m *mockSessionStore) RevokeAllUserSessions(_ context.Context, _ int64) error { return nil }

type mockJWT struct{}

func (m *mockJWT) Generate(_ int64, _, userName, _ string) (string, error) {
	return "token_" + userName, nil
}

func (m *mockJWT) TTL() time.Duration { return time.Hour }

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
		Language:       "ru",
		Role:           "player",
		Rating:         0,
		PasswordDigest: &hash,
	}
	store.byID[id] = store.users[userName]
}

func TestLogin_Success(t *testing.T) {
	store := newMockUserStore()
	addTestUser(store, 1, "alice", "password123")

	svc := NewService(store, &mockSessionStore{}, &mockJWT{}, &mockChecker{})
	token, user, err := svc.Login(context.Background(), "alice", "password123", "1.2.3.4", "ua")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "token_alice" {
		t.Errorf("expected token_alice, got %s", token)
	}
	if user.UserName != "alice" {
		t.Errorf("expected alice, got %s", user.UserName)
	}
	if user.Language != "ru" {
		t.Errorf("expected language ru, got %s", user.Language)
	}
	if user.LastLoginIp == nil || *user.LastLoginIp != "1.2.3.4" {
		t.Errorf("expected last login IP 1.2.3.4, got %v", user.LastLoginIp)
	}
	if user.LastLoginAt == nil {
		t.Error("expected last login time")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	store := newMockUserStore()
	svc := NewService(store, &mockSessionStore{}, &mockJWT{}, &mockChecker{})
	_, _, err := svc.Login(context.Background(), "nobody", "pass", "", "")
	if err != errs.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	store := newMockUserStore()
	addTestUser(store, 1, "alice", "password123")

	svc := NewService(store, &mockSessionStore{}, &mockJWT{}, &mockChecker{})
	_, _, err := svc.Login(context.Background(), "alice", "wrong", "", "")
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

	svc := NewService(store, &mockSessionStore{}, &mockJWT{}, &mockChecker{})
	_, _, err := svc.Login(context.Background(), "bob", "any", "", "")
	if err != errs.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestMe_Success(t *testing.T) {
	store := newMockUserStore()
	addTestUser(store, 1, "alice", "password123")

	svc := NewService(store, &mockSessionStore{}, &mockJWT{}, &mockChecker{})
	user, err := svc.Me(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.UserName != "alice" {
		t.Errorf("expected alice, got %s", user.UserName)
	}
	if user.Language != "ru" {
		t.Errorf("expected language ru, got %s", user.Language)
	}
}

func TestValidateAndTouch(t *testing.T) {
	now := time.Now()
	mk := func(row db.GetSessionForAuthRow) (*Service, *mockSessionStore) {
		ss := &mockSessionStore{authRow: row}
		return NewService(newMockUserStore(), ss, &mockJWT{}, &mockChecker{}), ss
	}

	t.Run("valid stale session is touched", func(t *testing.T) {
		svc, ss := mk(db.GetSessionForAuthRow{ExpiresAt: now.Add(time.Hour), LastSeenAt: now.Add(-10 * time.Minute)})
		if !svc.ValidateAndTouch(context.Background(), "jti", "1.2.3.4") {
			t.Fatal("expected valid")
		}
		if ss.touchCalls != 1 {
			t.Errorf("touchCalls = %d, want 1", ss.touchCalls)
		}
	})

	t.Run("valid recent session is not touched", func(t *testing.T) {
		svc, ss := mk(db.GetSessionForAuthRow{ExpiresAt: now.Add(time.Hour), LastSeenAt: now.Add(-time.Minute)})
		if !svc.ValidateAndTouch(context.Background(), "jti", "1.2.3.4") {
			t.Fatal("expected valid")
		}
		if ss.touchCalls != 0 {
			t.Errorf("touchCalls = %d, want 0 (throttled)", ss.touchCalls)
		}
	})

	t.Run("blocked owner is rejected", func(t *testing.T) {
		svc, _ := mk(db.GetSessionForAuthRow{ExpiresAt: now.Add(time.Hour), LastSeenAt: now, IsBlocked: true})
		if svc.ValidateAndTouch(context.Background(), "jti", "") {
			t.Error("blocked owner must be invalid")
		}
	})

	t.Run("expired is rejected", func(t *testing.T) {
		svc, _ := mk(db.GetSessionForAuthRow{ExpiresAt: now.Add(-time.Hour), LastSeenAt: now})
		if svc.ValidateAndTouch(context.Background(), "jti", "") {
			t.Error("expired session must be invalid")
		}
	})

	t.Run("revoked is rejected", func(t *testing.T) {
		svc, _ := mk(db.GetSessionForAuthRow{ExpiresAt: now.Add(time.Hour), LastSeenAt: now, RevokedAt: pgtype.Timestamptz{Time: now, Valid: true}})
		if svc.ValidateAndTouch(context.Background(), "jti", "") {
			t.Error("revoked session must be invalid")
		}
	})

	t.Run("empty jti is rejected", func(t *testing.T) {
		svc, _ := mk(db.GetSessionForAuthRow{ExpiresAt: now.Add(time.Hour), LastSeenAt: now})
		if svc.ValidateAndTouch(context.Background(), "", "") {
			t.Error("empty jti must be invalid")
		}
	})

	t.Run("missing session is rejected", func(t *testing.T) {
		ss := &mockSessionStore{authErr: errors.New("no rows in result set")}
		svc := NewService(newMockUserStore(), ss, &mockJWT{}, &mockChecker{})
		if svc.ValidateAndTouch(context.Background(), "jti", "") {
			t.Error("missing session must be invalid")
		}
	})
}

func TestMe_NotFound(t *testing.T) {
	store := newMockUserStore()
	svc := NewService(store, &mockSessionStore{}, &mockJWT{}, &mockChecker{})
	_, err := svc.Me(context.Background(), 999)
	if err != errs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
