package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
)

type UserStore interface {
	GetUserByUserName(ctx context.Context, userName string) (db.User, error)
	GetUserByID(ctx context.Context, id int64) (db.User, error)
	SetUserLastLogin(ctx context.Context, arg db.SetUserLastLoginParams) (db.User, error)
}

type SessionStore interface {
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.UserSession, error)
	GetSessionForAuth(ctx context.Context, jti string) (db.GetSessionForAuthRow, error)
	ListActiveSessionsByUser(ctx context.Context, userID int64) ([]db.UserSession, error)
	TouchSession(ctx context.Context, arg db.TouchSessionParams) error
	RevokeSession(ctx context.Context, jti string) error
	RevokeSessionByID(ctx context.Context, arg db.RevokeSessionByIDParams) error
	RevokeAllUserSessions(ctx context.Context, userID int64) error
}

// sessionTouchInterval throttles last-seen updates so a read endpoint does not
// turn into a row write on every request.
const sessionTouchInterval = 5 * time.Minute

type JWTManager interface {
	Generate(userID int64, role, userName, jti string) (string, error)
	TTL() time.Duration
}

type PasswordChecker interface {
	CheckPassword(hash, plain string) bool
}

type Service struct {
	store    UserStore
	sessions SessionStore
	jwt      JWTManager
	checker  PasswordChecker
}

func NewService(store UserStore, sessions SessionStore, jwt JWTManager, checker PasswordChecker) *Service {
	return &Service{store: store, sessions: sessions, jwt: jwt, checker: checker}
}

// Session is the public view of an active login.
type Session struct {
	ID         int64     `json:"id"`
	IPAddress  *string   `json:"ip_address,omitempty"`
	UserAgent  *string   `json:"user_agent,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Current    bool      `json:"current"`
}

func (s *Service) Login(ctx context.Context, userName, password, ipAddress, userAgent string) (string, *usersvc.User, error) {
	dbUser, err := s.store.GetUserByUserName(ctx, userName)
	if err != nil {
		return "", nil, errs.ErrUnauthorized
	}

	if dbUser.PasswordDigest == nil || !s.checker.CheckPassword(*dbUser.PasswordDigest, password) {
		return "", nil, errs.ErrUnauthorized
	}

	if dbUser.IsBlocked {
		return "", nil, fmt.Errorf("%w: account is blocked", errs.ErrForbidden)
	}

	jti, err := auth.NewSessionID()
	if err != nil {
		return "", nil, fmt.Errorf("generating session id: %w", err)
	}

	ip := strToPtr(ipAddress)
	// Order matters: record login and sign the token before the session row.
	// CreateSession runs last so a failure leaves no orphan session, and a
	// returned token always has a backing session.
	dbUser, err = s.store.SetUserLastLogin(ctx, db.SetUserLastLoginParams{ID: dbUser.ID, LastLoginIp: ip})
	if err != nil {
		return "", nil, fmt.Errorf("recording login: %w", err)
	}

	token, err := s.jwt.Generate(dbUser.ID, dbUser.Role, dbUser.UserName, jti)
	if err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}

	if _, err := s.sessions.CreateSession(ctx, db.CreateSessionParams{
		UserID:    dbUser.ID,
		Jti:       jti,
		IpAddress: ip,
		UserAgent: strToPtr(userAgent),
		ExpiresAt: time.Now().Add(s.jwt.TTL()),
	}); err != nil {
		return "", nil, fmt.Errorf("creating session: %w", err)
	}

	u := userFromDB(dbUser)
	return token, &u, nil
}

// VerifyUserPassword reports whether the given plaintext password matches the
// stored digest for the user. Used to confirm sensitive admin actions.
func (s *Service) VerifyUserPassword(ctx context.Context, userID int64, password string) bool {
	dbUser, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return false
	}
	return dbUser.PasswordDigest != nil && s.checker.CheckPassword(*dbUser.PasswordDigest, password)
}

func (s *Service) Me(ctx context.Context, userID int64) (*usersvc.User, error) {
	dbUser, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	u := userFromDB(dbUser)
	return &u, nil
}

// ValidateAndTouch reports whether the session identified by jti is usable
// (exists, not revoked, not expired, owner not blocked) using a single read. It
// also refreshes last-seen/IP, but at most once per sessionTouchInterval so a
// read request is not turned into a write on every call. An empty jti is
// invalid.
//
// Checking the blocked flag here makes blocking effective immediately even if
// the bulk session revocation that accompanies it fails.
func (s *Service) ValidateAndTouch(ctx context.Context, jti, ipAddress string) bool {
	if jti == "" {
		return false
	}
	row, err := s.sessions.GetSessionForAuth(ctx, jti)
	if err != nil {
		return false
	}
	if row.RevokedAt.Valid || time.Now().After(row.ExpiresAt) || row.IsBlocked {
		return false
	}
	if time.Since(row.LastSeenAt) >= sessionTouchInterval {
		_ = s.sessions.TouchSession(ctx, db.TouchSessionParams{Jti: jti, IpAddress: strToPtr(ipAddress)})
	}
	return true
}

// Logout revokes the session tied to the current token.
func (s *Service) Logout(ctx context.Context, jti string) error {
	if jti == "" {
		return nil
	}
	return s.sessions.RevokeSession(ctx, jti)
}

// ListSessions returns the active sessions for a user, flagging the one that
// matches currentJTI (the caller's own token).
func (s *Service) ListSessions(ctx context.Context, userID int64, currentJTI string) ([]Session, error) {
	rows, err := s.sessions.ListActiveSessionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, len(rows))
	for i, r := range rows {
		sessions[i] = Session{
			ID:         r.ID,
			IPAddress:  r.IpAddress,
			UserAgent:  r.UserAgent,
			CreatedAt:  r.CreatedAt,
			LastSeenAt: r.LastSeenAt,
			ExpiresAt:  r.ExpiresAt,
			Current:    r.Jti == currentJTI,
		}
	}
	return sessions, nil
}

// RevokeUserSession revokes a single session owned by the given user.
func (s *Service) RevokeUserSession(ctx context.Context, userID, sessionID int64) error {
	return s.sessions.RevokeSessionByID(ctx, db.RevokeSessionByIDParams{ID: sessionID, UserID: userID})
}

// RevokeAllSessions revokes every active session for a user (used on block).
func (s *Service) RevokeAllSessions(ctx context.Context, userID int64) error {
	return s.sessions.RevokeAllUserSessions(ctx, userID)
}

func userFromDB(u db.User) usersvc.User {
	var lastLoginAt *time.Time
	if u.LastLoginAt.Valid {
		lastLoginAt = &u.LastLoginAt.Time
	}
	return usersvc.User{
		ID:          u.ID,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		Language:    userLanguage(u.Language),
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

func strToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func userLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ru":
		return "ru"
	default:
		return "en"
	}
}
