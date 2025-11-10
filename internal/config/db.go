package config

import (
	"flag"

	"github.com/caarlos0/env"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/fx"
)

type DatabaseConfig struct {
	URL     string `env:"DATABASE_DSN"`
	MaxConn int    `env:"DATABASE_MAX_CONN"`
	MinConn int    `env:"DATABASE_MIN_CONN"`
}

var DatabaseModule = fx.Module("database_config",
	fx.Provide(NewDatabaseConfig),
)

func NewDatabaseConfig() (*DatabaseConfig, error) {
	cfg := &DatabaseConfig{}

	flag.StringVar(&cfg.URL, "d", "postgres://postgres:postgres@localhost:5432/diploma?search_path=public&sslmode=disable", "db url")
	flag.IntVar(&cfg.MaxConn, "maxcon", 25, "max pool size")
	flag.IntVar(&cfg.MinConn, "minconn", 5, "min pool size")

	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
