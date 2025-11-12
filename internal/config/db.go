package config

import (
	"flag"
)

type DatabaseConfig struct {
	URL     string `env:"DATABASE_URI"`
	MaxConn int    `env:"DATABASE_MAX_CONN"`
	MinConn int    `env:"DATABASE_MIN_CONN"`
}

func NewDatabaseConfig(cfg *ServerConfig) error {
	flag.StringVar(&cfg.URL, "d", "postgres://postgres:postgres@localhost:5432/diploma?search_path=public&sslmode=disable", "db url")
	flag.IntVar(&cfg.MaxConn, "maxcon", 25, "max pool size")
	flag.IntVar(&cfg.MinConn, "minconn", 5, "min pool size")

	return nil
}
