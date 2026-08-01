package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
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
)

// TestCustomKeyImport covers POST /admin/keys accepting a caller-supplied key.
func TestCustomKeyImport(t *testing.T) {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("clortho_test_custom_key"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, postgresContainer.Terminate(ctx))
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	cfg := config.Config{
		DatabaseURL:               connStr,
		AdminSecret:               "test-secret",
		ResponseSigningPrivateKey: base64.StdEncoding.EncodeToString(privKey),
		RateLimitAdmin:            config.RateLimitConfig{Enabled: false},
		RateLimitCheck:            config.RateLimitConfig{Enabled: false},
		TrustedProxies:            []string{"127.0.0.1"},
	}

	absPath, _ := filepath.Abs("../../migrations")
	require.NoError(t, database.Migrate(cfg.DatabaseURL, absPath))

	pool, err := database.New(ctx, cfg.DatabaseURL)
	require.NoError(t, err)
	defer pool.Close()

	server := NewServer(cfg, pool,
		store.NewPostgresLicenseStore(pool),
		store.NewPostgresProductStore(pool),
		store.NewPostgresProductGroupStore(pool),
		store.NewPostgresReleaseStore(pool),
		store.NewPostgresFeatureStore(pool),
		store.NewPostgresLogStore(pool),
		store.NewPostgresStatsStore(pool),
	)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "test-admin",
		"iss": "clortho-test",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(cfg.AdminSecret))
	require.NoError(t, err)
	authHeader := "Bearer " + tokenString

	post := func(t *testing.T, path string, body map[string]interface{}) *httptest.ResponseRecorder {
		t.Helper()
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		req, _ := http.NewRequest("POST", path, bytes.NewBuffer(encoded))
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		return w
	}

	// A product with a prefix of its own, so a successful import is visibly the
	// caller's key rather than one the product configuration would have produced.
	var productID string
	{
		w := post(t, "/admin/products", map[string]interface{}{
			"name":           "Custom Key Test Product",
			"license_prefix": "GENERATED",
		})
		require.Equal(t, http.StatusCreated, w.Code)
		var resp models.Product
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		productID = resp.ID.String()
	}

	const importedKey = "Owned-3f9a2c8e1b7d"

	t.Run("imports the supplied key verbatim", func(t *testing.T) {
		w := post(t, "/admin/keys", map[string]interface{}{
			"product_id": productID,
			"type":       "perpetual",
			"key":        importedKey,
		})
		require.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, importedKey, resp["key"],
			"the imported key must survive byte-identical; the customer already holds it")
	})

	t.Run("the imported key validates at /check", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/check", nil)
		req.Header.Set("X-License-Key", importedKey)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, true, resp["valid"],
			"an imported licence must check out under the key it was imported with")
	})

	t.Run("rejects a key that is already taken", func(t *testing.T) {
		w := post(t, "/admin/keys", map[string]interface{}{
			"product_id": productID,
			"type":       "perpetual",
			"key":        importedKey,
		})
		require.Equal(t, http.StatusConflict, w.Code,
			"a re-run of the importer must report the collision, not fail opaquely")
	})

	t.Run("rejects a key that would break the check header", func(t *testing.T) {
		for _, key := range []string{
			"Owned-3f9a 2c8e",
			"Owned-3f9a\r\nX-Injected: 1",
			"",
		} {
			w := post(t, "/admin/keys", map[string]interface{}{
				"product_id": productID,
				"type":       "perpetual",
				"key":        key,
			})
			if key == "" {
				// An absent key is the generate path, not an error.
				require.Equal(t, http.StatusCreated, w.Code)
				continue
			}
			require.Equal(t, http.StatusBadRequest, w.Code, "key %q should be rejected", key)
		}
	})

	t.Run("rejects key together with prefix or length", func(t *testing.T) {
		w := post(t, "/admin/keys", map[string]interface{}{
			"product_id": productID,
			"type":       "perpetual",
			"key":        "Owned-aaaaaaaaaaaa",
			"prefix":     "SOMETHING",
		})
		require.Equal(t, http.StatusBadRequest, w.Code)

		w = post(t, "/admin/keys", map[string]interface{}{
			"product_id": productID,
			"type":       "perpetual",
			"key":        "Owned-bbbbbbbbbbbb",
			"length":     20,
		})
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("still generates a key when none is supplied", func(t *testing.T) {
		w := post(t, "/admin/keys", map[string]interface{}{
			"product_id": productID,
			"type":       "perpetual",
		})
		require.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		key, ok := resp["key"].(string)
		require.True(t, ok)
		require.Contains(t, key, "GENERATED", "the product prefix should still apply")
		require.NotEqual(t, importedKey, key)
	})
}
