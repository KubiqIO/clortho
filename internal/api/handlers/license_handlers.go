package handlers

import (

	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"clortho/internal/models"
	"clortho/internal/service"
	"clortho/internal/store"
)

type generateLicenseRequest struct {
	ProductID       string             `json:"product_id" binding:"required"`
	Type            models.LicenseType `json:"type" binding:"required"`
	ExpiresAt       *time.Time         `json:"expires_at"`
	Duration        string             `json:"duration"`
	Prefix          string             `json:"prefix"`
	Length          int                `json:"length"`
	FeatureCodes    []string           `json:"feature_codes"`
	ReleaseVersions []string           `json:"release_versions"`
	AllowedIPs      []string           `json:"allowed_ips"`
	AllowedNetworks []string           `json:"allowed_networks"`
	OwnerID         *string            `json:"owner_id"`
}

type updateLicenseRequest struct {
	Type            models.LicenseType `json:"type"`
	ExpiresAt       *time.Time         `json:"expires_at"`
	Duration        string             `json:"duration"`
	Prefix          string             `json:"prefix"`
	Length          int                `json:"length"`
	FeatureCodes    []string           `json:"feature_codes"`
	ReleaseVersions []string           `json:"release_versions"`
	AllowedIPs      []string           `json:"allowed_ips"`
	AllowedNetworks []string           `json:"allowed_networks"`
	Status          models.LicenseStatus `json:"status"`
	OwnerID         *string              `json:"owner_id"`
}

// CheckLicenseHandler handles GET /check
func CheckLicenseHandler(licenseStore store.LicenseStore, productStore store.ProductStore, responseSigningPrivateKey string, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-License-Key")
		
		logEntry := &models.LicenseCheckLog{
			RequestPayload: map[string]interface{}{
				"version": c.Query("version"),
				"feature": c.Query("feature"),
			},
			LicenseKey: key,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			CreatedAt: time.Now(),
		}
		
		defer func() {
			service.AsyncLogLicenseCheck(c.Request.Context(), logStore, logEntry, logEntry.StatusCode == http.StatusOK && (logEntry.ResponsePayload["valid"] == true), func() string {
				if r, ok := logEntry.ResponsePayload["reason"].(string); ok {
					return r
				}
				return ""
			}())
		}()

		key, ok := requireLicenseKey(c)
		if !ok {
			return
		}

		license, err := licenseStore.GetLicenseByKey(c.Request.Context(), key)
		if err != nil {
			logEntry.StatusCode = http.StatusNotFound
			logEntry.ResponsePayload = map[string]interface{}{"error": "License not found"}
			c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
			return
		}
		
		logEntry.ProductID = &license.ProductID
		logEntry.LicenseID = &license.ID

		valid := true
		var reason string

		// Check expiration
		if license.Status == models.LicenseStatusRevoked {
			valid = false
			reason = "License is revoked"
		} else if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
			valid = false
			reason = "License has expired"
		}

		// Check IP restrictions
		if valid && (len(license.AllowedIPs) > 0 || len(license.AllowedNetworks) > 0) {
			clientIPStr := c.ClientIP()
			clientIP := net.ParseIP(clientIPStr)
			if clientIP == nil {
				valid = false
				reason = "Unable to determine client IP for validation"
			} else {
				ipAllowed := false

				if len(license.AllowedIPs) > 0 {
					for _, allowedIPStr := range license.AllowedIPs {
						allowedIP := net.ParseIP(allowedIPStr)
						if allowedIP == nil {
							ip, _, err := net.ParseCIDR(allowedIPStr)
							if err == nil {
								allowedIP = ip
							} else {
								slog.Error("Failed to parse allowed IP", "ip_str", allowedIPStr)
								continue
							}
						}
						if allowedIP.Equal(clientIP) {
							ipAllowed = true
							break
						}
					}
				}

				// Check AllowedNetworks if not already allowed
				if !ipAllowed && len(license.AllowedNetworks) > 0 {
					for _, allowedNetStr := range license.AllowedNetworks {
						_, subnet, err := net.ParseCIDR(allowedNetStr)
						if err == nil && subnet.Contains(clientIP) {
							ipAllowed = true
							break
						}
					}
				}

				if !ipAllowed {
					valid = false
					reason = "IP address not allowed"
				}
			}
		}

		// Check version if query param is provided
		version := c.Query("version")
		if version != "" && valid {
			versionAllowed := false
			if len(license.Releases) == 0 {
				versionAllowed = true
			} else {
				for _, v := range license.Releases {
					if v == version {
						versionAllowed = true
						break
					}
				}
			}
			if !versionAllowed {
				valid = false
				reason = "License not valid for version " + version
			}
		}

		// Check feature if query param is provided
		feature := c.Query("feature")
		if feature != "" && valid {
			featureAllowed := false
			for _, code := range license.Features {
				if code == feature {
					featureAllowed = true
					break
				}
			}
			if !featureAllowed {
				valid = false
				reason = "Feature not enabled: " + feature
			}
		}

		response := gin.H{
			"valid":      valid,
			"expires_at": license.ExpiresAt,
		}
		if reason != "" {
			response["reason"] = reason
		}

		if responseSigningPrivateKey != "" {
			// Generate signed response token (JWT)
			token, err := service.SignLicense(responseSigningPrivateKey, key, license.ExpiresAt, valid, license.Features)
			if err != nil {
				slog.Error("Failed to generate response signing token", "error", err, "key", key)
			} else {
				response["token"] = token
			}
		}

		logEntry.StatusCode = http.StatusOK
		logEntry.ResponsePayload = map[string]interface{}(response)
		c.JSON(http.StatusOK, response)
	}
}

