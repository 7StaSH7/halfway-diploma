package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Route interface {
	RegisterHandler(*gin.Engine)
}

var Module = fx.Module("router",
	fx.Provide(NewRouter),
	fx.Invoke(RegisterAllRoutes),
)

type RouterParams struct {
	fx.In
	Logger *zap.Logger
}

func NewRouter(p RouterParams) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(ginLogger(p.Logger))

	return router
}

type RouteRegistrationParams struct {
	fx.In
	Router *gin.Engine
	Routes []Route `group:"routes"`
	Logger *zap.Logger
}

func RegisterAllRoutes(lc fx.Lifecycle, p RouteRegistrationParams) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			for _, route := range p.Routes {
				route.RegisterHandler(p.Router)
			}

			p.Logger.Info("all routes registered successfully",
				zap.Int("route_count", len(p.Routes)))
			return nil
		},
	})
}

func ginLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)

		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", latency.Milliseconds()),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}
