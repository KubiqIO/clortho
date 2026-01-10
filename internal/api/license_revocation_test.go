package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"clortho/internal/api/handlers"
	"clortho/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRevokeAndPurgeLicense(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("RevokeLicense_SoftDelete", func(t *testing.T) {
		mockLicenseStore := new(MockLicenseStore)
		mockLogStore := new(MockLogStore)
		// Allow async logging
		mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

		router := gin.New()
		router.DELETE("/admin/keys", handlers.RevokeLicenseHandler(mockLicenseStore, mockLogStore))

		key := "test-revoke-key"
		license := &models.License{
			ID:     uuid.New(),
			Key:    key,
			Status: models.LicenseStatusActive,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil)
		mockLicenseStore.On("UpdateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.Key == key && l.Status == models.LicenseStatusRevoked
		})).Return(nil)

		req, _ := http.NewRequest("DELETE", "/admin/keys", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("PurgeLicense_HardDelete", func(t *testing.T) {
		mockLicenseStore := new(MockLicenseStore)
		mockLogStore := new(MockLogStore)
		mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

		router := gin.New()
		router.DELETE("/admin/keys/purge", handlers.DeleteLicenseHandler(mockLicenseStore, mockLogStore))

		key := "test-purge-key"
		license := &models.License{
			ID:  uuid.New(),
			Key: key,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil)
		mockLicenseStore.On("DeleteLicense", mock.Anything, key).Return(nil)

		req, _ := http.NewRequest("DELETE", "/admin/keys/purge", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("CheckLicense_Revoked", func(t *testing.T) {
		mockLicenseStore := new(MockLicenseStore)
		mockProductStore := new(MockProductStore)
		mockLogStore := new(MockLogStore)
		mockLogStore.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()

		router := gin.New()
		router.GET("/check", handlers.CheckLicenseHandler(mockLicenseStore, mockProductStore, "", mockLogStore))

		key := "test-revoked-check"
		license := &models.License{
			ID:     uuid.New(),
			Key:    key,
			Status: models.LicenseStatusRevoked,
			Type:   models.LicenseTypePerpetual,
		}
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil)

		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp["valid"].(bool))
		assert.Equal(t, "License is revoked", resp["reason"])
	})

	t.Run("UpdateLicenseStatus", func(t *testing.T) {
		mockLicenseStore := new(MockLicenseStore)
		mockLogStore := new(MockLogStore)
		mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

		router := gin.New()
		router.PUT("/admin/keys", handlers.UpdateLicenseHandler(mockLicenseStore, mockLogStore))

		key := "test-status-update-key"
		existingLicense := &models.License{
			ID:     uuid.New(),
			Key:    key,
			Status: models.LicenseStatusRevoked,
			Type:   models.LicenseTypePerpetual,
		}

		type updateLicenseRequest struct {
			Status models.LicenseStatus `json:"status"`
		}
		reqBody := updateLicenseRequest{
			Status: models.LicenseStatusActive,
		}
		body, _ := json.Marshal(reqBody)

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(existingLicense, nil).Once()
		mockLicenseStore.On("UpdateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.Key == key && l.Status == models.LicenseStatusActive
		})).Return(nil)


		req, _ := http.NewRequest("PUT", "/admin/keys", bytes.NewBuffer(body))
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockLicenseStore.AssertExpectations(t)
	})
}
