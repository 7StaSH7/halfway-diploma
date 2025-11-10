package database

import (
	"context"
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var DriverModule = fx.Module("database_driver",
	fx.Provide(NewPostgresDriver),
	fx.Invoke(
		VerifyConnections,
		RunMigrations,
	),
)

type DriverParams struct {
	fx.In
	Config *config.ServerConfig
	Logger *zap.Logger
}
type queryTracer struct {
	log *zap.Logger
}

func (tracer *queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	tracer.log.Debug("executing query",
		zap.String("sql", data.SQL),
		zap.Any("args", data.Args),
	)
	return ctx
}

func (tracer *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if data.Err != nil {
		tracer.log.Error("query failed",
			zap.Error(data.Err),
		)
	} else {
		tracer.log.Debug("query completed",
			zap.Int64("rows_affected", data.CommandTag.RowsAffected()),
		)
	}
}

func NewPostgresDriver(lc fx.Lifecycle, p DriverParams) (*pgxpool.Pool, error) {
	conf, err := pgxpool.ParseConfig(p.Config.URL)
	if err != nil {
		p.Logger.Error("failed to parse database config", zap.Error(err))
		return nil, err
	}

	conf.ConnConfig.Tracer = &queryTracer{
		log: p.Logger,
	}
	conf.MaxConns = int32(p.Config.MaxConn)
	conf.MinConns = int32(p.Config.MinConn)
	conf.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), conf)
	if err != nil {
		p.Logger.Error("failed to create connection pool", zap.Error(err))
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pool.Ping(ctx); err != nil {
				p.Logger.Error("database connection failed", zap.Error(err))
				return err
			}
			p.Logger.Info("database connection verified successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			pool.Close()
			p.Logger.Info("database connection pool closed")
			return nil
		},
	})

	return pool, nil
}

func VerifyConnections(lc fx.Lifecycle, pool *pgxpool.Pool, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var result int
			err := pool.QueryRow(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				logger.Error("database health check failed", zap.Error(err))
				return err
			}

			if result != 1 {
				logger.Error("unexpected database health check result", zap.Int("result", result))
				return err
			}

			logger.Info("database health check passed")
			return nil
		},
	})
}

func RunMigrations(lc fx.Lifecycle, cfg *config.ServerConfig, logger *zap.Logger) {
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				return autoMigrate(cfg.URL, logger)
			},
		},
	)
}

func autoMigrate(dsn string, logger *zap.Logger) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Error("failed to create migration instance", zap.Error(err))
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Info("no migrations to run")
			return nil
		}
		logger.Error("migration failed", zap.Error(err))
		return err
	}

	logger.Info("migrations completed successfully")
	return nil
}
