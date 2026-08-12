package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"clortho/internal/api/handlers"
	"clortho/internal/models"
)

// checkTokenClaims drives GET /check with the given query string and returns the
// verified claims of the signed response token, plus the decoded JSON body.
func checkTokenClaims(t *testing.T, query string) (jwt.MapClaims, map[string]interface{}) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	mockLicenseStore := new(MockLicenseStore)
	mockProductStore := new(MockProductStore)
	mockLogStore := new(MockLogStore)
	mockLogStore.On("CreateLicenseCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()

	expiresAt := time.Now().Add(24 * time.Hour)
	mockLicenseStore.On("GetLicenseByKey", mock.Anything, "testkey").Return(&models.License{
		Key:       "testkey",
		ExpiresAt: &expiresAt,
		Status:    models.LicenseStatusActive,
	}, nil)

	router := gin.New()
	router.GET("/check", handlers.CheckLicenseHandler(
		mockLicenseStore,
		mockProductStore,
		base64.StdEncoding.EncodeToString(privKey),
		mockLogStore,
	))

	req, _ := http.NewRequest("GET", "/check"+query, nil)
	req.Header.Set("X-License-Key", "testkey")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	tokenStr, ok := resp["token"].(string)
	require.True(t, ok, "response should carry a signed token")

	token, err := jwt.Parse(tokenStr, func(*jwt.Token) (interface{}, error) { return pubKey, nil })
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	return claims, resp
}

// usageClaim pulls the reported_usage object out of decoded claims.
func usageClaim(t *testing.T, claims jwt.MapClaims) (map[string]interface{}, bool) {
	t.Helper()
	raw, present := claims["reported_usage"]
	if !present {
		return nil, false
	}
	usage, ok := raw.(map[string]interface{})
	require.True(t, ok, "reported_usage must decode as an object")
	return usage, true
}

func TestCheckLicenseHandler_ReportedUsageEcho(t *testing.T) {
	t.Run("echoes every usage field verbatim", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.clients=200&usage.ips=20")
		usage, present := usageClaim(t, claims)
		require.True(t, present)
		assert.Equal(t, map[string]interface{}{"clients": "200", "ips": "20"}, usage)
	})

	t.Run("accepts field names it has never seen", func(t *testing.T) {
		// The prefix is the whole contract; Clortho holds no list of field names.
		claims, _ := checkTokenClaims(t, "?usage.tenants=9&usage.gpu_hours=13.5&usage.wxyz=1")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, map[string]interface{}{"tenants": "9", "gpu_hours": "13.5", "wxyz": "1"}, usage)
	})

	t.Run("absent when no usage param arrived", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "")
		_, present := claims["reported_usage"]
		assert.False(t, present, "a stripped report must not be defaulted to 0")
	})

	t.Run("a stripped report is distinguishable from a reported zero", func(t *testing.T) {
		stripped, _ := checkTokenClaims(t, "")
		zero, _ := checkTokenClaims(t, "?usage.clients=0")

		_, present := stripped["reported_usage"]
		assert.False(t, present)

		usage, _ := usageClaim(t, zero)
		assert.Equal(t, "0", usage["clients"])
	})

	t.Run("a stripped field is visible as a missing key", func(t *testing.T) {
		// A proxy dropping usage.ips leaves a report that is still well formed,
		// so the client detects it by the key it sent no longer being there.
		claims, _ := checkTokenClaims(t, "?usage.clients=200")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, "200", usage["clients"])
		assert.NotContains(t, usage, "ips")
	})

	t.Run("a blanked value is echoed as present and empty", func(t *testing.T) {
		// "?usage.clients=" reached the server, so the field is present and the
		// client sees its "200" did not survive the trip.
		claims, _ := checkTokenClaims(t, "?usage.clients=")
		usage, _ := usageClaim(t, claims)
		val, present := usage["clients"]
		assert.True(t, present)
		assert.Equal(t, "", val)
	})

	t.Run("caps each value at 64 chars", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.clients="+strings.Repeat("7", 200)+"&usage.ips=20")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, strings.Repeat("7", 64), usage["clients"])
		assert.Equal(t, "20", usage["ips"])
	})

	t.Run("echoes a percent-encoded value as decoded", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.clients=12%20units")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, "12 units", usage["clients"])
	})

	t.Run("echoes a percent-encoded field name as decoded", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.gpu%20hours=4")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, "4", usage["gpu hours"])
	})

	t.Run("a repeated field collapses to the first value", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.clients=200&usage.clients=1")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, "200", usage["clients"])
	})

	t.Run("ignores a prefix with no field name", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.=5")
		_, present := claims["reported_usage"]
		assert.False(t, present)
	})
}

