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
)

func RequireAuth(jwtMgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "missing or invalid authorization header"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "invalid token"})
			c.Abort()
			return
		}

		var userID int64
		if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "invalid token subject"})
			c.Abort()
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
			c.JSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "not authenticated"})
			c.Abort()
			return
		}

		currentRole, ok := val.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"code": "forbidden", "message": "insufficient permissions"})
			c.Abort()
			return
		}

		if !hasRoleLevel(currentRole, role) {
			c.JSON(http.StatusForbidden, gin.H{"code": "forbidden", "message": "insufficient permissions"})
			c.Abort()
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
	"guest":  roleGuestLevel,
	"player": rolePlayerLevel,
	"admin":  roleAdminLevel,
}

func hasRoleLevel(current, required string) bool {
	return roleLevel[current] >= roleLevel[required]
}
