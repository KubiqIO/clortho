package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"clortho/internal/models"
	"clortho/internal/service"
	"clortho/internal/store"
)

type createProductGroupRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	LicensePrefix    string `json:"license_prefix"`
	LicenseSeparator string `json:"license_separator"`
	LicenseCharset   string `json:"license_charset"`
	LicenseLength    int    `json:"license_length"`
	AutoAllowedIP    bool   `json:"auto_allowed_ip"`
	AutoAllowedIPLimit int `json:"auto_allowed_ip_limit"`
	OwnerID          *string `json:"owner_id"`
}

type updateProductGroupRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	LicensePrefix    string `json:"license_prefix"`
	LicenseSeparator string `json:"license_separator"`
	LicenseCharset   string `json:"license_charset"`
	LicenseLength    *int   `json:"license_length"`
	AutoAllowedIP    *bool  `json:"auto_allowed_ip"`
	AutoAllowedIPLimit *int `json:"auto_allowed_ip_limit"`
	OwnerID          *string `json:"owner_id"`
}

// ListProductGroupsHandler handles GET /admin/product-groups
func ListProductGroupsHandler(productGroupStore store.ProductGroupStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ownerID *string
		if idStr := c.Query("owner_id"); idStr != "" {
			ownerID = &idStr
		}

		pagination := ParsePaginationParams(c)

		groups, totalCount, err := productGroupStore.ListProductGroups(c.Request.Context(), ownerID, pagination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list product groups"})
			return
		}

		// Ensure groups is an empty slice instead of nil for JSON consistency
		if groups == nil {
			groups = []models.ProductGroup{}
		}

		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
		}

		response := models.PaginatedList[models.ProductGroup]{
			Items:      groups,
			TotalCount: totalCount,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}

// CreateProductGroupHandler handles POST /admin/product-groups
func CreateProductGroupHandler(productGroupStore store.ProductGroupStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createProductGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		group := &models.ProductGroup{
			ID:               uuid.New(),
			OwnerID:          req.OwnerID,
			Name:             req.Name,
			Description:      req.Description,
			LicensePrefix:    req.LicensePrefix,
			LicenseSeparator: req.LicenseSeparator,
			LicenseCharset:   req.LicenseCharset,
			LicenseLength:    req.LicenseLength,
			AutoAllowedIP:    req.AutoAllowedIP,
			AutoAllowedIPLimit: req.AutoAllowedIPLimit,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		// Audit Log
		details := map[string]interface{}(nil)
		dt, _ := json.Marshal(group)
		json.Unmarshal(dt, &details)

		logEntry := &models.AdminLog{
			Action:     "CREATE_PRODUCT_GROUP",
			OwnerID:    group.OwnerID,
			Details:    details,
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		if err := productGroupStore.CreateProductGroup(c.Request.Context(), group); err != nil {
			slog.Error("Failed to create product group", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product group"})
			return
		}

		slog.Info("Product group created successfully", "id", group.ID)
		c.JSON(http.StatusCreated, group)
	}
}

// GetProductGroupHandler handles GET /admin/product-groups/:id
func GetProductGroupHandler(productGroupStore store.ProductGroupStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		group, err := productGroupStore.GetProductGroup(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product group not found"})
			return
		}
		c.JSON(http.StatusOK, group)
	}
}

// UpdateProductGroupHandler handles PUT /admin/product-groups/:id
func UpdateProductGroupHandler(productGroupStore store.ProductGroupStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req updateProductGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		group, err := productGroupStore.GetProductGroup(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product group not found"})
			return
		}

		if req.Name != "" {
			group.Name = req.Name
		}
		if req.Description != "" {
			group.Description = req.Description
		}
		if req.LicensePrefix != "" {
			group.LicensePrefix = req.LicensePrefix
		}
		if req.LicenseSeparator != "" {
			group.LicenseSeparator = req.LicenseSeparator
		}
		if req.LicenseCharset != "" {
			group.LicenseCharset = req.LicenseCharset
		}
		if req.LicenseLength != nil && *req.LicenseLength > 0 {
			group.LicenseLength = *req.LicenseLength
		}
		if req.AutoAllowedIP != nil {
			group.AutoAllowedIP = *req.AutoAllowedIP
		}
		if req.AutoAllowedIPLimit != nil {
			group.AutoAllowedIPLimit = *req.AutoAllowedIPLimit
		}

		group.UpdatedAt = time.Now()

		if err := productGroupStore.UpdateProductGroup(c.Request.Context(), group); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product group"})
			return
		}

		// Audit Log
		details := map[string]interface{}(nil)
		dt, _ := json.Marshal(req) // Log the changes requested
		json.Unmarshal(dt, &details)

		logEntry := &models.AdminLog{
			Action:     "UPDATE_PRODUCT_GROUP",
			EntityID:   &group.ID,
			OwnerID:    group.OwnerID,
			Details:    details,
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, group)
	}
}

// DeleteProductGroupHandler handles DELETE /admin/product-groups/:id
func DeleteProductGroupHandler(productGroupStore store.ProductGroupStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Fetch group to get owner_id for log
		group, err := productGroupStore.GetProductGroup(c.Request.Context(), id)
		if err != nil {
			if err.Error() == "product group not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product group not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product group for deletion"})
			return
		}

		if err := productGroupStore.DeleteProductGroup(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product group"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Product group deleted"})

		// Audit Log
		pId := group.ID
		logEntry := &models.AdminLog{
			Action:     "DELETE_PRODUCT_GROUP",
			EntityType: "PRODUCT_GROUP",
			EntityID:   &pId,
			OwnerID:    group.OwnerID,
			Details:    nil,
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)
	}
}
