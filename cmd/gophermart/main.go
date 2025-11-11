package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/client/accrual"
	"github.com/7StaSH7/halfway-diploma/internal/config"
	"github.com/7StaSH7/halfway-diploma/internal/database"
	"github.com/7StaSH7/halfway-diploma/internal/handler"
	"github.com/7StaSH7/halfway-diploma/internal/logger"
	"github.com/7StaSH7/halfway-diploma/internal/middleware"
	"github.com/7StaSH7/halfway-diploma/internal/repository"
	"github.com/7StaSH7/halfway-diploma/internal/router"
	"github.com/7StaSH7/halfway-diploma/internal/service"
	"github.com/7StaSH7/halfway-diploma/internal/utils"
	"github.com/7StaSH7/halfway-diploma/internal/worker"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		// Core
		config.ServerModule,
		database.DriverModule,
		logger.Module,

		// Utility
		utils.JWTModule,

		// Repositories
		repository.OrderModule,
		repository.UserModule,

		// Middlewares
		middleware.AuthModule,

		// Services
		service.AuthModule,
		service.OrderModule,

		// Clients
		accrual.Module,

		// Workers
		worker.OrderWorkerModule,

		// Handlers
		handler.AuthModule,
		handler.HealthModule,
		handler.OrderModule,

		// Router
		router.Module,

		// Server
		serverModule,

		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	)

	app.Run()
}

var serverModule = fx.Module("server",
	fx.Provide(NewHTTPServer),
	fx.Invoke(RegisterServerHooks),
)

type ServerParams struct {
	fx.In
	Router *gin.Engine
	Config *config.ServerConfig
	Logger *zap.Logger
}

func NewHTTPServer(p ServerParams) *http.Server {
	p.Logger.Info("creating server",
		zap.String("address", p.Config.Address),
		zap.Duration("read_timeout", time.Duration(p.Config.ReadTimeout)*time.Second),
		zap.Duration("write_timeout", time.Duration(p.Config.WriteTimeout)*time.Second),
		zap.Duration("idle_timeout", time.Duration(p.Config.IdleTimeout)*time.Second),
	)

	return &http.Server{
		Addr:         p.Config.Address,
		Handler:      p.Router,
		ReadTimeout:  time.Duration(p.Config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(p.Config.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(p.Config.IdleTimeout) * time.Second,
	}
}

func RegisterServerHooks(lc fx.Lifecycle, server *http.Server, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", server.Addr)
			if err != nil {
				logger.Error("failed to create listener", zap.Error(err), zap.String("address", server.Addr))
				return fmt.Errorf("failed to listen on %s: %w", server.Addr, err)
			}

			logger.Info("server listener created", zap.String("address", ln.Addr().String()))

			go func() {
				if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
					logger.Error("server error", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down server...")

			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error("server shutdown error", zap.Error(err))
				return fmt.Errorf("server shutdown failed: %w", err)
			}

			logger.Info("server shutdown completed successfully")
			return nil
		},
	})
}