func TestCheckLicenseHandler_NonceEcho(t *testing.T) {
	t.Run("echoes the nonce verbatim", func(t *testing.T) {
		nonce := "b3Nz-Qq7_1kXpL0m"
		claims, _ := checkTokenClaims(t, "?nonce="+nonce)
		assert.Equal(t, nonce, claims["nonce"])
	})

	t.Run("absent when no nonce arrived", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.clients=5")
		_, present := claims["nonce"]
		assert.False(t, present)
	})

	t.Run("ignores a nonce longer than 64 chars", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?nonce="+strings.Repeat("z", 65))
		_, present := claims["nonce"]
		assert.False(t, present)
	})

	t.Run("carries both echoes together", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.clients=200&usage.ips=20&nonce=abc123")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, map[string]interface{}{"clients": "200", "ips": "20"}, usage)
		assert.Equal(t, "abc123", claims["nonce"])
	})

	t.Run("nonce is not swept into the usage report", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "?usage.clients=200&nonce=abc123")
		usage, _ := usageClaim(t, claims)
		assert.NotContains(t, usage, "nonce")
	})
}

// TestCheckLicenseHandler_EchoDoesNotAffectValidation covers the deploy-safety
// promise: the new params change nothing an existing client can observe apart
// from the added claims, and unknown params stay ignored as before.
func TestCheckLicenseHandler_EchoDoesNotAffectValidation(t *testing.T) {
	t.Run("response shape is unchanged", func(t *testing.T) {
		_, resp := checkTokenClaims(t, "?usage.clients=200&nonce=abc123")

		assert.Equal(t, true, resp["valid"])
		assert.NotEmpty(t, resp["expires_at"])
		assert.NotContains(t, resp, "reason")
		// The echoes live only inside the signed token, where no intermediary
		// can rewrite them.
		assert.NotContains(t, resp, "reported_usage")
		assert.NotContains(t, resp, "nonce")
		assert.Len(t, resp, 3, "response keys should stay valid/expires_at/token")
	})

	t.Run("unprefixed params remain ignored", func(t *testing.T) {
		// This is what the prefix buys: extras from older integrations, and a
		// bare "clients" that looks like a usage field, stay out of the token.
		claims, resp := checkTokenClaims(t, "?usage.clients=200&clients=999&legacy_param=1&whatever=x")

		assert.Equal(t, true, resp["valid"])
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, map[string]interface{}{"clients": "200"}, usage)
		assert.NotContains(t, claims, "legacy_param")
		assert.NotContains(t, claims, "whatever")
	})

	t.Run("version and feature are not swept into the usage report", func(t *testing.T) {
		// Reserved params sit outside the namespace, so they cannot collide.
		claims, resp := checkTokenClaims(t, "?usage.clients=200&version=2.0.0&feature=sso")

		assert.Equal(t, false, resp["valid"], "feature sso is not enabled on this licence")
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, map[string]interface{}{"clients": "200"}, usage)
	})

	t.Run("a request without either param carries todays claims plus iat", func(t *testing.T) {
		claims, _ := checkTokenClaims(t, "")

		keys := make([]string, 0, len(claims))
		for k := range claims {
			keys = append(keys, k)
		}
		assert.ElementsMatch(t, []string{"sub", "iss", "valid", "features", "exp", "iat"}, keys)
	})

	t.Run("token carries iat for freshness checks", func(t *testing.T) {
		before := time.Now().Unix()
		claims, _ := checkTokenClaims(t, "?nonce=abc123")

		iat, ok := claims["iat"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, int64(iat), before)
		assert.LessOrEqual(t, int64(iat), time.Now().Unix())
	})
}
