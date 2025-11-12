package middleware

import (
	"net/http"

	"github.com/7StaSH7/halfway-diploma/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var AuthModule = fx.Module("auth_middleware",
	fx.Provide(NewAuthMiddleware),
)

type AuthMiddleware struct {
	jwtService utils.JWTService
	logger     *zap.Logger
}

type AuthMiddlewareParams struct {
	fx.In
	JWTService utils.JWTService
	Logger     *zap.Logger
}

func NewAuthMiddleware(p AuthMiddlewareParams) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: p.JWTService,
		logger:     p.Logger,
	}
}

func (m *AuthMiddleware) RequireAuth(c *gin.Context) {
	token, err := c.Cookie(utils.AuthCookieName)
	if err != nil {
		m.logger.Debug("no auth cookie found", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid request",
		})
		return
	}

	claims, err := m.jwtService.ValidateToken(token)
	if err != nil {
		m.logger.Warn("invalid token", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
		})
		return
	}

	c.Set(utils.UserIDKey, claims.UserID)

	m.logger.Debug("user authenticated",
		zap.String("user_id", claims.UserID),
	)

	c.Next()

}

func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(utils.UserIDKey)
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", false
	}

	return userIDStr, true
}
