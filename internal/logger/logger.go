package logger

import (
	"github.com/7StaSH7/halfway-diploma/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("logger",
	fx.Provide(NewLogger),
)

func NewLogger(cfg *config.ServerConfig) (*zap.Logger, error) {
	level, err := zap.ParseAtomicLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = level

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	logger.Info("logger initialized", zap.String("level", cfg.LogLevel))
	return logger, nil
}
