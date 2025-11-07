package handler

import (
	"net/http"

	"github.com/7StaSH7/halfway-diploma/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var HealthModule = fx.Module("health_handler",
	fx.Provide(
		NewHealthHandler,
		fx.Annotate(
			NewHealthRoute,
			fx.ResultTags(`group:"routes"`),
		),
	),
)

type HealthHandler interface {
	Ping(*gin.Context)
}

type healthHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

type HealthHandlerParams struct {
	fx.In
	Pool   *pgxpool.Pool
	Logger *zap.Logger
}

func NewHealthHandler(p HealthHandlerParams) HealthHandler {
	return &healthHandler{
		pool:   p.Pool,
		logger: p.Logger,
	}
}

func (h *healthHandler) Ping(c *gin.Context) {
	if h.pool == nil {
		h.logger.Error("database pool is nil")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "database connection is nil",
		})
		return
	}

	if err := h.pool.Ping(c.Request.Context()); err != nil {
		h.logger.Error("database health check failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"database": "connected",
	})
}

type HealthRoute struct {
	handler HealthHandler
	logger  *zap.Logger
}

type HealthRouteParams struct {
	fx.In
	Handler HealthHandler
}

func NewHealthRoute(p HealthRouteParams) router.Route {
	return &HealthRoute{
		handler: p.Handler,
	}
}

func (r *HealthRoute) RegisterHandler(router *gin.Engine) {
	router.GET("/api/ping", r.handler.Ping)
}
