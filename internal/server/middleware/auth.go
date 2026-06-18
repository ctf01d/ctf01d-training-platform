package middleware

import "github.com/gin-gonic/gin"

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	roleKey     contextKey = "role"
	userNameKey contextKey = "user_name"

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

var roleLevel = map[string]int{
	roleGuest:  roleGuestLevel,
	rolePlayer: rolePlayerLevel,
	roleAdmin:  roleAdminLevel,
}

func hasRoleLevel(current, required string) bool {
	return roleLevel[current] >= roleLevel[required]
}
