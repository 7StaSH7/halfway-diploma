package config

import (
	"flag"

	"github.com/caarlos0/env/v11"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/fx"
)

var ServerModule = fx.Module("config",
	fx.Provide(NewServerConfig),
)

type ServerConfig struct {
	LogLevel             string `env:"LOG_LEVEL"`
	Address              string `env:"RUN_ADDRESS"`
	JWTSecret            string `env:"JWT_SECRET"`
	AccuralSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	ReadTimeout          int    `env:"READ_TIMEOUT"`
	WriteTimeout         int    `env:"WRITE_TIMEOUT"`
	IdleTimeout          int    `env:"IDLE_TIMEOUT"`
	DatabaseConfig
}

func NewServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{}

	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "host to listen on")
	flag.StringVar(&cfg.JWTSecret, "js", "mysuperdupersecret", "secret key for jwt")
	flag.StringVar(&cfg.AccuralSystemAddress, "r", "http://localhost:8081", "address to accural system")
	flag.IntVar(&cfg.ReadTimeout, "rt", 15, "read timeout")
	flag.IntVar(&cfg.WriteTimeout, "wt", 15, "write timeout")
	flag.IntVar(&cfg.IdleTimeout, "it", 60, "idle timeout")

	if err := NewDatabaseConfig(cfg); err != nil {
		return nil, err
	}

	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
