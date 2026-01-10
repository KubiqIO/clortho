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

type createReleaseRequest struct {
	Version        string  `json:"version" binding:"required"`
	OwnerID        *string `json:"owner_id"`
	ProductID      *string `json:"product_id"`
	ProductGroupID *string `json:"product_group_id"`
}

type updateReleaseRequest struct {
	Version string `json:"version" binding:"required"`
}



// UpdateReleaseHandler handles PUT /admin/releases/:releaseId
func UpdateReleaseHandler(releaseStore store.ReleaseStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		releaseID := c.Param("releaseId")
		var req updateReleaseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rID, err := uuid.Parse(releaseID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid release ID"})
			return
		}

		release := &models.Release{
			ID:      rID,
			Version: req.Version,
		}
		
		// Fetch existing release to get owner_id for log
		existingRelease, _ := releaseStore.GetRelease(c.Request.Context(), releaseID)

		if err := releaseStore.UpdateRelease(c.Request.Context(), release); err != nil {
			if err.Error() == "release not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Release not found"})
				return
			}
			slog.Error("Failed to update release", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update release"})
			return
		}

		// Audit Log
		logEntry := &models.AdminLog{
			Action:     "UPDATE_RELEASE",
			EntityType: "releases",
			EntityID:   &rID,
			OwnerID:    func() *string { if existingRelease != nil { return existingRelease.OwnerID }; return nil }(),
			Details: map[string]interface{}{
				"request": req,
			},
			CreatedAt: time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, gin.H{"message": "Release updated"})
	}
}

// DeleteReleaseHandler handles DELETE /admin/releases/:releaseId
func DeleteReleaseHandler(releaseStore store.ReleaseStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		releaseID := c.Param("releaseId")
		
		// Fetch existing release to get owner_id for log
		existingRelease, _ := releaseStore.GetRelease(c.Request.Context(), releaseID)

		if err := releaseStore.DeleteRelease(c.Request.Context(), releaseID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete release"})
			return
		}
		// Audit Log
		rID, _ := uuid.Parse(releaseID)
		logEntry := &models.AdminLog{
			Action:     "DELETE_RELEASE",
			EntityType: "releases",
			EntityID:   &rID,
			OwnerID:    func() *string { if existingRelease != nil { return existingRelease.OwnerID }; return nil }(),
			Details:    map[string]interface{}{},
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, gin.H{"message": "Release deleted"})
	}
}

// CreateReleaseHandler handles POST /admin/releases
func CreateReleaseHandler(releaseStore store.ReleaseStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createReleaseRequest
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

		release := &models.Release{
			ID:             uuid.New(),
			OwnerID:        req.OwnerID,
			ProductID:      prodID,
			ProductGroupID: groupID,
			Version:        req.Version,
			CreatedAt:      time.Now(),
		}

		if err := releaseStore.CreateRelease(c.Request.Context(), release); err != nil {
			slog.Error("Failed to create release", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create release"})
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
			Action:     "CREATE_RELEASE",
			EntityType: "releases",
			EntityID:   &release.ID,
			OwnerID:    release.OwnerID,
			Details:    logDetails,
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusCreated, release)
	}
}

// ListGlobalReleasesHandler handles GET /admin/releases/global
func ListGlobalReleasesHandler(releaseStore store.ReleaseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ownerID *string
		if idStr := c.Query("owner_id"); idStr != "" {
			ownerID = &idStr
		}

		pagination := ParsePaginationParams(c)

		releases, totalCount, err := releaseStore.ListGlobalReleases(c.Request.Context(), ownerID, pagination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list releases"})
			return
		}

		// Ensure releases is an empty slice instead of nil for JSON consistency
		if releases == nil {
			releases = []models.Release{}
		}

		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
		}

		response := models.PaginatedList[models.Release]{
			Items:      releases,
			TotalCount: totalCount,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}

// ListAllReleasesHandler handles GET /admin/releases
func ListAllReleasesHandler(releaseStore store.ReleaseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ownerID *string
		if idStr := c.Query("owner_id"); idStr != "" {
			ownerID = &idStr
		}

		pagination := ParsePaginationParams(c)

		var releases []models.Release
		var totalCount int
		var err error

		if productID := c.Query("product_id"); productID != "" {
			releases, totalCount, err = releaseStore.ListReleasesByProduct(c.Request.Context(), productID, ownerID, pagination)
		} else if productGroupID := c.Query("product_group_id"); productGroupID != "" {
			releases, totalCount, err = releaseStore.ListReleasesByProductGroup(c.Request.Context(), productGroupID, ownerID, pagination)
		} else {
			releases, totalCount, err = releaseStore.ListAllReleases(c.Request.Context(), ownerID, pagination)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list all releases"})
			return
		}

		// Ensure releases is an empty slice instead of nil for JSON consistency
		if releases == nil {
			releases = []models.Release{}
		}

		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
		}

		response := models.PaginatedList[models.Release]{
			Items:      releases,
			TotalCount: totalCount,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}

// GetReleaseHandler handles GET /admin/releases/:id
func GetReleaseHandler(releaseStore store.ReleaseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		release, err := releaseStore.GetRelease(c.Request.Context(), id)
		if err != nil {
			if err.Error() == "release not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Release not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch release"})
			return
		}
		c.JSON(http.StatusOK, release)
	}
}