// GenerateLicenseHandler handles POST /admin/keys
func GenerateLicenseHandler(licenseStore store.LicenseStore, productStore store.ProductStore, productGroupStore store.ProductGroupStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req generateLicenseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		slog.Info("Generating license", "product_id", req.ProductID, "owner_id", req.OwnerID)

		if req.ExpiresAt != nil && req.Duration != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot specify both expires_at and duration"})
			return
		}

		var expiresAt *time.Time
		if req.ExpiresAt != nil {
			expiresAt = req.ExpiresAt
		} else if req.Duration != "" {
			exp, err := ParseExpirationDuration(req.Duration)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid duration format: %v", err)})
				return
			}
			expiresAt = &exp
		}

		product, err := productStore.GetProduct(c.Request.Context(), req.ProductID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product_id or product not found"})
			return
		}

		// Resolve settings with inheritance from ProductGroup
		prefix := product.LicensePrefix
		separator := product.LicenseSeparator
		charsetRaw := product.LicenseCharset
		length := req.Length
		if length == 0 {
			length = product.LicenseLength
		}

		// If product belongs to a group, inherit missing settings
		if product.ProductGroupID != nil {
			group, err := productGroupStore.GetProductGroup(c.Request.Context(), product.ProductGroupID.String())
			if err == nil {
				if prefix == "" {
					prefix = group.LicensePrefix
				}
				if separator == "" || separator == "-" {
					if group.LicenseSeparator != "" {
						separator = group.LicenseSeparator
					}
				}
				if charsetRaw == "" {
					charsetRaw = group.LicenseCharset
				}
				if length == 0 {
					length = group.LicenseLength
				}
			}
		}

		if length <= 0 {
			length = 12 // Default length
		}

		if prefix == "" {
			prefix = "LICENSE"
		}
		if req.Prefix != "" {
			prefix = req.Prefix
		}

		parsedCharset, err := service.ParseCharset(charsetRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid charset configuration: %v", err)})
			return
		}

		key, err := service.GenerateLicenseKey(prefix, length, separator, parsedCharset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate license key"})
			return
		}

		productID := product.ID

		license := &models.License{
			ID:              uuid.New(),
			Key:             key,
			OwnerID:         req.OwnerID,
			Type:            req.Type,
			ProductID:       productID,
			ExpiresAt:       expiresAt,
			AllowedIPs:      req.AllowedIPs,
			AllowedNetworks: req.AllowedNetworks,
			Status:          models.LicenseStatusActive,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if len(req.FeatureCodes) > 0 {
			license.Features = req.FeatureCodes
		}

		if len(req.ReleaseVersions) > 0 {
			license.Releases = req.ReleaseVersions
		}

		if err := licenseStore.CreateLicense(c.Request.Context(), license); err != nil {
			slog.Error("Failed to create license", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save license"})
			return
		}

		slog.Info("License generated", "license_key", license.Key, "product_id", productID)

		details := map[string]interface{}(nil)
		dt, _ := json.Marshal(license)
		json.Unmarshal(dt, &details)

		logEntry := &models.AdminLog{
			Action:     "GENERATE_LICENSE",
			EntityType: "LICENSE",
			EntityID:   &license.ID,
			OwnerID:    req.OwnerID,
			Details:    details,
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusCreated, license)
	}
}

// RevokeLicenseHandler handles DELETE /admin/keys
func RevokeLicenseHandler(licenseStore store.LicenseStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, ok := requireLicenseKey(c)
		if !ok {
			return
		}

		slog.Info("Revoking license", "key", key)

		license, err := licenseStore.GetLicenseByKey(c.Request.Context(), key)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
				return
			}
			slog.Error("Failed to get license for revocation", "error", err, "key", key)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke license"})
			return
		}

		license.Status = models.LicenseStatusRevoked
		if err := licenseStore.UpdateLicense(c.Request.Context(), license); err != nil {
			slog.Error("Failed to revoke license", "error", err, "key", key)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke license"})
			return
		}

		slog.Info("License revoked", "key", key)

		logEntry := &models.AdminLog{
			Action:     "REVOKE_LICENSE",
			EntityType: "LICENSE",
			EntityID:   &license.ID,
			OwnerID:    license.OwnerID,
			Details:    map[string]interface{}{"key": key},
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, gin.H{"message": "License revoked"})
	}
}

