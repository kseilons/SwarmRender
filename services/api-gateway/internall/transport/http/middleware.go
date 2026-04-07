package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type AuthMiddleware struct {
	logger  *slog.Logger
	jwksURL string
	cache   jwk.Set
}

func NewAuthMiddleware(logger *slog.Logger, jwksURL string) *AuthMiddleware {
	return &AuthMiddleware{
		logger:  logger,
		jwksURL: jwksURL,
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.logger.Warn("missing authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.logger.Warn("invalid authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer <token>"})
			return
		}

		tokenString := parts[1]

		keySet, err := jwk.Fetch(c.Request.Context(), m.jwksURL)
		if err != nil {
			m.logger.Error("failed to fetch JWKS", slog.Any("error", err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify token"})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("missing kid in header")
			}

			key, found := keySet.LookupKeyID(kid)
			if !found {
				return nil, fmt.Errorf("key %s not found in JWKS", kid)
			}

			var rawKey interface{}
			if err := key.Raw(&rawKey); err != nil {
				return nil, fmt.Errorf("failed to get raw key: %w", err)
			}

			return rawKey, nil
		})

		if err != nil || !token.Valid {
			m.logger.Warn("invalid token", slog.Any("error", err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		userID, _ := claims["sub"].(string)
		c.Set("user_id", userID)

		c.Request.Header.Set("X-User-Id", userID)

		c.Next()
	}
}

func (m *AuthMiddleware) GetJWKS(ctx context.Context) (jwk.Set, error) {
	return jwk.Fetch(ctx, m.jwksURL)
}
