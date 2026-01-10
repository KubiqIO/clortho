package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"clortho/internal/api/handlers"
	"clortho/internal/models"
)

func TestParseExpirationDuration(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		param   string
		val     int
	}{
		{"1d", false, "d", 1},
		{"3w", false, "w", 3},
		{"1mo", false, "mo", 1},
		{"2y", false, "y", 2},
		{"invalid", true, "", 0},
		{"", true, "", 0},
		{"10d", false, "d", 10},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := handlers.ParseExpirationDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExpirationDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
				if !tt.wantErr {
				now := time.Now()
				var expected time.Time
				switch tt.param {
				case "d":
					expected = now.AddDate(0, 0, tt.val)
				case "w":
					expected = now.AddDate(0, 0, tt.val*7)
				case "mo":
					expected = now.AddDate(0, tt.val, 0)
				case "y":
					expected = now.AddDate(tt.val, 0, 0)
				}

				// Allow small delta
				diff := got.Sub(expected)
				if diff > time.Second || diff < -time.Second {
				}
			}
		})
	}
}

func TestGenerateLicenseHandler_Duration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)

	router := gin.New()
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockProductGroupStore := new(MockProductGroupStore)
	router.POST("/admin/keys", handlers.GenerateLicenseHandler(mockLicenseStore, mockProductStore, mockProductGroupStore, mockLogStore))

	t.Run("Success with duration", func(t *testing.T) {
		pID := uuid.New()
		type generateLicenseRequest struct {
			ProductID string             `json:"product_id"`
			Type      models.LicenseType `json:"type"`
			Duration  string             `json:"duration"`
			Prefix    string             `json:"prefix"`
		}
		reqBody := generateLicenseRequest{
			ProductID: pID.String(),
			Type:      models.LicenseTypeTimed,
			Duration:  "2w",
			Prefix:    "TEST",
		}
		body, _ := json.Marshal(reqBody)

		// Mock Product Get
		mockProductStore.On("GetProduct", mock.Anything, pID.String()).Return(&models.Product{
			ID:            pID,
			LicensePrefix: "DEFAULT",
		}, nil)


		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			if l.ExpiresAt == nil {
				return false
			}

			expected := time.Now().AddDate(0, 0, 14)
			diff := l.ExpiresAt.Sub(expected)
			return diff < time.Minute && diff > -time.Minute
		})).Return(nil)

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("Error both duration and expires_at", func(t *testing.T) {
		now := time.Now()
		type generateLicenseRequest struct {
			ProductID string             `json:"product_id"`
			Type      models.LicenseType `json:"type"`
			Duration  string             `json:"duration"`
			ExpiresAt *time.Time         `json:"expires_at"`
		}
		reqBody := generateLicenseRequest{
			ProductID: uuid.New().String(),
			Type:      models.LicenseTypeTimed,
			Duration:  "2w",
			ExpiresAt: &now,
		}

		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCheckLicenseHandler_ReturnsExpiration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	router := gin.New()
	router.GET("/check", handlers.CheckLicenseHandler(mockLicenseStore, mockProductStore, "", mockLogStore))

	t.Run("Returns Expiration", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		license := &models.License{
			Key:       "testkey",
			ExpiresAt: &expiresAt,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, "testkey").Return(license, nil)

		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", "testkey")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)



		expStr, ok := resp["expires_at"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, expStr)
	})
}