// DeleteLicenseHandler handles DELETE /admin/keys/purge
func DeleteLicenseHandler(licenseStore store.LicenseStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, ok := requireLicenseKey(c)
		if !ok {
			return
		}

		slog.Info("Deleting license permanently", "key", key)

		// Fetch license first to get details for logging
		license, err := licenseStore.GetLicenseByKey(c.Request.Context(), key)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
				return
			}
			slog.Error("Failed to get license for deletion", "error", err, "key", key)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete license"})
			return
		}

		err = licenseStore.DeleteLicense(c.Request.Context(), key)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				// Should not happen if GetLicenseByKey succeeded, but handle anyway
				c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
				return
			}
			slog.Error("Failed to delete license", "error", err, "key", key)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete license"})
			return
		}

		slog.Info("License deleted permanently", "key", key)

		logEntry := &models.AdminLog{
			Action:     "DELETE_LICENSE",
			EntityType: "LICENSE",
			EntityID:   &license.ID,
			OwnerID:    license.OwnerID,
			Details:    map[string]interface{}{"key": key},
			CreatedAt:  time.Now(),
		}
		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, gin.H{"message": "License deleted permanently"})
	}
}

// UpdateLicenseHandler handles PUT /admin/keys
func UpdateLicenseHandler(licenseStore store.LicenseStore, logStore store.LogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, ok := requireLicenseKey(c)
		if !ok {
			return
		}

		var req updateLicenseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		existing, err := licenseStore.GetLicenseByKey(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
			return
		}

		if req.Type != "" {
			existing.Type = req.Type
		}

		if req.ExpiresAt != nil {
			existing.ExpiresAt = req.ExpiresAt
		} else if req.Duration != "" {
			exp, err := ParseExpirationDuration(req.Duration)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid duration format: %v", err)})
				return
			}
			existing.ExpiresAt = &exp
		}

		if req.AllowedIPs != nil {
			existing.AllowedIPs = req.AllowedIPs
		}
		if req.AllowedNetworks != nil {
			existing.AllowedNetworks = req.AllowedNetworks
		}

		if req.FeatureCodes != nil {
			existing.Features = req.FeatureCodes
		}

		if req.ReleaseVersions != nil {
			existing.Releases = req.ReleaseVersions
		}

		if req.Status != "" {
			existing.Status = req.Status
		}

		// Set UpdatedAt locally to avoid re-fetch, as store now respects this field
		existing.UpdatedAt = time.Now()

		if err := licenseStore.UpdateLicense(c.Request.Context(), existing); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update license"})
			return
		}

		details := map[string]interface{}(nil)
		dt, _ := json.Marshal(req)
		json.Unmarshal(dt, &details)
		logEntry := &models.AdminLog{
			Action:     "UPDATE_LICENSE",
			EntityType: "LICENSE",
			EntityID:   &existing.ID,
			OwnerID:    req.OwnerID,
			Details:    details,
			CreatedAt:  time.Now(),
		}
		if existing.ID != uuid.Nil {
			logEntry.EntityID = &existing.ID
		}

		service.AsyncLogAdminAction(c.Request.Context(), logStore, logEntry)

		c.JSON(http.StatusOK, existing)
	}
}

// GetLicenseHandler handles GET /admin/keys
func GetLicenseHandler(licenseStore store.LicenseStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-License-Key")
		
		// If key is empty, return all licenses (with optional filter)
		if key == "" {
			var ownerID *string
			if idStr := c.Query("owner_id"); idStr != "" {
				ownerID = &idStr
			}

			pagination := ParsePaginationParams(c)

			licenses, totalCount, err := licenseStore.ListLicenses(c.Request.Context(), ownerID, pagination)
			if err != nil {
				slog.Error("Failed to list licenses", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list licenses"})
				return
			}

			// Ensure licenses is an empty slice instead of nil for JSON consistency
			if licenses == nil {
				licenses = []models.License{}
			}

			totalPages := 0
			if pagination.Limit > 0 {
				totalPages = (totalCount + pagination.Limit - 1) / pagination.Limit
			}

			response := models.PaginatedList[models.License]{
				Items:      licenses,
				TotalCount: totalCount,
				Page:       pagination.Page,
				Limit:      pagination.Limit,
				TotalPages: totalPages,
			}

			c.JSON(http.StatusOK, response)
			return
		}

		// If key is present, return specific license
		license, err := licenseStore.GetLicenseByKey(c.Request.Context(), key)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
				return
			}
			slog.Error("Failed to get license", "error", err, "key", key)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get license"})
			return
		}

		c.JSON(http.StatusOK, license)
	}
}

func requireLicenseKey(c *gin.Context) (string, bool) {
	key := c.GetHeader("X-License-Key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-License-Key header is required"})
		return "", false
	}
	return key, true
}
