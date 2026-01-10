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

type createProductRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	LicensePrefix    string `json:"license_prefix"`
	LicenseSeparator string `json:"license_separator"`
	LicenseCharset   string `json:"license_charset"`
	LicenseLength    int    `json:"license_length"`
	LicenseType      models.LicenseType `json:"license_type"`
	LicenseDuration  string `json:"license_duration"`
	ProductGroupID   string `json:"product_group_id"`
	OwnerID          *string `json:"owner_id"`
}

type updateProductRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	LicensePrefix    string `json:"license_prefix"`
	LicenseSeparator string `json:"license_separator"`
	LicenseCharset   string `json:"license_charset"`
	LicenseLength    int    `json:"license_length"`
	LicenseType      models.LicenseType `json:"license_type"`
	LicenseDuration  string `json:"license_duration"`
	ProductGroupID   string `json:"product_group_id"`
	OwnerID          *string `json:"owner_id"`
}

// ListProductsHandler handles GET /admin/products
func ListProductsHandler(productStore store.ProductStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ownerID *string
		if idStr := c.Query("owner_id"); idStr != "" {
			ownerID = &idStr
		}

		pagination := ParsePaginationParams(c)

		products, totalCount, err := productStore.ListProducts(c.Request.Context(), ownerID, pagination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list products"})
			return
		}

		// Ensure products is an empty slice instead of nil for JSON consistency
		if products == nil {
			products = []models.Product{}
		}

		totalPages := 0
		if pagination.Limit > 0 {
			totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
		}

		response := models.PaginatedList[models.Product]{
			Items:      products,
			TotalCount: totalCount,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, response)
	}
}

// CreateProductHandler handles POST /admin/products
func CreateProductHandler(productStore store.ProductStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			slog.Error("Failed to bind product JSON", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		product := &models.Product{
			ID:               uuid.New(),
			OwnerID:          req.OwnerID,
			Name:             req.Name,
			Description:      req.Description,
			LicensePrefix:    req.LicensePrefix,
			LicenseSeparator: req.LicenseSeparator,
			LicenseCharset:   req.LicenseCharset,
			LicenseLength:    req.LicenseLength,
			LicenseType:      req.LicenseType,
			LicenseDuration:  req.LicenseDuration,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if req.ProductGroupID != "" {
			groupID, err := uuid.Parse(req.ProductGroupID)
			if err != nil {
				slog.Error("Invalid product_group_id", "error", err, "group_id", req.ProductGroupID)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product_group_id"})
				return
			}
			product.ProductGroupID = &groupID
		}

		if err := productStore.CreateProduct(c.Request.Context(), product); err != nil {
			slog.Error("Failed to create product", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
			return
		}

		slog.Info("Product created", "product_id", product.ID, "name", product.Name)

		// Audit Log
		logEntry := &models.AdminLog{
			Action:     "CREATE_PRODUCT",
			EntityType: "products",
			EntityID:   &product.ID,
			OwnerID:    product.OwnerID,
			Details: map[string]interface{}{
				"name":    product.Name,
				"request": req,
			},
			CreatedAt: time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusCreated, product)
	}
}

// GetProductHandler handles GET /admin/products/:id
func GetProductHandler(productStore store.ProductStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		product, err := productStore.GetProduct(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusOK, product)
	}
}

// UpdateProductHandler handles PUT /admin/products/:id
func UpdateProductHandler(productStore store.ProductStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req updateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		product, err := productStore.GetProduct(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		if req.Name != "" {
			product.Name = req.Name
		}
		if req.Description != "" {
			product.Description = req.Description
		}
		if req.LicensePrefix != "" {
			product.LicensePrefix = req.LicensePrefix
		}
		if req.LicenseSeparator != "" {
			product.LicenseSeparator = req.LicenseSeparator
		}
		if req.LicenseCharset != "" {
			product.LicenseCharset = req.LicenseCharset
		}
		if req.LicenseLength > 0 {
			product.LicenseLength = req.LicenseLength
		}
		if req.LicenseType != "" {
			product.LicenseType = req.LicenseType
		}
		if req.LicenseDuration != "" {
			product.LicenseDuration = req.LicenseDuration
		}
		if req.ProductGroupID != "" {
			groupID, err := uuid.Parse(req.ProductGroupID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product_group_id"})
				return
			}
			product.ProductGroupID = &groupID
		}

		product.UpdatedAt = time.Now()

		if err := productStore.UpdateProduct(c.Request.Context(), product); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
			return
		}

		// Audit Log
		productID := product.ID
		logEntry := &models.AdminLog{
			Action:     "UPDATE_PRODUCT",
			EntityType: "products",
			EntityID:   &productID,
			OwnerID:    product.OwnerID,
			Details: map[string]interface{}{
				"request": req,
			},
			CreatedAt: time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, product)
	}
}

// DeleteProductHandler handles DELETE /admin/products/:id
func DeleteProductHandler(productStore store.ProductStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		
		// Fetch product to get owner_id for log
		product, err := productStore.GetProduct(c.Request.Context(), id)
		if err != nil {
			if err.Error() == "product not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product for deletion"})
			return
		}

		if err := productStore.DeleteProduct(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
			return
		}

		// Audit Log
		pID := product.ID
		logEntry := &models.AdminLog{
			Action:     "DELETE_PRODUCT",
			EntityType: "products",
			EntityID:   &pID,
			OwnerID:    product.OwnerID,
			Details:    map[string]interface{}{},
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
	}
}
