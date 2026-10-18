package api

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"clortho/internal/api/handlers"
	"clortho/internal/models"
)

func TestCheckLicense_IPPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	past := time.Now().Add(-time.Hour)
	for _, tc := range []struct {
		name                             string
		productDisabled, licenseDisabled bool
		status                           models.LicenseStatus
		expiry                           *time.Time
		query, reason                    string
	}{
		{name: "default enforces saved IP rules", reason: "IP address not allowed"},
		{name: "HA product", productDisabled: true},
		{name: "individual HA license", licenseDisabled: true},
		{name: "both disabled", productDisabled: true, licenseDisabled: true},
		{name: "revocation", licenseDisabled: true, status: models.LicenseStatusRevoked, reason: "License is revoked"},
		{name: "expiry", productDisabled: true, expiry: &past, reason: "License has expired"},
		{name: "version", licenseDisabled: true, query: "?version=2.0", reason: "License not valid for version 2.0"},
		{name: "feature", productDisabled: true, query: "?feature=enterprise", reason: "Feature not enabled: enterprise"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			licenses, products, logs := new(MockLicenseStore), new(MockProductStore), new(MockLogStore)
			license := &models.License{
				ID: uuid.New(), Key: "HA-KEY", ProductID: uuid.New(),
				AllowedIPs: []string{"10.0.0.1"}, AllowedNetworks: []string{"192.168.0.0/24"},
				AutoAllowedIP: true, AutoAllowedIPLimit: 5,
				IPRestrictionsDisabled: tc.licenseDisabled, Status: tc.status, ExpiresAt: tc.expiry,
				Releases: []string{"1.0"},
			}
			licenses.On("GetLicenseByKey", mock.Anything, license.Key).Return(license, nil).Twice()
			// Use a full auto-IP list for the enforcement case; disabled cases
			// have spare capacity and must never register the changing pod IP.
			if tc.reason == "IP address not allowed" {
				license.AutoAllowedIPLimit = 1
			}
			products.On("GetProduct", mock.Anything, license.ProductID.String()).Return(&models.Product{IPRestrictionsDisabled: tc.productDisabled}, nil).Maybe()
			logs.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()
			router := gin.New()
			router.GET("/check", handlers.CheckLicenseHandler(licenses, products, base64.StdEncoding.EncodeToString(priv), logs))
			for _, addr := range []string{"203.0.113.1:1234", "[2001:db8::1]:1234"} {
				req := httptest.NewRequest("GET", "/check"+tc.query, nil)
				req.RemoteAddr = addr
				req.Header.Set("X-License-Key", license.Key)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				require.Equal(t, http.StatusOK, w.Code)
				var response map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.Equal(t, tc.reason == "", response["valid"])
				if tc.reason != "" {
					require.Equal(t, tc.reason, response["reason"])
				}
				token, err := jwt.Parse(response["token"].(string), func(*jwt.Token) (interface{}, error) { return pub, nil }, jwt.WithValidMethods([]string{"EdDSA"}), jwt.WithoutClaimsValidation())
				require.NoError(t, err)
				require.Equal(t, response["valid"], token.Claims.(jwt.MapClaims)["valid"])
			}
			require.Equal(t, []string{"10.0.0.1"}, license.AllowedIPs)
			licenses.AssertNotCalled(t, "UpdateLicense", mock.Anything, mock.Anything)
			licenses.AssertExpectations(t)
		})
	}
}

func TestCheckLicense_IPPolicyLookupFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	licenses, products, logs := new(MockLicenseStore), new(MockProductStore), new(MockLogStore)
	license := &models.License{Key: "HA-KEY", ProductID: uuid.New(), AutoAllowedIP: true}
	licenses.On("GetLicenseByKey", mock.Anything, license.Key).Return(license, nil)
	products.On("GetProduct", mock.Anything, license.ProductID.String()).Return(nil, errors.New("database unavailable"))
	logs.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	router := gin.New()
	router.GET("/check", handlers.CheckLicenseHandler(licenses, products, "", logs))
	req := httptest.NewRequest("GET", "/check", nil)
	req.Header.Set("X-License-Key", license.Key)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Equal(t, "30", w.Header().Get("Retry-After"))
	require.JSONEq(t, `{"error":"License validation service unavailable"}`, w.Body.String())
}

func TestUpdateLicense_IPPolicyPartialUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	licenses, logs := new(MockLicenseStore), new(MockLogStore)
	license := &models.License{ID: uuid.New(), Key: "HA-KEY", AllowedIPs: []string{"10.0.0.1"}, AutoAllowedIP: true, AutoAllowedIPLimit: 1}
	licenses.On("GetLicenseByKey", mock.Anything, license.Key).Return(license, nil)
	licenses.On("UpdateLicense", mock.Anything, license).Return(nil)
	logs.On("CreateAdminLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	router := gin.New()
	router.PUT("/admin/keys", handlers.UpdateLicenseHandler(licenses, logs))
	for _, body := range []string{`{"ip_restrictions_disabled":true}`, `{"feature_codes":[]}`, `{"ip_restrictions_disabled":false}`} {
		req := httptest.NewRequest("PUT", "/admin/keys", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-License-Key", license.Key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, body != `{"ip_restrictions_disabled":false}`, license.IPRestrictionsDisabled)
		require.Equal(t, []string{"10.0.0.1"}, license.AllowedIPs)
		require.True(t, license.AutoAllowedIP)
	}
}
