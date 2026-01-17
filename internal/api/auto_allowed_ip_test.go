package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"clortho/internal/api/handlers"
	"clortho/internal/models"
)

func TestGenerateLicense_AutoIPInheritance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	mockProductGroupStore := new(MockProductGroupStore)
	mockLogStore := new(MockLogStore)
	// Allow any log creation
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router := gin.New()
	router.POST("/admin/keys", handlers.GenerateLicenseHandler(mockLicenseStore, mockProductStore, mockProductGroupStore, mockLogStore))

	t.Run("Inherit_From_Product", func(t *testing.T) {
		productID := uuid.New()
		product := &models.Product{
			ID:                 productID,
			Name:               "Product Only",
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 5,
		}

		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       "perpetual",
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil).Once()
		
		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.AutoAllowedIP == true && l.AutoAllowedIPLimit == 5
		})).Return(nil).Once()

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockProductStore.AssertExpectations(t)
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("Inherit_From_Group_When_Product_Default", func(t *testing.T) {
		productID := uuid.New()
		groupID := uuid.New()
		
		group := &models.ProductGroup{
			ID:                 groupID,
			Name:               "Group Settings",
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 10,
		}

		product := &models.Product{
			ID:             productID,
			Name:           "Product Inherit",
			ProductGroupID: &groupID,
			AutoAllowedIP:      false,
			AutoAllowedIPLimit: 0,
		}

		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       "perpetual",
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil).Once()
		mockProductGroupStore.On("GetProductGroup", mock.Anything, groupID.String()).Return(group, nil).Once()

		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.AutoAllowedIP == true && l.AutoAllowedIPLimit == 10
		})).Return(nil).Once()

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockProductStore.AssertExpectations(t)
		mockProductGroupStore.AssertExpectations(t)
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("Product_Overrides_Group", func(t *testing.T) {
		productID := uuid.New()
		groupID := uuid.New()
		
		group := &models.ProductGroup{
			ID:                 groupID,
			Name:               "Group Settings",
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 10,
		}

		product := &models.Product{
			ID:             productID,
			Name:           "Product Override",
			ProductGroupID: &groupID,
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 3,
		}

		reqBody := map[string]interface{}{
			"product_id": productID.String(),
			"type":       "perpetual",
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil).Once()

		mockProductGroupStore.On("GetProductGroup", mock.Anything, groupID.String()).Return(group, nil).Maybe()

		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.AutoAllowedIP == true && l.AutoAllowedIPLimit == 3
		})).Return(nil).Once()

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockProductStore.AssertExpectations(t)
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("Request_Overrides_All", func(t *testing.T) {
		productID := uuid.New()
		
		product := &models.Product{
			ID:                 productID,
			Name:               "Product Default",
			AutoAllowedIP:      false,
			AutoAllowedIPLimit: 0,
		}

		reqBody := map[string]interface{}{
			"product_id":            productID.String(),
			"type":                  "perpetual",
			"auto_allowed_ip":       true,
			"auto_allowed_ip_limit": 99,
		}
		body, _ := json.Marshal(reqBody)

		mockProductStore.On("GetProduct", mock.Anything, productID.String()).Return(product, nil).Once()

		mockLicenseStore.On("CreateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.AutoAllowedIP == true && l.AutoAllowedIPLimit == 99
		})).Return(nil).Once()

		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockProductStore.AssertExpectations(t)
		mockLicenseStore.AssertExpectations(t)
	})
}

func TestCheckLicense_AutoIPLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	mockLogStore := new(MockLogStore)

	mockLogStore.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router := gin.New()
	router.SetTrustedProxies([]string{"0.0.0.0/0"})
	router.GET("/check", handlers.CheckLicenseHandler(mockLicenseStore, mockProductStore, "", mockLogStore))

	t.Run("AutoAllowedIP_AddSuccess", func(t *testing.T) {
		key := "TEST-AUTO-ADD"
		ip := "192.168.1.100"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{},
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 2,
			Status:             models.LicenseStatusActive,
			ExpiresAt:          nil,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()
		
		mockLicenseStore.On("UpdateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return len(l.AllowedIPs) == 1 && l.AllowedIPs[0] == ip
		})).Return(nil).Once()

		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", key)
		req.Header.Set("X-Forwarded-For", ip)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["valid"].(bool), "Should be valid after auto-add")
		
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("AutoAllowedIP_LimitReached", func(t *testing.T) {
		key := "TEST-LIMIT-REACHED"
		ip := "192.168.1.101"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{"10.0.0.1", "10.0.0.2"},
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 2,
			Status:             models.LicenseStatusActive,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()
		
		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", key)
		req.Header.Set("X-Forwarded-For", ip)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp["valid"].(bool), "Should be invalid due to IP limit")
		assert.Equal(t, "IP address not allowed", resp["reason"])
		
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("AutoAllowedIP_AlreadyAllowed", func(t *testing.T) {
		key := "TEST-ALREADY-ALLOWED"
		ip := "10.0.0.1"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{"10.0.0.1"},
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 5,
			Status:             models.LicenseStatusActive,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", key)
		req.Header.Set("X-Forwarded-For", ip)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp["valid"].(bool))
		
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("AutoAllowedIP_Disabled", func(t *testing.T) {
		key := "TEST-DISABLED"
		ip := "192.168.1.100"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{"10.0.0.1"},
			AutoAllowedIP:      false,
			AutoAllowedIPLimit: 5,
			Status:             models.LicenseStatusActive,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", key)
		req.Header.Set("X-Forwarded-For", ip)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp["valid"].(bool))
		assert.Equal(t, "IP address not allowed", resp["reason"])
		
		mockLicenseStore.AssertExpectations(t)
	})
}

func TestUpdateLicense_AutoAllowedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockLogStore := new(MockLogStore)

	// Allow any log creation
	mockLogStore.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router := gin.New()
	router.PUT("/admin/keys", handlers.UpdateLicenseHandler(mockLicenseStore, mockLogStore))

	t.Run("Update_AutoAllowedIP_Settings", func(t *testing.T) {
		key := "TEST-UPDATE-AUTO-IP"
		licenseID := uuid.New()
		
		existing := &models.License{
			ID:                 licenseID,
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AutoAllowedIP:      false,
			AutoAllowedIPLimit: 0,
			Status:             models.LicenseStatusActive,
		}

		autoAllowedIP := true
		limit := 10
		
		reqBody := map[string]interface{}{
			"auto_allowed_ip":       true,
			"auto_allowed_ip_limit": 10,
		}
		body, _ := json.Marshal(reqBody)

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(existing, nil).Once()
		
		mockLicenseStore.On("UpdateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return l.ID == licenseID && l.AutoAllowedIP == autoAllowedIP && l.AutoAllowedIPLimit == limit
		})).Return(nil).Once()

		req, _ := http.NewRequest("PUT", "/admin/keys", bytes.NewBuffer(body))
		req.Header.Set("X-License-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var resp models.License
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp.AutoAllowedIP)
		assert.Equal(t, 10, resp.AutoAllowedIPLimit)
		
		mockLicenseStore.AssertExpectations(t)
	})
}
