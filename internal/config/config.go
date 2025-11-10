package config

import (
	"flag"

	"github.com/caarlos0/env"
	"go.uber.org/fx"
)

var ServerModule = fx.Module("config",
	fx.Provide(NewServerConfig),
)

type ServerConfig struct {
	LogLevel             string `env:"LOG_LEVEL"`
	Address              string `env:"ADDRESS"`
	JWTSecret            string `env:"JWT_SECRET"`
	AccuralSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func NewServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{}

	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "host to listen on")
	flag.StringVar(&cfg.JWTSecret, "js", "mysuperdupersecret", "secret key for jwt")
	flag.StringVar(&cfg.AccuralSystemAddress, "r", "http://localhost:8081", "address to accural system")

	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
