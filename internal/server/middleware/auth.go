package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
)

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

func RequireAuth(jwtMgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "missing or invalid authorization header")
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil {
			abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "invalid token")
			return
		}

		var userID int64
		if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil {
			abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "invalid token subject")
			return
		}

		c.Set(string(userIDKey), userID)
		c.Set(string(roleKey), claims.Role)
		c.Set(string(userNameKey), claims.UserName)
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get(string(roleKey))
		if !exists {
			abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "not authenticated")
			return
		}

		currentRole, ok := val.(string)
		if !ok {
			abortWithJSON(c, http.StatusForbidden, errCodeForbidden, "insufficient permissions")
			return
		}

		if !hasRoleLevel(currentRole, role) {
			abortWithJSON(c, http.StatusForbidden, errCodeForbidden, "insufficient permissions")
			return
		}
		c.Next()
	}
}

func OptionalAuth(jwtMgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil {
			c.Next()
			return
		}

		var userID int64
		if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil {
			c.Next()
			return
		}

		c.Set(string(userIDKey), userID)
		c.Set(string(roleKey), claims.Role)
		c.Set(string(userNameKey), claims.UserName)
		c.Next()
	}
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
