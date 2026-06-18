package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

type contextKey string

// SessionChecker validates and refreshes server-side sessions in a single call.
// It is satisfied by the auth service; a nil checker disables session
// enforcement (used in lightweight tests).
type SessionChecker interface {
	ValidateAndTouch(ctx context.Context, jti, ipAddress string) bool
}

const (
	userIDKey   contextKey = "user_id"
	roleKey     contextKey = "role"
	userNameKey contextKey = "user_name"
	sessionKey  contextKey = "session_jti"

	roleGuestLevel  = 0
	rolePlayerLevel = 1
	roleAdminLevel  = 2

	roleGuest  = "guest"
	rolePlayer = "player"
	roleAdmin  = "admin"

	jsonKeyCode         = "code"
	jsonKeyMessage      = "message"
	errCodeUnauthorized = "unauthorized"
	errCodeForbidden    = "forbidden"
)

func abortWithJSON(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{jsonKeyCode: code, jsonKeyMessage: msg})
	c.Abort()
}

func CurrentUserID(c *gin.Context) (int64, bool) {
	id, exists := c.Get(string(userIDKey))
	if !exists {
		return 0, false
	}
	return id.(int64), true
}

func CurrentRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(string(roleKey))
	if !exists {
		return "", false
	}
	return role.(string), true
}

func CurrentUserName(c *gin.Context) (string, bool) {
	name, exists := c.Get(string(userNameKey))
	if !exists {
		return "", false
	}
	return name.(string), true
}

// CurrentSessionJTI returns the session identifier from the caller's token.
func CurrentSessionJTI(c *gin.Context) (string, bool) {
	jti, exists := c.Get(string(sessionKey))
	if !exists {
		return "", false
	}
	return jti.(string), true
}

var roleLevel = map[string]int{
	roleGuest:  roleGuestLevel,
	rolePlayer: rolePlayerLevel,
	roleAdmin:  roleAdminLevel,
}

func hasRoleLevel(current, required string) bool {
	return roleLevel[current] >= roleLevel[required]
}
