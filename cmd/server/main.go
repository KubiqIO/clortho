package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"clortho/internal/api"
	"clortho/internal/config"
	"clortho/internal/database"
	"clortho/internal/store"
	"clortho/internal/version"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := database.Migrate(cfg.DatabaseURL, "migrations"); err != nil {
		slog.Info("Migration error (may be safe if no changes)", "error", err)
	}

	ctx := context.Background()
	pool, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	licenseStore := store.NewPostgresLicenseStore(pool)
	productStore := store.NewPostgresProductStore(pool)
	productGroupStore := store.NewPostgresProductGroupStore(pool)
	releaseStore := store.NewPostgresReleaseStore(pool)
	featureStore := store.NewPostgresFeatureStore(pool)
	logStore := store.NewPostgresLogStore(pool)
	statsStore := store.NewPostgresStatsStore(pool)

	server := api.NewServer(cfg, pool, licenseStore, productStore, productGroupStore, releaseStore, featureStore, logStore, statsStore)

	slog.Info("Clortho the Keymaster ("+version.Version+") is now onduty", "port", cfg.Port)
	if err := server.Router.Run(":" + cfg.Port); err != nil {
		slog.Error("Failed to run server", "error", err)
		os.Exit(1)
	}
}
