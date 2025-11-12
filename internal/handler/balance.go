package handler

import (
	"errors"
	"net/http"

	"github.com/7StaSH7/halfway-diploma/internal/dto"
	"github.com/7StaSH7/halfway-diploma/internal/middleware"
	"github.com/7StaSH7/halfway-diploma/internal/router"
	"github.com/7StaSH7/halfway-diploma/internal/service"
	"github.com/7StaSH7/halfway-diploma/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var BalanceModule = fx.Module("balance_handler",
	fx.Provide(
		NewBalanceHandler,
		fx.Annotate(
			NewBalanceRoute,
			fx.ResultTags(`group:"routes"`),
		),
	),
)

type BalanceHandler interface {
	GetBalance(*gin.Context)
	Withdraw(*gin.Context)
	GetWithdrawals(*gin.Context)
}

type balanceHandler struct {
	logger         *zap.Logger
	balanceService service.BalanceService
}

type BalanceHandlerParams struct {
	fx.In
	Logger         *zap.Logger
	BalanceService service.BalanceService
}

func NewBalanceHandler(p BalanceHandlerParams) BalanceHandler {
	return &balanceHandler{
		logger:         p.Logger,
		balanceService: p.BalanceService,
	}
}

func (h *balanceHandler) GetBalance(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid request",
		})
		return
	}

	ctx := c.Request.Context()
	current, withdrawn, err := h.balanceService.GetUserBalance(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user balance",
			zap.Error(err),
			zap.String("user_id", userID))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get balance",
		})
		return
	}

	response := dto.BalanceResponse{
		Current:   float64(current) / 100,
		Withdrawn: float64(withdrawn) / 100,
	}

	c.JSON(http.StatusOK, response)
}

func (h *balanceHandler) Withdraw(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid request",
		})
		return
	}

	var req dto.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	if err := utils.ValidateLuhn(req.Order); err != nil {
		h.logger.Warn("invalid order number format", zap.String("order", req.Order), zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error": "invalid order number",
		})
		return
	}

	sum := uint(req.Sum * 100)

	ctx := c.Request.Context()
	err := h.balanceService.Withdraw(ctx, userID, req.Order, sum)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := "failed to process withdrawal"

		switch {
		case errors.Is(err, errors.New("insufficient funds")):
			statusCode = http.StatusPaymentRequired
			errorMessage = "insufficient funds"
		case errors.Is(err, errors.New("invalid order number")):
			statusCode = http.StatusUnprocessableEntity
			errorMessage = "invalid order number"
		case errors.Is(err, errors.New("user not found")):
			statusCode = http.StatusUnauthorized
			errorMessage = "user not found"
		}

		h.logger.Error("failed to process withdrawal",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("order", req.Order),
			zap.Float64("sum", req.Sum))

		c.AbortWithStatusJSON(statusCode, gin.H{
			"error": errorMessage,
		})
		return
	}

	h.logger.Debug("withdrawal processed successfully",
		zap.String("user_id", userID),
		zap.String("order", req.Order),
		zap.Float64("sum", req.Sum))

	c.Status(http.StatusOK)
}

func (h *balanceHandler) GetWithdrawals(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid request",
		})
		return
	}

	ctx := c.Request.Context()
	withdrawals, err := h.balanceService.GetUserWithdrawals(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user withdrawals",
			zap.Error(err),
			zap.String("user_id", userID))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get withdrawals",
		})
		return
	}

	response := make([]dto.WithdrawalResponse, 0, len(withdrawals))
	for _, withdrawal := range withdrawals {
		response = append(response, dto.ToWithdrawalResponse(withdrawal))
	}

	if len(response) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, response)
}

type BalanceRoute struct {
	handler        BalanceHandler
	authMiddleware *middleware.AuthMiddleware
}

type BalanceRouteParams struct {
	fx.In
	Handler        BalanceHandler
	AuthMiddleware *middleware.AuthMiddleware
}

func NewBalanceRoute(p BalanceRouteParams) router.Route {
	return &BalanceRoute{
		handler:        p.Handler,
		authMiddleware: p.AuthMiddleware,
	}
}

func (r *BalanceRoute) RegisterHandler(router *gin.Engine) {
	api := router.Group("/api/user")
	api.GET("/balance", r.authMiddleware.RequireAuth, r.handler.GetBalance)
	api.POST("/balance/withdraw", r.authMiddleware.RequireAuth, r.handler.Withdraw)
	api.GET("/withdrawals", r.authMiddleware.RequireAuth, r.handler.GetWithdrawals)
}
