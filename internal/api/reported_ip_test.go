package api

import (
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

// A client validating through a CDN may report the address it actually runs
// on as usage.ip=. On auto-tracking licenses that reported address — not the
// connection's source — is what the allowlist collects, so the list counts
// the machines the software lives on rather than whichever edge proxies
// relayed the checks. Licenses enforcing a hand-pinned allowlist must keep
// ignoring the claim: it is self-reported.
func TestCheckLicense_ReportedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	mockProductStore.On("GetProduct", mock.Anything, mock.Anything).Return(&models.Product{}, nil).Maybe()
	mockLogStore := new(MockLogStore)

	mockLogStore.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	router := gin.New()
	router.SetTrustedProxies([]string{"0.0.0.0/0"})
	router.GET("/check", handlers.CheckLicenseHandler(mockLicenseStore, mockProductStore, "", mockLogStore))

	check := func(key, forwardedFor, query string) map[string]interface{} {
		req, _ := http.NewRequest("GET", "/check"+query, nil)
		req.Header.Set("X-License-Key", key)
		req.Header.Set("X-Forwarded-For", forwardedFor)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}

	t.Run("Reported_IP_AutoAdded_Instead_Of_Connection", func(t *testing.T) {
		key := "TEST-REPORTED-ADD"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{},
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 2,
			Status:             models.LicenseStatusActive,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()
		mockLicenseStore.On("UpdateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return len(l.AllowedIPs) == 1 && l.AllowedIPs[0] == "10.0.1.5"
		})).Return(nil).Once()

		resp := check(key, "203.0.113.7", "?usage.ip=10.0.1.5")
		assert.True(t, resp["valid"].(bool), "Should be valid after auto-adding the reported IP")
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("Reported_IP_Already_Allowed_Despite_Edge_Churn", func(t *testing.T) {
		key := "TEST-REPORTED-STABLE"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{"10.0.1.5"},
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 2,
			Status:             models.LicenseStatusActive,
		}

		// Two checks arriving via different edge addresses: neither consumes
		// a slot, because the reported address never changed.
		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Twice()

		for _, edge := range []string{"203.0.113.7", "203.0.113.8"} {
			resp := check(key, edge, "?usage.ip=10.0.1.5")
			assert.True(t, resp["valid"].(bool))
		}
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("Reported_IP_Ignored_On_Manual_Allowlist", func(t *testing.T) {
		key := "TEST-REPORTED-SPOOF"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{"10.0.0.1"},
			AutoAllowedIP:      false,
			AutoAllowedIPLimit: 0,
			Status:             models.LicenseStatusActive,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()

		resp := check(key, "203.0.113.9", "?usage.ip=10.0.0.1")
		assert.False(t, resp["valid"].(bool), "A claimed address must not pass a hand-pinned allowlist")
		assert.Equal(t, "IP address not allowed", resp["reason"])
		mockLicenseStore.AssertExpectations(t)
	})

	t.Run("Garbage_Reported_IP_Falls_Back_To_Connection", func(t *testing.T) {
		key := "TEST-REPORTED-GARBAGE"
		license := &models.License{
			ID:                 uuid.New(),
			Key:                key,
			Type:               models.LicenseTypePerpetual,
			AllowedIPs:         []string{},
			AutoAllowedIP:      true,
			AutoAllowedIPLimit: 2,
			Status:             models.LicenseStatusActive,
		}

		mockLicenseStore.On("GetLicenseByKey", mock.Anything, key).Return(license, nil).Once()
		mockLicenseStore.On("UpdateLicense", mock.Anything, mock.MatchedBy(func(l *models.License) bool {
			return len(l.AllowedIPs) == 1 && l.AllowedIPs[0] == "192.168.1.100"
		})).Return(nil).Once()

		resp := check(key, "192.168.1.100", "?usage.ip=not-an-ip")
		assert.True(t, resp["valid"].(bool))
		mockLicenseStore.AssertExpectations(t)
	})
}
