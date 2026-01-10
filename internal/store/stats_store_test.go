package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/google/uuid"
	"path/filepath"

	"clortho/internal/database"
	"clortho/internal/models"
)

func TestGetDashboardStats(t *testing.T) {
	ctx := context.Background()

	dbName := "clortho_test_stats"
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

	// Run migrations
	absPath, _ := filepath.Abs("../../migrations")
	err = database.Migrate(connStr, absPath)
	require.NoError(t, err)

	pool, err := database.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()
	
	statsStore := NewPostgresStatsStore(pool)
	productStore := NewPostgresProductStore(pool)
	licenseStore := NewPostgresLicenseStore(pool)
	logStore := NewPostgresLogStore(pool)

	// Setup Data
	owner1 := "owner1"
	owner2 := "owner2"

	// Create Products
	// P1: New (Default CreatedAt is 0, so needs setting or assumed Now)
	p1 := &models.Product{ID: uuid.New(), OwnerID: &owner1, Name: "P1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, productStore.CreateProduct(ctx, p1))
	
	// P2: Old (>30d ago)
	p2 := &models.Product{ID: uuid.New(), OwnerID: &owner2, Name: "P2", CreatedAt: time.Now().Add(-40 * 24 * time.Hour), UpdatedAt: time.Now()}
	require.NoError(t, productStore.CreateProduct(ctx, p2))

	// Create Licenses
	// L1: New
	l1 := &models.License{
        ID: uuid.New(), Key: "KEY1", OwnerID: &owner1, ProductID: p1.ID, Type: models.LicenseTypePerpetual, Status: models.LicenseStatusActive,
        CreatedAt: time.Now(), UpdatedAt: time.Now(),
    }
	require.NoError(t, licenseStore.CreateLicense(ctx, l1))

	// L2: Old (>30d ago)
	l2 := &models.License{
        ID: uuid.New(), Key: "KEY2", OwnerID: &owner2, ProductID: p2.ID, Type: models.LicenseTypePerpetual, Status: models.LicenseStatusActive,
        CreatedAt: time.Now().Add(-40 * 24 * time.Hour), UpdatedAt: time.Now(),
    }
	require.NoError(t, licenseStore.CreateLicense(ctx, l2))
	
	// Create Logs
	now := time.Now()
    
    // Check 1: Recent (Last 1 hour)
	chk1 := &models.LicenseCheckLog{ProductID: &p1.ID, LicenseKey: "KEY1", StatusCode: 200}
	require.NoError(t, logStore.CreateLicenseCheckLog(ctx, chk1)) // CreatedAt = Now

    // Check 2: Prev 24h Window (30 hours ago)
	chk2 := &models.LicenseCheckLog{ProductID: &p1.ID, LicenseKey: "KEY1", StatusCode: 200}
	require.NoError(t, logStore.CreateLicenseCheckLog(ctx, chk2))
	_, err = pool.Exec(ctx, "UPDATE license_check_logs SET created_at = $1 WHERE id = $2", now.Add(-30*time.Hour), chk2.ID)
	require.NoError(t, err)

    // Check 3: Old (> 48h ago)
	chk3 := &models.LicenseCheckLog{ProductID: &p2.ID, LicenseKey: "KEY2", StatusCode: 200}
	require.NoError(t, logStore.CreateLicenseCheckLog(ctx, chk3))
	_, err = pool.Exec(ctx, "UPDATE license_check_logs SET created_at = $1 WHERE id = $2", now.Add(-60*time.Hour), chk3.ID)
	require.NoError(t, err)

	// Create some error logs for checking stats
	// Error 1: Recent (Last 1 hour) for Owner 1
	err1 := &models.LicenseCheckLog{ProductID: &p1.ID, LicenseKey: "KEY1", StatusCode: 500}
	require.NoError(t, logStore.CreateLicenseCheckLog(ctx, err1))

	// Error 2: Previous 24h (30 hours ago) for Owner 1. 
	// This one has status 200 but has a "reason" in payload, should count as error.
	err2 := &models.LicenseCheckLog{
		ProductID: &p1.ID, 
		LicenseKey: "KEY1", 
		StatusCode: 200,
		ResponsePayload: map[string]interface{}{"reason": "expired"},
	}
	require.NoError(t, logStore.CreateLicenseCheckLog(ctx, err2))
	_, err = pool.Exec(ctx, "UPDATE license_check_logs SET created_at = $1 WHERE id = $2", now.Add(-30*time.Hour), err2.ID)
	require.NoError(t, err)

	// Additional success check to separate totals
	// Success check for Owner 1 (Recent)
	chk4 := &models.LicenseCheckLog{ProductID: &p1.ID, LicenseKey: "KEY1", StatusCode: 200}
	require.NoError(t, logStore.CreateLicenseCheckLog(ctx, chk4))

	// Admin Logs
	// Create 3 logs to verify Recent Logs list
	log1 := &models.AdminLog{Action: "create_prod_1", EntityType: "product", OwnerID: &owner1}
	require.NoError(t, logStore.CreateAdminLog(ctx, log1))
	
	log2 := &models.AdminLog{Action: "create_prod_2", EntityType: "product", OwnerID: &owner2}
	require.NoError(t, logStore.CreateAdminLog(ctx, log2))

	log3 := &models.AdminLog{Action: "create_prod_3", EntityType: "product", OwnerID: &owner1}
	require.NoError(t, logStore.CreateAdminLog(ctx, log3))
	// Update log3 to be old to check sorting order (though creating them in sequence usually implies time order, updating makes it explicit)
	_, err = pool.Exec(ctx, "UPDATE admin_logs SET created_at = $1 WHERE id = $2", now.Add(-10*time.Hour), log3.ID)
	require.NoError(t, err)

    // And one more recent one
    require.NoError(t, logStore.CreateAdminLog(ctx, &models.AdminLog{Action: "act", EntityType: "prod", OwnerID: &owner1}))


	// Test Cases

	// 1. All Stats (No Filter)
	// Products: 2 (1 new, 1 old). Change: 1.
	// Licenses: 2 (1 new, 1 old). Change: 1.
	// Checks: 6 (3 success from before + 2 errors + 1 new success check).
	//   Original Checks: chk1(P1, New), chk2(P1, Prev24), chk3(P2, Old)
	//   New Checks: err1(P1, New, 500), err2(P1, Prev24, 404), chk4(P1, New, 200)
	//   Total Checks: 6.
	//
	// Errors:
	//   Total: 2 (err1, err2)
	//   Last 24h: 1 (err1)
	//   Prev 24h: 1 (err2)
	//   Change: 1 - 1 = 0
	stats, err := statsStore.GetDashboardStats(ctx, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalProducts)
	assert.Equal(t, 1, stats.TotalProductsChange)
	assert.Equal(t, 2, stats.TotalLicenses)
	assert.Equal(t, 1, stats.TotalLicensesChange)
	assert.Equal(t, 6, stats.TotalLicenseChecks) 
	// Checks Last 24h: chk1, err1, chk4 = 3
	// Checks Prev 24h: chk2, err2 = 2
	// Change: 3 - 2 = 1
	assert.Equal(t, 1, stats.TotalLicenseChecksChange)
	
	// Verify Error Stats
	assert.Equal(t, 2, stats.TotalLicenseCheckErrors)
	assert.Equal(t, 0, stats.TotalLicenseCheckErrorsChange) // 1 (recent) - 1 (prev) = 0
	assert.Equal(t, 4, stats.TotalAdminActions)
	assert.Len(t, stats.RecentAdminLogs, 3) // We created 4 admin logs, but it limits to 3
	// Check order (descending by created at)
	// owner1/owner2 yesterday are newer than owner1-old
	// Note: owner1-yest and owner2-yest have same timestamp (yesterday). Order is non-deterministic between them unless ID sorts or insertion order. 
	// But they should be first 2. 
	
	// 2. Filter by Owner1
	// Owner 1 has P1.
	// Checks for P1:
	//   chk1 (New, 200)
	//   chk2 (Prev24, 200)
	//   err1 (New, 500)
	//   err2 (Prev24, 404)
	//   chk4 (New, 200)
	// Total Checks: 5
	// Checks Last 24h: chk1, err1, chk4 = 3
	// Checks Prev 24h: chk2, err2 = 2
	// Change: 3 - 2 = 1
	//
	// Errors:
	//   Total: 2 (err1, err2)
	//   Last 24h: 1 (err1)
	//   Prev 24h: 1 (err2)
	//   Change: 0
	statsOwner1, err := statsStore.GetDashboardStats(ctx, &owner1, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, statsOwner1.TotalProducts)
	assert.Equal(t, 1, statsOwner1.TotalProductsChange)
	assert.Equal(t, 1, statsOwner1.TotalLicenses)
	assert.Equal(t, 1, statsOwner1.TotalLicensesChange)
	assert.Equal(t, 5, statsOwner1.TotalLicenseChecks)
	assert.Equal(t, 1, statsOwner1.TotalLicenseChecksChange)
	assert.Equal(t, 2, statsOwner1.TotalLicenseCheckErrors)
	assert.Equal(t, 0, statsOwner1.TotalLicenseCheckErrorsChange)
	assert.Equal(t, 3, statsOwner1.TotalAdminActions)

}
