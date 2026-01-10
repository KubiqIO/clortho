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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"clortho/internal/config"
	"clortho/internal/database"
	"clortho/internal/models"
	"clortho/internal/store"
)

func TestLogFiltering(t *testing.T) {
	ctx := context.Background()

	dbName := "clortho_test_logs"
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

	cfg := config.Config{
		DatabaseURL:               connStr,
		AdminSecret:                 "test-secret",
		RateLimitAdmin: config.RateLimitConfig{
			Enabled: false,
		},
		RateLimitCheck: config.RateLimitConfig{
			Enabled: false,
		},
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
	logStore := store.NewPostgresLogStore(pool)
	statsStore := store.NewPostgresStatsStore(pool)
	server := NewServer(cfg, pool, ls, ps, pgs, rs, fs, logStore, statsStore)

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
			"name":           "Log Test Product",
			"owner_id":       "test-admin",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/admin/products", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		
		var resp models.PaginatedList[models.Product]
		reqList, _ := http.NewRequest("GET", "/admin/products", nil)
		reqList.Header.Set("Authorization", authHeader)
		wList := httptest.NewRecorder()
		server.Router.ServeHTTP(wList, reqList)
		json.Unmarshal(wList.Body.Bytes(), &resp)
		for _, p := range resp.Items {
			if p.Name == "Log Test Product" {
				productID = p.ID.String()
			}
		}
	}

	// 2. Create License
	licenseKey := ""
	licenseIDStr := ""
	{
		reqBody := map[string]interface{}{
			"product_id": productID,
			"type":       "perpetual",
			"owner_id":   "test-admin",
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
		
		// Extract ID from response
		idStr, ok := resp["id"].(string)
		if !ok {
			// If not in response, try to fetch by key
			l, err := ls.GetLicenseByKey(ctx, licenseKey)
			require.NoError(t, err)
			idStr = l.ID.String()
		}
		licenseIDStr = idStr
	}

	// 3. Generate Logs
	// Manually insert logs into the store to simulate different status codes
	prodUUID, _ := uuid.Parse(productID)
	licUUID, _ := uuid.Parse(licenseIDStr)
	licenseID := licUUID

	// Log 1: Success 200
	log1 := &models.LicenseCheckLog{
		ProductID:      &prodUUID,
		LicenseID:      &licenseID,
		LicenseKey:     licenseKey,
		RequestPayload: map[string]interface{}{"foo": "bar"},
		ResponsePayload: map[string]interface{}{"valid": true},
		IPAddress:      "1.2.3.4",
		UserAgent:      "test-agent",
		StatusCode:     200,
	}
	err = logStore.CreateLicenseCheckLog(ctx, log1)
	require.NoError(t, err)

	// Log 2: Not Found 404
	log2 := &models.LicenseCheckLog{
		ProductID:      &prodUUID,
		LicenseID:      &licenseID,
		LicenseKey:     licenseKey,
		RequestPayload: map[string]interface{}{"foo": "baz"},
		ResponsePayload: map[string]interface{}{"error": "not found"},
		IPAddress:      "1.2.3.4",
		UserAgent:      "test-agent",
		StatusCode:     404,
	}
	err = logStore.CreateLicenseCheckLog(ctx, log2)
	require.NoError(t, err)

	// Log 3: Server Error 500
	log3 := &models.LicenseCheckLog{
		ProductID:      &prodUUID,
		LicenseID:      &licenseID,
		LicenseKey:     licenseKey,
		RequestPayload: map[string]interface{}{"foo": "qux"},
		ResponsePayload: map[string]interface{}{"error": "server error"},
		IPAddress:      "1.2.3.4",
		UserAgent:      "test-agent",
		StatusCode:     500,
	}
	err = logStore.CreateLicenseCheckLog(ctx, log3)
	require.NoError(t, err)


	// 4. Verify Filtering
	// Case 1: Filter by status 200
	{
		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?license_key="+licenseKey+"&status_code=200", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp models.PaginatedList[models.LicenseCheckLog]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, 200, resp.Items[0].StatusCode)
	}

	// Case 2: Filter by status 404
	{
		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?license_key="+licenseKey+"&status_code=404", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp models.PaginatedList[models.LicenseCheckLog]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, 404, resp.Items[0].StatusCode)
	}

	// Case 3: Filter by status 500
	{
		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?license_key="+licenseKey+"&status_code=500", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp models.PaginatedList[models.LicenseCheckLog]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, 500, resp.Items[0].StatusCode)
	}

	// Case 4: No Filter (should get all 3)
	{
		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?license_key="+licenseKey, nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp models.PaginatedList[models.LicenseCheckLog]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 3)
	}
	
	// Case 5: Filter by Product ID and Status
	{
		req, _ := http.NewRequest("GET", "/admin/logs/license-checks?product_id="+productID+"&status_code=500", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp models.PaginatedList[models.LicenseCheckLog]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, 500, resp.Items[0].StatusCode)
	}
}
