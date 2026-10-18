package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"clortho/internal/config"
	"clortho/internal/database"
	"clortho/internal/models"
	"clortho/internal/store"
	"crypto/ed25519"
	"encoding/base64"
)

func TestIPRestrictions(t *testing.T) {
	ctx := context.Background()

	dbName := "clortho_test_ip"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %s", err)
	}
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %s", err)
		}
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Setup
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	privKeyBase64 := base64.StdEncoding.EncodeToString(privKey)
	_ = pubKey // unused

	cfg := config.Config{
		DatabaseURL:               connStr,
		AdminSecret:               "test-secret",
		ResponseSigningPrivateKey: privKeyBase64,
		RateLimitAdmin: config.RateLimitConfig{
			Enabled: false,
		},
		RateLimitCheck: config.RateLimitConfig{
			Enabled: false,
		},
		TrustedProxies: []string{"127.0.0.1"},
	}

	absPath, _ := filepath.Abs("../../migrations")
	err = database.Migrate(cfg.DatabaseURL, absPath)
	require.NoError(t, err)

	pool, err := database.New(ctx, cfg.DatabaseURL)
	require.NoError(t, err)
	defer pool.Close()

	ls := store.NewPostgresLicenseStore(pool)
	ps := store.NewPostgresProductStore(pool)
	pgs := store.NewPostgresProductGroupStore(pool)
	rs := store.NewPostgresReleaseStore(pool)
	fs := store.NewPostgresFeatureStore(pool)
	logs := store.NewPostgresLogStore(pool)
	statsStore := store.NewPostgresStatsStore(pool)
	server := NewServer(cfg, pool, ls, ps, pgs, rs, fs, logs, statsStore)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "test-admin",
		"iss": "clortho-test",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(cfg.AdminSecret))
	require.NoError(t, err)
	authHeader := "Bearer " + tokenString

	// 1. Create Product
	productID := ""
	{
		reqBody := map[string]interface{}{
			"name":           "IP Test Product",
			"license_prefix": "IPTEST",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/admin/products", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp models.Product
		json.Unmarshal(w.Body.Bytes(), &resp)
		productID = resp.ID.String()
	}

	// 2. Create License with IP restrictions
	var licenseKey string
	{
		reqBody := map[string]interface{}{
			"product_id":       productID,
			"type":             "perpetual",
			"allowed_ips":      []string{"10.0.0.1", "2001:db8::1"},
			"allowed_networks": []string{"192.168.1.0/24"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		licenseKey = resp["key"].(string)
	}

	tests := []struct {
		name        string
		clientIP    string
		expectValid bool
	}{
		{"Allowed IPv4", "10.0.0.1", true},
		{"Allowed IPv6", "2001:db8::1", true},
		{"Allowed Subnet IP", "192.168.1.5", true},
		{"Allowed Subnet IP 2", "192.168.1.200", true},
		{"Disallowed IP", "10.0.0.2", false},
		{"Disallowed Subnet IP", "192.168.2.1", false},
		{"Disallowed IPv6", "2001:db8::2", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/check", nil)
			req.Header.Set("X-License-Key", licenseKey)
			req.Header.Set("X-Forwarded-For", tc.clientIP)
			req.RemoteAddr = "127.0.0.1:1234"

			w := httptest.NewRecorder()
			server.Router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			valid, ok := resp["valid"].(bool)
			require.True(t, ok)
			require.Equal(t, tc.expectValid, valid, "Validation mismatch for IP %s", tc.clientIP)

			if !tc.expectValid {
				reason, _ := resp["reason"].(string)
				require.Equal(t, "IP address not allowed", reason)
			}
		})
	}

	t.Run("HA product and license overrides persist without clearing IP rules", func(t *testing.T) {
		requestJSON := func(method, path, body, key string, output interface{}) {
			req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
			req.Header.Set("Authorization", authHeader)
			req.Header.Set("Content-Type", "application/json")
			if key != "" {
				req.Header.Set("X-License-Key", key)
			}
			w := httptest.NewRecorder()
			server.Router.ServeHTTP(w, req)
			require.Contains(t, []int{http.StatusOK, http.StatusCreated}, w.Code, w.Body.String())
			if output != nil {
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), output))
			}
		}
		check := func(key string, valid bool) {
			for _, ip := range []string{"203.0.113.42", "2001:db8::42"} {
				req := httptest.NewRequest("GET", "/check", nil)
				req.RemoteAddr = "127.0.0.1:1234"
				req.Header.Set("X-License-Key", key)
				req.Header.Set("X-Forwarded-For", ip)
				w := httptest.NewRecorder()
				server.Router.ServeHTTP(w, req)
				require.Equal(t, http.StatusOK, w.Code)
				var response map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.Equal(t, valid, response["valid"], w.Body.String())
			}
		}
		var product models.Product
		requestJSON("PUT", "/admin/products/"+productID, `{"ip_restrictions_disabled":true}`, "", &product)
		require.True(t, product.IPRestrictionsDisabled)
		check(licenseKey, true) // Existing license picks up the product policy.
		requestJSON("PUT", "/admin/products/"+productID, `{"name":"HA Product"}`, "", &product)
		require.True(t, product.IPRestrictionsDisabled) // Omitted leaves the policy alone.
		requestJSON("GET", "/admin/products/"+productID+"?include=group", "", "", &product)
		require.True(t, product.IPRestrictionsDisabled)

		var created models.License
		body, err := json.Marshal(map[string]interface{}{
			"product_id": productID, "type": "perpetual", "ip_restrictions_disabled": true,
			"allowed_ips": []string{"10.0.0.1"}, "auto_allowed_ip": true, "auto_allowed_ip_limit": 5,
		})
		require.NoError(t, err)
		requestJSON("POST", "/admin/keys", string(body), "", &created)
		require.True(t, created.IPRestrictionsDisabled)
		check(created.Key, true)
		requestJSON("PUT", "/admin/products/"+productID, `{"ip_restrictions_disabled":false}`, "", &product)
		require.False(t, product.IPRestrictionsDisabled)
		check(licenseKey, false) // Restores the original IP restrictions.
		check(created.Key, true) // The individual override survives the product change.

		var license models.License
		requestJSON("PUT", "/admin/keys", `{"ip_restrictions_disabled":true}`, licenseKey, &license)
		require.True(t, license.IPRestrictionsDisabled)
		check(licenseKey, true)
		requestJSON("GET", "/admin/keys", "", licenseKey, &license)
		require.True(t, license.IPRestrictionsDisabled)
		require.Equal(t, []string{"10.0.0.1/32", "2001:db8::1/128"}, license.AllowedIPs)
		require.Equal(t, []string{"192.168.1.0/24"}, license.AllowedNetworks)
		requestJSON("PUT", "/admin/keys", `{"ip_restrictions_disabled":false}`, licenseKey, &license)
		require.False(t, license.IPRestrictionsDisabled)
		check(licenseKey, false)

		listed, _, err := ls.ListLicenses(ctx, nil, models.PaginationParams{Page: 1, Limit: 100})
		require.NoError(t, err)
		for _, item := range listed {
			if item.Key == created.Key {
				require.True(t, item.IPRestrictionsDisabled)
				require.Equal(t, []string{"10.0.0.1/32"}, item.AllowedIPs) // No auto-registration in HA mode.
			}
		}
		products, _, err := ps.ListProducts(ctx, nil, models.PaginationParams{Page: 1, Limit: 100})
		require.NoError(t, err)
		require.False(t, products[0].IPRestrictionsDisabled)
	})
}
