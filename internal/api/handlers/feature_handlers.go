package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"clortho/internal/models"
	"clortho/internal/service"
	"clortho/internal/store"
)

type createFeatureRequest struct {
	Name           string  `json:"name" binding:"required"`
	Code           string  `json:"code" binding:"required"`
	Description    string  `json:"description"`
	OwnerID        *string `json:"owner_id"`
	ProductID      *string `json:"product_id"`
	ProductGroupID *string `json:"product_group_id"`
}

type updateFeatureRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}



// UpdateFeatureHandler handles PUT /admin/features/:featureId
func UpdateFeatureHandler(featureStore store.FeatureStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		featureID := c.Param("featureId")
		var req updateFeatureRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fID, err := uuid.Parse(featureID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feature ID"})
			return
		}

		// Fetch existing feature to get owner_id
		existingFeature, err := featureStore.GetFeature(c.Request.Context(), featureID)
		if err != nil {
			if err.Error() == "feature not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Feature not found"})
				return
			}
		}

		feature := &models.Feature{
			ID:          fID,
			Name:        req.Name,
			Code:        req.Code,
			Description: req.Description,
		}

		if err := featureStore.UpdateFeature(c.Request.Context(), feature); err != nil {
			if err.Error() == "feature not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Feature not found"})
				return
			}
			slog.Error("Failed to update feature", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update feature"})
			return
		}

		// Audit Log
		logEntry := &models.AdminLog{
			Action:     "UPDATE_FEATURE",
			EntityType: "features",
			EntityID:   &fID,
			OwnerID:    existingFeature.OwnerID,
			Details: map[string]interface{}{
				"request": req,
			},
			CreatedAt: time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, gin.H{"message": "Feature updated"})
	}
}

// DeleteFeatureHandler handles DELETE /admin/products/:id/features/:featureId
func DeleteFeatureHandler(featureStore store.FeatureStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		featureID := c.Param("featureId")
		
		// Fetch existing feature to get owner_id for log
		existingFeature, _ := featureStore.GetFeature(c.Request.Context(), featureID)

		if err := featureStore.DeleteFeature(c.Request.Context(), featureID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete feature"})
			return
		}
		// Audit Log
		fID, _ := uuid.Parse(featureID)
		logEntry := &models.AdminLog{
			Action:     "DELETE_FEATURE",
			EntityType: "features",
			EntityID:   &fID,
			OwnerID:    func() *string { if existingFeature != nil { return existingFeature.OwnerID }; return nil }(),
			Details:    map[string]interface{}{},
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, gin.H{"message": "Feature deleted"})
	}
}



// CreateFeatureHandler handles POST /admin/features
func CreateFeatureHandler(featureStore store.FeatureStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createFeatureRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var prodID *uuid.UUID
		if req.ProductID != nil {
			id, err := uuid.Parse(*req.ProductID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product_id"})
				return
			}
			prodID = &id
		}

		var groupID *uuid.UUID
		if req.ProductGroupID != nil {
			id, err := uuid.Parse(*req.ProductGroupID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product_group_id"})
				return
			}
			groupID = &id
		}

		feature := &models.Feature{
			ID:             uuid.New(),
			OwnerID:        req.OwnerID,
			ProductID:      prodID,
			ProductGroupID: groupID,
			Name:           req.Name,
			Code:           req.Code,
			Description:    req.Description,
			CreatedAt:      time.Now(),
		}

		if err := featureStore.CreateFeature(c.Request.Context(), feature); err != nil {
			slog.Error("Failed to create feature", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feature"})
			return
		}

		// Audit Log
		logDetails := map[string]interface{}{
			"request": req,
		}
		if prodID != nil {
			logDetails["scope"] = "product"
			logDetails["product_id"] = prodID.String()
		} else if groupID != nil {
			logDetails["scope"] = "product_group"
			logDetails["product_group_id"] = groupID.String()
		} else {
			logDetails["scope"] = "global"
		}

		logEntry := &models.AdminLog{
			Action:     "CREATE_FEATURE",
			EntityType: "features",
			EntityID:   &feature.ID,
			Details:    logDetails,
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusCreated, feature)
	}
}

// ListGlobalFeaturesHandler handles GET /admin/features/global
func ListGlobalFeaturesHandler(featureStore store.FeatureStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extracting owner_id from query param for filtering
		var ownerID *string
		if idStr := c.Query("owner_id"); idStr != "" {
			ownerID = &idStr
		}

		pagination := ParsePaginationParams(c)

		features, totalCount, err := featureStore.ListGlobalFeatures(c.Request.Context(), ownerID, pagination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list features"})
			return
		}

		// Ensure features is an empty slice instead of nil for JSON consistency
		if features == nil {
			features = []models.Feature{}
		}

		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
		}

		response := models.PaginatedList[models.Feature]{
			Items:      features,
			TotalCount: totalCount,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}

// ListAllFeaturesHandler handles GET /admin/features, allowing filtering by product_id or product_group_id
func ListAllFeaturesHandler(featureStore store.FeatureStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extracting owner_id from query param for filtering
		var ownerID *string
		if idStr := c.Query("owner_id"); idStr != "" {
			ownerID = &idStr
		}

		pagination := ParsePaginationParams(c)

		var features []models.Feature
		var totalCount int
		var err error

		if productID := c.Query("product_id"); productID != "" {
			features, totalCount, err = featureStore.ListFeaturesByProduct(c.Request.Context(), productID, ownerID, pagination)
		} else if productGroupID := c.Query("product_group_id"); productGroupID != "" {
			features, totalCount, err = featureStore.ListFeaturesByProductGroup(c.Request.Context(), productGroupID, ownerID, pagination)
		} else {
			features, totalCount, err = featureStore.ListAllFeatures(c.Request.Context(), ownerID, pagination)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list features"})
			return
		}

		// Ensure features is an empty slice instead of nil for JSON consistency
		if features == nil {
			features = []models.Feature{}
		}

		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
		}

		response := models.PaginatedList[models.Feature]{
			Items:      features,
			TotalCount: totalCount,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}

// GetFeatureHandler handles GET /admin/features/:id
func GetFeatureHandler(featureStore store.FeatureStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		feature, err := featureStore.GetFeature(c.Request.Context(), id)
		if err != nil {
			if err.Error() == "feature not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Feature not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feature"})
			return
		}
		c.JSON(http.StatusOK, feature)
	}
}
