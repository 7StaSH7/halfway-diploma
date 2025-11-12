package main

import (
	"flag"
	"log"
	"os"

	"github.com/7StaSH7/halfway-diploma/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	Up   = "up"
	Down = "down"
)

const (
	migrationDir = "file://migrations"
)

func main() {
	dir := parseArgs()

	cfg := &config.ServerConfig{}
	if err := config.NewDatabaseConfig(cfg); err != nil {
		log.Fatal(err)
	}

	m, err := migrate.New(migrationDir, cfg.URL)
	if err != nil {
		log.Fatal(err)
	}

	switch dir {
	case Up:
		if err := up(m); err != nil {
			log.Fatal(err)
		}
		log.Println("migration up completed")
		os.Exit(0)
	case Down:
		if err := down(m); err != nil {
			log.Fatal(err)
		}
		log.Println("migration down completed")
		os.Exit(0)
	default:
		{
			log.Fatal("invalid migration direction")
			os.Exit(0)
		}
	}
}

func parseArgs() string {
	var dir string
	flag.StringVar(&dir, "dir", "up", "migration direction (up/down)")
	flag.Parse()

	return dir
}

func up(m *migrate.Migrate) error {
	if err := m.Up(); err != nil {
		return err
	}
	return nil
}

func down(m *migrate.Migrate) error {
	if err := m.Down(); err != nil {
		return err
	}
	return nil
}
