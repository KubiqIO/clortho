package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clortho/internal/config"
	"clortho/internal/database"
	"clortho/internal/store"
	"crypto/ed25519"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestSeparateRateLimits(t *testing.T) {
	ctx := context.Background()

	dbName := "clortho_test_rl"
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

	pool, err := database.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()
	
	// Generate keys
	_, privKey, _ := ed25519.GenerateKey(nil)
	privKeyBase64 := base64.StdEncoding.EncodeToString(privKey)

	cfg := config.Config{
		DatabaseURL:               connStr,
		AdminSecret:               "test-secret",
		ResponseSigningPrivateKey: privKeyBase64,
		RateLimitAdmin: config.RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 1,
			Burst:             1,
		},
		RateLimitCheck: config.RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 10,
			Burst:             10,
		},
	}

	// Initialize Server with real or stub stores. Using real ones is easier if DB is up.
	ls := store.NewPostgresLicenseStore(pool)
	ps := store.NewPostgresProductStore(pool)
	pgs := store.NewPostgresProductGroupStore(pool)
	rs := store.NewPostgresReleaseStore(pool)
	fs := store.NewPostgresFeatureStore(pool)
	logs := store.NewPostgresLogStore(pool)
	statsStore := store.NewPostgresStatsStore(pool)
	
	server := NewServer(cfg, pool, ls, ps, pgs, rs, fs, logs, statsStore)

	// Test 1: Admin Rate Limit Exhaustion
	t.Run("Admin Rate Limit Exhaustion", func(t *testing.T) {
		// Generate JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-admin",
			"iss": "clortho-test",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString([]byte(cfg.AdminSecret))
		authHeader := "Bearer " + tokenString

		// Request 1: Should pass (Burst 1)
		req, _ := http.NewRequest("GET", "/admin/products", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		// We might get DB error or 200, but NOT 429
		require.NotEqual(t, http.StatusTooManyRequests, w.Code, "First request should not be rate limited")

		// Request 2: Should fail (Limit 1/s, Burst 1 consumed)
		req2, _ := http.NewRequest("GET", "/admin/products", nil)
		req2.Header.Set("Authorization", authHeader)
		w2 := httptest.NewRecorder()
		server.Router.ServeHTTP(w2, req2)
		require.Equal(t, http.StatusTooManyRequests, w2.Code, "Second request should be rate limited")
	})

	// Test 2: Check API should NOT be affected by Admin exhaustion
	t.Run("Check API Independence", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("GET", "/check", nil)
			w := httptest.NewRecorder()
			server.Router.ServeHTTP(w, req)
			require.NotEqual(t, http.StatusTooManyRequests, w.Code, "Check API should still allow requests")
		}
	})
	
	// Test 3: Check API Exhaustion
	t.Run("Check API Exhaustion", func(t *testing.T) {
		// config: rate 10, burst 10.
		// We already consumed 5 in previous step.
		// Consume remaining 5
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("GET", "/check", nil)
			w := httptest.NewRecorder()
			server.Router.ServeHTTP(w, req)
			require.NotEqual(t, http.StatusTooManyRequests, w.Code)
		}
		
		// Next one should fail
		req, _ := http.NewRequest("GET", "/check", nil)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusTooManyRequests, w.Code)
	})
}
