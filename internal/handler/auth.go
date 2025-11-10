package handler

import (
	"errors"
	"net/http"

	"github.com/7StaSH7/halfway-diploma/internal/dto"
	"github.com/7StaSH7/halfway-diploma/internal/router"
	"github.com/7StaSH7/halfway-diploma/internal/service"
	"github.com/7StaSH7/halfway-diploma/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var AuthModule = fx.Module("auth_handler",
	fx.Provide(
		NewAuthHandler,
		fx.Annotate(
			NewAuthRoute,
			fx.ResultTags(`group:"routes"`),
		),
	),
)

type AuthHandler interface {
	Register(*gin.Context)
	Login(*gin.Context)
}

type authHandler struct {
	authService service.AuthService
	jwtService  utils.JWTService
	logger      *zap.Logger
}

type AuthHandlerParams struct {
	fx.In
	AuthService service.AuthService
	JWTService  utils.JWTService
	Logger      *zap.Logger
}

func NewAuthHandler(p AuthHandlerParams) AuthHandler {
	return &authHandler{
		authService: p.AuthService,
		jwtService:  p.JWTService,
		logger:      p.Logger,
	}
}

func (h *authHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := "registration failed"

		if errors.Is(err, errors.New("user already exists")) {
			statusCode = http.StatusConflict
			errorMessage = "invalid credentials"
		}

		c.AbortWithStatusJSON(statusCode, gin.H{
			"error": errorMessage,
		})
		return
	}

	h.setAuthCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"user": user.ToResponse(),
	})
}

func (h *authHandler) Login(c *gin.Context) {
	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		code := http.StatusInternalServerError
		errMsg := "authentication failed"

		if err.Error() == "invalid credentials" {
			code = http.StatusUnauthorized
			errMsg = "invalid credentials"
		}

		c.AbortWithStatusJSON(code, gin.H{
			"error": errMsg,
		})
		return
	}

	h.setAuthCookie(c, token)
	h.logger.Info("user logged in successfully",
		zap.String("user_id", user.ID),
		zap.String("username", user.Username))

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToResponse(),
	})
}

func (h *authHandler) setAuthCookie(c *gin.Context, token string) {
	c.SetCookie(
		utils.AuthCookieName,
		token,
		utils.CookieMaxAge,
		"/",
		"",
		false,
		true,
	)
}

type AuthRoute struct {
	handler AuthHandler
}

type AuthRouteParams struct {
	fx.In
	Handler AuthHandler
}

func NewAuthRoute(p AuthRouteParams) router.Route {
	return &AuthRoute{
		handler: p.Handler,
	}
}

func (r *AuthRoute) RegisterHandler(router *gin.Engine) {
	api := router.Group("/api/user")
	api.POST("/register", r.handler.Register)
	api.POST("/login", r.handler.Login)
}
