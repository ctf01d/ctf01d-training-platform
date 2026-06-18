package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/auth"
)

// OpenAPIAuth enforces BearerAuth only when the generated wrapper marks the
// operation as secured. For public operations it still accepts a valid token so
// handlers can render privileged fields for authenticated viewers.
func OpenAPIAuth(jwtMgr *auth.Manager) httpserver.MiddlewareFunc {
	return func(c *gin.Context) {
		requiresAuth := openAPIRequiresBearer(c)
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			if requiresAuth {
				abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "missing or invalid authorization header")
			}
			return
		}

		if jwtMgr == nil {
			if requiresAuth {
				abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "invalid token")
			}
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil {
			if requiresAuth {
				abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "invalid token")
			}
			return
		}

		var userID int64
		if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil {
			if requiresAuth {
				abortWithJSON(c, http.StatusUnauthorized, errCodeUnauthorized, "invalid token subject")
			}
			return
		}

		c.Set(string(userIDKey), userID)
		c.Set(string(roleKey), claims.Role)
		c.Set(string(userNameKey), claims.UserName)
	}
}

// OpenAPIRole enforces the role gates declared in OpenAPI via x-required-role.
func OpenAPIRole() httpserver.MiddlewareFunc {
	validateOpenAPIRoles(httpserver.OperationRequiredRoles)

	return func(c *gin.Context) {
		requiredRole := requiredOpenAPIRole(c.Request.Method, normalizedOpenAPIPath(c.FullPath()))
		if requiredRole == "" {
			return
		}

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

		if !hasRoleLevel(currentRole, requiredRole) {
			abortWithJSON(c, http.StatusForbidden, errCodeForbidden, "insufficient permissions")
		}
	}
}

func OpenAPIErrorHandler(c *gin.Context, err error, statusCode int) {
	c.JSON(statusCode, gin.H{jsonKeyCode: errCodeBadRequest, jsonKeyMessage: err.Error()})
}

func openAPIRequiresBearer(c *gin.Context) bool {
	_, ok := c.Get(string(httpserver.BearerAuthScopes))
	return ok
}

func normalizedOpenAPIPath(path string) string {
	if path == "" {
		return path
	}
	return strings.TrimPrefix(path, "/api/v1")
}

func requiredOpenAPIRole(method, path string) string {
	return httpserver.OperationRequiredRoles[method+" "+openAPIPathFromGin(path)]
}

func openAPIPathFromGin(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			segments[i] = "{" + strings.TrimPrefix(segment, ":") + "}"
		}
	}
	return strings.Join(segments, "/")
}

func validateOpenAPIRoles(roles map[string]string) {
	for operation, role := range roles {
		if _, ok := roleLevel[role]; !ok {
			panic(fmt.Sprintf("invalid OpenAPI required role %q for %s", role, operation))
		}
	}
}

const errCodeBadRequest = "bad_request"
