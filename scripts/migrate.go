package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"clortho/internal/config"
)

func main() {
	var databaseURL string
	var migrationsPath string
	var direction string
	var version int

	flag.StringVar(&databaseURL, "database-url", "", "Database URL")
	flag.StringVar(&migrationsPath, "path", "migrations", "Path to migrations folder")
	flag.StringVar(&direction, "direction", "up", "Migration direction (up/down)")
	flag.IntVar(&version, "version", -1, "Migration version (required for force)")
	flag.Parse()

	if databaseURL == "" {
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		databaseURL = cfg.DatabaseURL
	}
	
	if databaseURL == "" {
		log.Fatal("database-url is required (via flag or config.yaml)")
	}

	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize migrate: %v", err)
	}

	if direction == "up" {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to migrate up: %v", err)
		}
		fmt.Println("Migrated up successfully")
	} else if direction == "down" {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to migrate down: %v", err)
		}
		fmt.Println("Migrated down successfully")
	} else if direction == "force" {
		if version == -1 {
			log.Fatal("version is required for force")
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		fmt.Printf("Forced version to %d successfully\n", version)
	} else if direction == "drop" {
		if err := m.Drop(); err != nil {
			log.Fatalf("Failed to drop database: %v", err)
		}
		fmt.Println("Database dropped successfully")
	} else {
		log.Fatalf("Invalid direction: %s", direction)
	}
}
