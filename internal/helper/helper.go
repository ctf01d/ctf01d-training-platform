package helper

import (
	"database/sql"
	"net/url"

	"ctf01d/internal/httpserver"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(s string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPasswordHash(s, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(s))
	return err == nil
}

func ToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{
			String: "",
			Valid:  false,
		}
	}
	return sql.NullString{
		String: *s,
		Valid:  true,
	}
}

func WithDefault(img string) string {
	return "api/v1/avatar/" + url.QueryEscape(img)
}

func ConvertUserRequestRoleToUserResponseRole(role httpserver.UserRequestRole) httpserver.UserResponseRole {
	switch role {
	case httpserver.UserRequestRoleAdmin:
		return httpserver.UserResponseRoleAdmin
	case httpserver.UserRequestRoleGuest:
		return httpserver.UserResponseRoleGuest
	case httpserver.UserRequestRolePlayer:
		return httpserver.UserResponseRolePlayer
	default:
		return ""
	}
}

func ConvertUserRequestRoleToString(role httpserver.UserRequestRole) string {
	switch role {
	case httpserver.UserRequestRoleAdmin:
		return "admin"
	case httpserver.UserRequestRoleGuest:
		return "guest"
	case httpserver.UserRequestRolePlayer:
		return "player"
	default:
		return ""
	}
}

func ConvertUserResponseRoleToUserRequestRole(role httpserver.UserResponseRole) httpserver.UserRequestRole {
	switch role {
	case httpserver.UserResponseRoleAdmin:
		return httpserver.UserRequestRoleAdmin
	case httpserver.UserResponseRoleGuest:
		return httpserver.UserRequestRoleGuest
	case httpserver.UserResponseRolePlayer:
		return httpserver.UserRequestRolePlayer
	default:
		return ""
	}
}
