package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"strconv"

	"clortho/internal/models"
	"clortho/internal/store"
)

func GetLicenseCheckLogsHandler(logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		licenseKey := c.Query("license_key")
		productID := c.Query("product_id")
		productGroupID := c.Query("product_group_id")
		statusCodeStr := c.Query("status_code")

		var statusCode *int
		if statusCodeStr != "" {
			code, err := strconv.Atoi(statusCodeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status_code parameter"})
				return
			}
			statusCode = &code
		}

		pagination := ParsePaginationParams(c)

		if licenseKey != "" {
			logs, totalCount, err := logStore.GetLicenseCheckLogsByLicenseKey(ctx, licenseKey, statusCode, pagination)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch license check logs"})
				return
			}
			
			if logs == nil {
				logs = []models.LicenseCheckLog{}
			}

			totalPages := 0
			if pagination.Limit > 0 {
				totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
			}

			c.JSON(http.StatusOK, models.PaginatedList[models.LicenseCheckLog]{
				Items:      logs,
				TotalCount: totalCount,
				Page:       pagination.Page,
				Limit:      pagination.Limit,
				TotalPages: totalPages,
			})
			return
		}

		if productID != "" {
			logs, totalCount, err := logStore.GetLicenseCheckLogsByProductID(ctx, productID, statusCode, pagination)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch license check logs"})
				return
			}
			
			if logs == nil {
				logs = []models.LicenseCheckLog{}
			}

			totalPages := 0
			if pagination.Limit > 0 {
				totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
			}

			c.JSON(http.StatusOK, models.PaginatedList[models.LicenseCheckLog]{
				Items:      logs,
				TotalCount: totalCount,
				Page:       pagination.Page,
				Limit:      pagination.Limit,
				TotalPages: totalPages,
			})
			return
		}

		if productGroupID != "" {
			logs, totalCount, err := logStore.GetLicenseCheckLogsByProductGroupID(ctx, productGroupID, statusCode, pagination)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch license check logs"})
				return
			}
			
			if logs == nil {
				logs = []models.LicenseCheckLog{}
			}

			totalPages := 0
			if pagination.Limit > 0 {
				totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
			}

			c.JSON(http.StatusOK, models.PaginatedList[models.LicenseCheckLog]{
				Items:      logs,
				TotalCount: totalCount,
				Page:       pagination.Page,
				Limit:      pagination.Limit,
				TotalPages: totalPages,
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing filter parameter (license_key, product_id, or product_group_id)"})
	}
}

func GetAdminLogsHandler(logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var ownerID *string
		if idStr := c.Query("owner_id"); idStr != "" {
			ownerID = &idStr
		}

		pagination := ParsePaginationParams(c)

		logs, totalCount, err := logStore.ListAdminLogs(ctx, ownerID, pagination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch admin logs"})
			return
		}

		if logs == nil {
			logs = []models.AdminLog{}
		}

		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
		}

		c.JSON(http.StatusOK, models.PaginatedList[models.AdminLog]{
			Items:      logs,
			TotalCount: totalCount,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: totalPages,
		})
	}
}
