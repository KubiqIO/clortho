package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/models"
)

type StatsStore interface {
	GetDashboardStats(ctx context.Context, ownerID *string, since *time.Time) (*models.DashboardStats, error)
}

type PostgresStatsStore struct {
	DB *pgxpool.Pool
}

func NewPostgresStatsStore(db *pgxpool.Pool) *PostgresStatsStore {
	return &PostgresStatsStore{DB: db}
}

func (s *PostgresStatsStore) GetDashboardStats(ctx context.Context, ownerID *string, since *time.Time) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	// 1. Total Products
	productQuery := `SELECT count(*) FROM products`
	productArgs := []interface{}{}
	if ownerID != nil {
		productQuery += ` WHERE owner_id = $1`
		productArgs = append(productArgs, ownerID)
	}
	if err := s.DB.QueryRow(ctx, productQuery, productArgs...).Scan(&stats.TotalProducts); err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	// 2. Total Licenses
	licenseQuery := `SELECT count(*) FROM licenses`
	licenseArgs := []interface{}{}
	if ownerID != nil {
		licenseQuery += ` WHERE owner_id = $1`
		licenseArgs = append(licenseArgs, ownerID)
	}
	if err := s.DB.QueryRow(ctx, licenseQuery, licenseArgs...).Scan(&stats.TotalLicenses); err != nil {
		return nil, fmt.Errorf("failed to count licenses: %w", err)
	}

	// 3. Total License Checks
	checkQuery := `
		SELECT count(*) 
		FROM license_check_logs lcl
	`
	checkArgs := []interface{}{}
	whereClauses := []string{}
	argIdx := 1

	if ownerID != nil {
		checkQuery += ` JOIN products p ON lcl.product_id = p.id`
		whereClauses = append(whereClauses, fmt.Sprintf("p.owner_id = $%d", argIdx))
		checkArgs = append(checkArgs, ownerID)
		argIdx++
	}

	if since != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("lcl.created_at >= $%d", argIdx))
		checkArgs = append(checkArgs, since)
		argIdx++
	}

	if len(whereClauses) > 0 {
		checkQuery += " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			checkQuery += " AND " + whereClauses[i]
		}
	}

	if err := s.DB.QueryRow(ctx, checkQuery, checkArgs...).Scan(&stats.TotalLicenseChecks); err != nil {
		return nil, fmt.Errorf("failed to count license checks: %w", err)
	}

	// 4. Total Admin Actions
	adminQuery := `SELECT count(*) FROM admin_logs`
	adminArgs := []interface{}{}
	adminWhere := []string{}
	adminArgIdx := 1

	if ownerID != nil {
		adminWhere = append(adminWhere, fmt.Sprintf("owner_id = $%d", adminArgIdx))
		adminArgs = append(adminArgs, ownerID)
		adminArgIdx++
	}

	if since != nil {
		adminWhere = append(adminWhere, fmt.Sprintf("created_at >= $%d", adminArgIdx))
		adminArgs = append(adminArgs, since)
		adminArgIdx++
	}

	if len(adminWhere) > 0 {
		adminQuery += " WHERE " + adminWhere[0]
		for i := 1; i < len(adminWhere); i++ {
			adminQuery += " AND " + adminWhere[i]
		}
	}

	if err := s.DB.QueryRow(ctx, adminQuery, adminArgs...).Scan(&stats.TotalAdminActions); err != nil {
		return nil, fmt.Errorf("failed to count admin actions: %w", err)
	}

	// 5. Change Metrics

	// 5a. TotalProductsChange (Change in last 30d)
	// Count products created in the last 30 days as a proxy for "Change".
	prodChangeQuery := `SELECT count(*) FROM products WHERE created_at >= NOW() - INTERVAL '30 days'`
	prodChangeArgs := []interface{}{}
	if ownerID != nil {
		prodChangeQuery += ` AND owner_id = $1`
		prodChangeArgs = append(prodChangeArgs, ownerID)
	}
	if err := s.DB.QueryRow(ctx, prodChangeQuery, prodChangeArgs...).Scan(&stats.TotalProductsChange); err != nil {
		return nil, fmt.Errorf("failed to count product change: %w", err)
	}

	// 5b. TotalLicensesChange (Change in last 30d)
	licChangeQuery := `SELECT count(*) FROM licenses WHERE created_at >= NOW() - INTERVAL '30 days'`
	licChangeArgs := []interface{}{}
	if ownerID != nil {
		licChangeQuery += ` AND owner_id = $1`
		licChangeArgs = append(licChangeArgs, ownerID)
	}
	if err := s.DB.QueryRow(ctx, licChangeQuery, licChangeArgs...).Scan(&stats.TotalLicensesChange); err != nil {
		return nil, fmt.Errorf("failed to count license change: %w", err)
	}

	// 5c. TotalLicenseChecksChange (Last 24h vs Previous 24h)
	// Checks Last 24h
	checks24hQuery := `
		SELECT count(*)
		FROM license_check_logs lcl
	`
	checks24hArgs := []interface{}{}
	checks24hWhere := []string{`lcl.created_at >= NOW() - INTERVAL '24 hours'`}
	argIdx = 1

	if ownerID != nil {
		checks24hQuery += ` JOIN products p ON lcl.product_id = p.id`
		checks24hWhere = append(checks24hWhere, fmt.Sprintf("p.owner_id = $%d", argIdx))
		checks24hArgs = append(checks24hArgs, ownerID)
		argIdx++
	}
	
	checks24hQuery += " WHERE " + checks24hWhere[0]
	for i := 1; i < len(checks24hWhere); i++ {
		checks24hQuery += " AND " + checks24hWhere[i]
	}

	var checks24h int
	if err := s.DB.QueryRow(ctx, checks24hQuery, checks24hArgs...).Scan(&checks24h); err != nil {
		return nil, fmt.Errorf("failed to count checks 24h: %w", err)
	}

	// Checks Previous 24h (48h to 24h ago)
	checksPrev24hQuery := `
		SELECT count(*)
		FROM license_check_logs lcl
	`
	checksPrev24hArgs := []interface{}{}
	checksPrev24hWhere := []string{
		`lcl.created_at >= NOW() - INTERVAL '48 hours'`,
		`lcl.created_at < NOW() - INTERVAL '24 hours'`,
	}
	argIdx = 1

	if ownerID != nil {
		checksPrev24hQuery += ` JOIN products p ON lcl.product_id = p.id`
		checksPrev24hWhere = append(checksPrev24hWhere, fmt.Sprintf("p.owner_id = $%d", argIdx))
		checksPrev24hArgs = append(checksPrev24hArgs, ownerID)
		argIdx++
	}

	checksPrev24hQuery += " WHERE " + checksPrev24hWhere[0]
	for i := 1; i < len(checksPrev24hWhere); i++ {
		checksPrev24hQuery += " AND " + checksPrev24hWhere[i]
	}

	var checksPrev24h int
	if err := s.DB.QueryRow(ctx, checksPrev24hQuery, checksPrev24hArgs...).Scan(&checksPrev24h); err != nil {
		return nil, fmt.Errorf("failed to count checks prev 24h: %w", err)
	}

	stats.TotalLicenseChecksChange = checks24h - checksPrev24h

	// 5d. TotalLicenseCheckErrors
	errorQuery := `
		SELECT count(*)
		FROM license_check_logs lcl
	`
	errorArgs := []interface{}{}
	errorWhere := []string{"(lcl.status_code != 200 OR lcl.response_payload ->> 'reason' IS NOT NULL)"}
	argIdx = 1

	if ownerID != nil {
		errorQuery += ` JOIN products p ON lcl.product_id = p.id`
		errorWhere = append(errorWhere, fmt.Sprintf("p.owner_id = $%d", argIdx))
		errorArgs = append(errorArgs, ownerID)
		argIdx++
	}

	if since != nil {
		errorWhere = append(errorWhere, fmt.Sprintf("lcl.created_at >= $%d", argIdx))
		errorArgs = append(errorArgs, since)
		argIdx++
	}

	errorQuery += " WHERE " + errorWhere[0]
	for i := 1; i < len(errorWhere); i++ {
		errorQuery += " AND " + errorWhere[i]
	}

	if err := s.DB.QueryRow(ctx, errorQuery, errorArgs...).Scan(&stats.TotalLicenseCheckErrors); err != nil {
		return nil, fmt.Errorf("failed to count license check errors: %w", err)
	}

	// 5e. TotalLicenseCheckErrorsChange (Last 24h vs Previous 24h)
	// Errors Last 24h
	errors24hQuery := `
		SELECT count(*)
		FROM license_check_logs lcl
	`
	errors24hArgs := []interface{}{}
	errors24hWhere := []string{
		`(lcl.status_code != 200 OR lcl.response_payload ->> 'reason' IS NOT NULL)`,
		`lcl.created_at >= NOW() - INTERVAL '24 hours'`,
	}
	argIdx = 1

	if ownerID != nil {
		errors24hQuery += ` JOIN products p ON lcl.product_id = p.id`
		errors24hWhere = append(errors24hWhere, fmt.Sprintf("p.owner_id = $%d", argIdx))
		errors24hArgs = append(errors24hArgs, ownerID)
		argIdx++
	}
	
	errors24hQuery += " WHERE " + errors24hWhere[0]
	for i := 1; i < len(errors24hWhere); i++ {
		errors24hQuery += " AND " + errors24hWhere[i]
	}

	var errors24h int
	if err := s.DB.QueryRow(ctx, errors24hQuery, errors24hArgs...).Scan(&errors24h); err != nil {
		return nil, fmt.Errorf("failed to count errors 24h: %w", err)
	}

	// Errors Previous 24h (48h to 24h ago)
	errorsPrev24hQuery := `
		SELECT count(*)
		FROM license_check_logs lcl
	`
	errorsPrev24hArgs := []interface{}{}
	errorsPrev24hWhere := []string{
		`(lcl.status_code != 200 OR lcl.response_payload ->> 'reason' IS NOT NULL)`,
		`lcl.created_at >= NOW() - INTERVAL '48 hours'`,
		`lcl.created_at < NOW() - INTERVAL '24 hours'`,
	}
	argIdx = 1

	if ownerID != nil {
		errorsPrev24hQuery += ` JOIN products p ON lcl.product_id = p.id`
		errorsPrev24hWhere = append(errorsPrev24hWhere, fmt.Sprintf("p.owner_id = $%d", argIdx))
		errorsPrev24hArgs = append(errorsPrev24hArgs, ownerID)
		argIdx++
	}

	errorsPrev24hQuery += " WHERE " + errorsPrev24hWhere[0]
	for i := 1; i < len(errorsPrev24hWhere); i++ {
		errorsPrev24hQuery += " AND " + errorsPrev24hWhere[i]
	}

	var errorsPrev24h int
	if err := s.DB.QueryRow(ctx, errorsPrev24hQuery, errorsPrev24hArgs...).Scan(&errorsPrev24h); err != nil {
		return nil, fmt.Errorf("failed to count errors prev 24h: %w", err)
	}

	stats.TotalLicenseCheckErrorsChange = errors24h - errorsPrev24h

	// 6. Recent Admin Logs (Last 3)
	recentLogsQuery := `
		SELECT id, action, entity_type, entity_id, owner_id, details, created_at
		FROM admin_logs
	`
	recentLogsArgs := []interface{}{}
	if ownerID != nil {
		recentLogsQuery += ` WHERE owner_id = $1`
		recentLogsArgs = append(recentLogsArgs, ownerID)
	}
	recentLogsQuery += ` ORDER BY created_at DESC LIMIT 3`

	rows, err := s.DB.Query(ctx, recentLogsQuery, recentLogsArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent admin logs: %w", err)
	}
	defer rows.Close()

	var recentLogs []models.AdminLog
	for rows.Next() {
		var log models.AdminLog
		var detailsJSON []byte
		if err := rows.Scan(
			&log.ID,
			&log.Action,
			&log.EntityType,
			&log.EntityID,
			&log.OwnerID,
			&detailsJSON,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan admin log: %w", err)
		}
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
		}
		recentLogs = append(recentLogs, log)
	}
	stats.RecentAdminLogs = recentLogs

	return stats, nil
}
