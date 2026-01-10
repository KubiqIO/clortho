package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"clortho/internal/store"
)

func GetDashboardStatsHandler(statsStore store.StatsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		ownerID := c.Query("owner_id")
		durationStr := c.Query("duration")
		if durationStr == "" {
			durationStr = "30d"
		}

		expiryTime, err := ParseExpirationDuration(durationStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration format. Use '7d', '2w', '1mo' or standard Go duration (e.g. 24h)"})
			return
		}
		duration := expiryTime.Sub(time.Now())
		since := time.Now().Add(-duration)

		var ownerIDPtr *string
		if ownerID != "" {
			ownerIDPtr = &ownerID
		}

		stats, err := statsStore.GetDashboardStats(ctx, ownerIDPtr, &since)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard stats"})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}
