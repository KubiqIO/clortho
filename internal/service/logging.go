package service

import (
	"context"
	"log/slog"

	"clortho/internal/models"
	"clortho/internal/store"
)

func AsyncLogAdminAction(ctx context.Context, logStore store.LogStore, entry *models.AdminLog) {
	slog.Info("Admin Action",
		"action", entry.Action,
		"entity_type", entry.EntityType,
		"entity_id", entry.EntityID,
		"owner_id", entry.OwnerID,
	)

	go func() {
		if err := logStore.CreateAdminLog(context.Background(), entry); err != nil {
			slog.Error("Failed to create admin log", "error", err, "action", entry.Action)
		}
	}()
}


func AsyncLogLicenseCheck(ctx context.Context, logStore store.LogStore, entry *models.LicenseCheckLog, valid bool, reason string) {
	slog.Info("License Check",
		"key", entry.LicenseKey,
		"valid", valid,
		"reason", reason,
		"ip", entry.IPAddress,
		"status", entry.StatusCode,
	)

	go func() {
		if err := logStore.CreateLicenseCheckLog(context.Background(), entry); err != nil {
			slog.Error("Failed to create license check log", "error", err, "key", entry.LicenseKey)
		}
	}()
}
