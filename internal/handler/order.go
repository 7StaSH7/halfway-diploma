package handler

import (
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

var OrderModule = fx.Module("order_handler",
	fx.Provide(
		NewOrderHandler,
		fx.Annotate(
			NewOrderRoute,
			fx.ResultTags(`group:"routes"`),
		),
	),
)

type OrderHandler interface {
	LoadOrder(*gin.Context)
	GetOrders(*gin.Context)
}

type orderHandler struct {
	logger       *zap.Logger
	orderService service.OrderService
}

type OrderHandlerParams struct {
	fx.In
	Logger       *zap.Logger
	OrderService service.OrderService
}

func NewOrderHandler(p OrderHandlerParams) OrderHandler {
	return &orderHandler{
		logger:       p.Logger,
		orderService: p.OrderService,
	}
}

func (h *orderHandler) LoadOrder(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid request",
		})
		return
	}

	var orderNumber string
	if err := c.ShouldBindPlain(&orderNumber); err != nil {
		h.logger.Error("invalid request", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	if err := utils.ValidateLuhn(orderNumber); err != nil {
		h.logger.Warn("invalid order number format", zap.String("order", orderNumber), zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error": "invalid request",
		})
		return
	}

	ctx := c.Request.Context()
	orderModel, err := h.orderService.CreateOrder(ctx, userID, orderNumber)
	if err != nil {
		if err.Error() == "order already exists" {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "order already exists",
			})
			return
		}

		h.logger.Error("failed to save order to database",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("order_number", orderNumber))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to process order",
		})
		return
	}

	statusCode := http.StatusAccepted
	if orderModel.Status != "NEW" || orderModel.Accrual != 0 {
		statusCode = http.StatusOK
	}

	h.logger.Info("order loaded successfully",
		zap.String("user_id", userID),
		zap.String("order_number", orderNumber),
		zap.String("order_id", orderModel.ID))

	c.JSON(statusCode, gin.H{
		"number": orderNumber,
	})
}

func (h *orderHandler) GetOrders(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid request",
		})
		return
	}

	ctx := c.Request.Context()
	orders, err := h.orderService.GetUserOrders(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user orders from database",
			zap.Error(err),
			zap.String("user_id", userID))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get user orders",
		})
		return
	}

	responseOrders := make([]dto.OrderResponse, 0)
	for _, order := range orders {
		responseOrders = append(responseOrders, dto.ToOrderResponse(order))
	}

	c.JSON(http.StatusOK, responseOrders)
}

type OrderRoute struct {
	handler        OrderHandler
	authMiddleware *middleware.AuthMiddleware
}

type OrderRouteParams struct {
	fx.In
	Handler        OrderHandler
	AuthMiddleware *middleware.AuthMiddleware
}

func NewOrderRoute(p OrderRouteParams) router.Route {
	return &OrderRoute{
		handler:        p.Handler,
		authMiddleware: p.AuthMiddleware,
	}
}

func (r *OrderRoute) RegisterHandler(router *gin.Engine) {
	api := router.Group("/api/user")
	api.POST("/orders", r.authMiddleware.RequireAuth, r.handler.LoadOrder)
	api.GET("/orders", r.authMiddleware.RequireAuth, r.handler.GetOrders)
}
