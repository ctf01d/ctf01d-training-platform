package auth

import (
	"context"
	"fmt"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
)

type UserStore interface {
	GetUserByUserName(ctx context.Context, userName string) (db.User, error)
	GetUserByID(ctx context.Context, id int64) (db.User, error)
}

type JWTManager interface {
	Generate(userID int64, role, userName string) (string, error)
}

type PasswordChecker interface {
	CheckPassword(hash, plain string) bool
}

type Service struct {
	store   UserStore
	jwt     JWTManager
	checker PasswordChecker
}

func NewService(store UserStore, jwt JWTManager, checker PasswordChecker) *Service {
	return &Service{store: store, jwt: jwt, checker: checker}
}

func (s *Service) Login(ctx context.Context, userName, password string) (string, *usersvc.User, error) {
	dbUser, err := s.store.GetUserByUserName(ctx, userName)
	if err != nil {
		return "", nil, errs.ErrUnauthorized
	}

	if dbUser.PasswordDigest == nil || !s.checker.CheckPassword(*dbUser.PasswordDigest, password) {
		return "", nil, errs.ErrUnauthorized
	}

	token, err := s.jwt.Generate(dbUser.ID, dbUser.Role, dbUser.UserName)
	if err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}

	u := userFromDB(dbUser)
	return token, &u, nil
}

func (s *Service) Me(ctx context.Context, userID int64) (*usersvc.User, error) {
	dbUser, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	u := userFromDB(dbUser)
	return &u, nil
}

func userFromDB(u db.User) usersvc.User {
	return usersvc.User{
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
