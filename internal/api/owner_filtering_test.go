package api

import (
	"context"
	"testing"
	"time"
	"path/filepath"

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

func TestOwnerFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	dbName := "clortho_test_filter"
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
		DatabaseURL: connStr,
	}

	// Run migrations
	absPath, _ := filepath.Abs("../../migrations")
	err = database.Migrate(cfg.DatabaseURL, absPath)
	require.NoError(t, err)

	pool, err := database.New(ctx, cfg.DatabaseURL)
	require.NoError(t, err)
	defer pool.Close()

	ps := store.NewPostgresProductStore(pool)

	owner1 := uuid.New().String()
	owner2 := uuid.New().String()

	p1 := &models.Product{
		ID:        uuid.New(),
		OwnerID:   &owner1,
		Name:      "Product 1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	p2 := &models.Product{
		ID:        uuid.New(),
		OwnerID:   &owner2,
		Name:      "Product 2",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	p3 := &models.Product{
		ID:        uuid.New(),
		OwnerID:   nil, // No owner
		Name:      "Product 3",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, ps.CreateProduct(ctx, p1))
	require.NoError(t, ps.CreateProduct(ctx, p2))
	require.NoError(t, ps.CreateProduct(ctx, p3))

	t.Run("Filter by Owner 1", func(t *testing.T) {
		products, _, err := ps.ListProducts(ctx, &owner1, models.PaginationParams{})
		require.NoError(t, err)
		assert.Len(t, products, 1)
		assert.Equal(t, p1.ID, products[0].ID)
	})

	t.Run("Filter by Owner 2", func(t *testing.T) {
		products, _, err := ps.ListProducts(ctx, &owner2, models.PaginationParams{})
		require.NoError(t, err)
		assert.Len(t, products, 1)
		assert.Equal(t, p2.ID, products[0].ID)
	})

	t.Run("No Filter (List All)", func(t *testing.T) {
		products, _, err := ps.ListProducts(ctx, nil, models.PaginationParams{})
		require.NoError(t, err)
		assert.Len(t, products, 3)
	})
	
	// Test License Filtering
	ls := store.NewPostgresLicenseStore(pool)
	
	l1 := &models.License{
		ID:        uuid.New(),
		Key:       "KEY-1234",
		OwnerID:   &owner1,
		Type:      models.LicenseTypePerpetual,
		ProductID: p1.ID,
		Status:    models.LicenseStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	l2 := &models.License{
		ID:        uuid.New(),
		Key:       "KEY-5678",
		OwnerID:   &owner2,
		Type:      models.LicenseTypePerpetual,
		ProductID: p2.ID,
		Status:    models.LicenseStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, ls.CreateLicense(ctx, l1))
	require.NoError(t, ls.CreateLicense(ctx, l2))

	t.Run("Filter Licenses by Owner 1", func(t *testing.T) {
		licenses, _, err := ls.ListLicenses(ctx, &owner1, models.PaginationParams{})
		require.NoError(t, err)
		assert.Len(t, licenses, 1)
		assert.Equal(t, l1.ID, licenses[0].ID)
	})
	
	t.Run("Filter Licenses by Owner 2", func(t *testing.T) {
		licenses, _, err := ls.ListLicenses(ctx, &owner2, models.PaginationParams{})
		require.NoError(t, err)
		assert.Len(t, licenses, 1)
		assert.Equal(t, l2.ID, licenses[0].ID)
	})
}
