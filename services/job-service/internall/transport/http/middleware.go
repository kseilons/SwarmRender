package http

import (
	"fmt"
	"kseilons/job-service/internall/domain"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	UserContextKey = "auth_user"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer <token>"})
			return
		}

		tokenString := parts[1]

		// Parse token without verification (we trust the gateway)
		token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		user := domain.AuthUser{
			ID:    claims["sub"].(string),
			Email: getStringClaim(claims, "email"),
		}

		// Extract roles if present
		if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
			if roles, ok := realmAccess["roles"].([]interface{}); ok {
				for _, r := range roles {
					if roleStr, ok := r.(string); ok {
						user.Roles = append(user.Roles, roleStr)
					}
				}
			}
		}

		c.Set(UserContextKey, user)
		c.Next()
	}
}

func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func GetUser(c *gin.Context) (domain.AuthUser, error) {
	val, exists := c.Get(UserContextKey)
	if !exists {
		return domain.AuthUser{}, fmt.Errorf("user not found in context")
	}

	user, ok := val.(domain.AuthUser)
	if !ok {
		return domain.AuthUser{}, fmt.Errorf("invalid user type in context")
	}

	return user, nil
}
