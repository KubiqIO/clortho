package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"clortho/internal/config"
	"clortho/internal/database"
	"clortho/internal/models"
	"clortho/internal/store"
)

func TestLicenseLifecycle(t *testing.T) {
	ctx := context.Background()

	dbName := "clortho_test"
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
	// Generate a key pair for testing
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	privKeyBase64 := base64.StdEncoding.EncodeToString(privKey)

	cfg := config.Config{
		DatabaseURL:               connStr,
		AdminSecret:                 "test-secret",
		ResponseSigningPrivateKey: privKeyBase64,
		RateLimitAdmin: config.RateLimitConfig{
			Enabled: false,
		},
		RateLimitCheck: config.RateLimitConfig{
			Enabled: false,
		},
		TrustedProxies: []string{"127.0.0.1"},
	}

	// Run migrations to ensure schema is fresh
	// Use absolute path or relative path to migrations
	// Assuming test is running from internal/api, so ../../migrations
	absPath, _ := filepath.Abs("../../migrations")
	err = database.Migrate(cfg.DatabaseURL, absPath)
	require.NoError(t, err)

	pool, err := database.New(ctx, cfg.DatabaseURL)
	require.NoError(t, err)
	defer pool.Close()

	// Initialize Stores
	ls := store.NewPostgresLicenseStore(pool)
	ps := store.NewPostgresProductStore(pool)
	pgs := store.NewPostgresProductGroupStore(pool)
	rs := store.NewPostgresReleaseStore(pool)
	fs := store.NewPostgresFeatureStore(pool)
	logs := store.NewPostgresLogStore(pool)
	statsStore := store.NewPostgresStatsStore(pool)
	// Initialize Server
	server := NewServer(cfg, pool, ls, ps, pgs, rs, fs, logs, statsStore)

	// Generate JWT for auth
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "test-admin",
		"iss": "clortho-test",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(cfg.AdminSecret))
	require.NoError(t, err)
	authHeader := "Bearer " + tokenString

	// Step 1: Create Product
	t.Log("Step 1: Create Product")
	productID := ""
	{
		reqBody := map[string]interface{}{
			"name":           "Integration Test Product",
			"description":    "Product for integration testing",
			"license_prefix": "TEST",
			"owner_id":       "test-admin",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/admin/products", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
	}
	
	// List products to get the ID
	{
		req, _ := http.NewRequest("GET", "/admin/products", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		
		var resp models.PaginatedList[models.Product]
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Items)
		for _, p := range resp.Items {
			if p.Name == "Integration Test Product" {
				productID = p.ID.String()
				break
			}
		}
		require.NotEmpty(t, productID, "Product ID not found")
	}

	// Step 1b: Create Feature & Release
	t.Log("Step 1b: Create Feature & Release")
	{
		// Create Feature
		reqBody := map[string]interface{}{
			"name":       "Integration Feature",
			"code":       "INT-FEAT",
			"owner_id":   "test-admin",
			"product_id": productID,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/admin/features", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Create Release
		reqBodyRel := map[string]interface{}{
			"version":    "1.0.0",
			"owner_id":   "test-admin",
			"product_id": productID,
		}
		bodyRel, _ := json.Marshal(reqBodyRel)
		reqRel, _ := http.NewRequest("POST", "/admin/releases", bytes.NewBuffer(bodyRel))
		reqRel.Header.Set("Authorization", authHeader)
		wRel := httptest.NewRecorder()
		server.Router.ServeHTTP(wRel, reqRel)
		require.Equal(t, http.StatusCreated, wRel.Code)
	}

	// Step 2: Create License
	t.Log("Step 2: Create License")
	var licenseKey string
	{
		reqBody := map[string]interface{}{
			"product_id":    productID,
			"type":          "perpetual",
			"feature_codes": []string{"INT-FEAT"},
			"release_versions": []string{"1.0.0"},
			"owner_id":         "test-admin",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/admin/keys", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		licenseKey = resp["key"].(string)
		require.NotEmpty(t, licenseKey)
		assert.Contains(t, licenseKey, "TEST") // Prefix we set
		
		features := resp["features"].([]interface{})
		require.Len(t, features, 1)
		assert.Equal(t, "INT-FEAT", features[0])

		releases := resp["releases"].([]interface{})
		require.Len(t, releases, 1)
		assert.Equal(t, "1.0.0", releases[0])
	}

	// Step 3: Check License (Valid)
	t.Log("Step 3: Check License (Valid)")
	{
		// Capture logs to verify IP
		var buf bytes.Buffer
		h := slog.NewJSONHandler(&buf, nil)
		logger := slog.New(h)
		slog.SetDefault(logger)
		// Do not defer here, as it runs at function exit
		
		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", licenseKey)
		// Set Client IP
		clientIP := "10.0.0.1"
		req.Header.Set("X-Forwarded-For", clientIP)
		req.RemoteAddr = "127.0.0.1:1234" // Required for Gin to check trusted proxies
		
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		
		valid, ok := resp["valid"].(bool)
		require.True(t, ok)
		assert.True(t, valid, "License should be valid")

		// Verify token (JWT)
		tokenStr, hasToken := resp["token"].(string)
		// Should have token as we have a key configured
		assert.True(t, hasToken, "Response SHOULD contain token by default if key is configured")

		// Verification of token
		parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return pubKey, nil
		})
		require.NoError(t, err)
		assert.True(t, parsedToken.Valid, "Token should be valid")

		// Verify claims
		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		require.True(t, ok)
		assert.Equal(t, licenseKey, claims["sub"])
		assert.Equal(t, valid, claims["valid"])
		assert.Equal(t, "clortho", claims["iss"])

		// Verify features claim
		featuresClaim, ok := claims["features"].([]interface{})
		require.True(t, ok, "Token should contain features claim")
		require.Len(t, featuresClaim, 1)
		assert.Equal(t, "INT-FEAT", featuresClaim[0])
	}

	// Step 4: Revoke License
	t.Log("Step 4: Revoke License")
	{
		req, _ := http.NewRequest("DELETE", "/admin/keys", nil)
		req.Header.Set("X-License-Key", licenseKey)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	}

	// Step 5: Check License (Revoked)
	t.Log("Step 5: Check License (Revoked)")
	{
		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", licenseKey)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Revoked license should return 200 OK with valid: false")
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		
		valid, ok := resp["valid"].(bool)
		require.True(t, ok)
		assert.False(t, valid, "License should be invalid")
		assert.Equal(t, "License is revoked", resp["reason"])
	}

	// Step 5b: Purge License
	t.Log("Step 5b: Purge License")
	{
		req, _ := http.NewRequest("DELETE", "/admin/keys/purge", nil)
		req.Header.Set("X-License-Key", licenseKey)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	}

	// Step 5c: Check License (Purged)
	t.Log("Step 5c: Check License (Purged)")
	{
		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", licenseKey)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code, "Purged license should return 404")
	}

	// Step 6: Verify Logs
	t.Log("Step 6: Verify Logs")
	{
		// Verify License Check Logs
		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?license_key="+licenseKey, nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp models.PaginatedList[models.LicenseCheckLog]
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Items, "Should have license check logs")
		// We expect at least 2 logs (valid check and revoked check)
		assert.GreaterOrEqual(t, len(resp.Items), 2)
		
		// Verify Admin Logs
		reqAdmin, _ := http.NewRequest("GET", "/admin/logs/admin-actions?actor=test-admin", nil)
		reqAdmin.Header.Set("Authorization", authHeader)
		wAdmin := httptest.NewRecorder()
		server.Router.ServeHTTP(wAdmin, reqAdmin)

		require.Equal(t, http.StatusOK, wAdmin.Code)
		require.Equal(t, http.StatusOK, wAdmin.Code)
		var adminResp models.PaginatedList[models.AdminLog]
		err = json.Unmarshal(wAdmin.Body.Bytes(), &adminResp)
		require.NoError(t, err)
		assert.NotEmpty(t, adminResp.Items, "Should have admin logs")
		// We expect multiple admin actions: Create Product, Create Feature, Create Release, Create License, Revoke License
		assert.GreaterOrEqual(t, len(adminResp.Items), 1)
	}
}
