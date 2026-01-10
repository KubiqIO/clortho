package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/models"
)

type LicenseStore interface {
	CreateLicense(ctx context.Context, license *models.License) error
	UpdateLicense(ctx context.Context, license *models.License) error
	GetLicenseByKey(ctx context.Context, key string) (*models.License, error)
	GetLicense(ctx context.Context, id string) (*models.License, error)
	DeleteLicense(ctx context.Context, key string) error
	ListLicenses(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.License, int, error)
}

type PostgresLicenseStore struct {
	DB *pgxpool.Pool
}

func NewPostgresLicenseStore(db *pgxpool.Pool) *PostgresLicenseStore {
	return &PostgresLicenseStore{DB: db}
}

func (s *PostgresLicenseStore) CreateLicense(ctx context.Context, license *models.License) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO licenses (
			id, key, owner_id, type, product_id, allowed_ips, allowed_networks, expires_at, created_at, updated_at, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	_, err = tx.Exec(ctx, query,
		license.ID,
		license.Key,
		license.OwnerID,
		license.Type,
		license.ProductID,
		license.AllowedIPs,
		license.AllowedNetworks,
		license.ExpiresAt,
		license.CreatedAt,
		license.UpdatedAt,
		license.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to create license: %w", err)
	}

	if len(license.Features) > 0 {
		fQuery := `
			INSERT INTO license_features (license_id, feature_id)
			SELECT $1, f.id 
			FROM features f
			JOIN products p ON p.id = $2
			WHERE (f.product_id = $2 OR (f.product_group_id = p.product_group_id AND f.product_group_id IS NOT NULL))
			AND f.code = ANY($3)
		`
		if _, err := tx.Exec(ctx, fQuery, license.ID, license.ProductID, license.Features); err != nil {
			return fmt.Errorf("failed to link features: %w", err)
		}
	}

	if len(license.Releases) > 0 {
		rQuery := `
			INSERT INTO license_releases (license_id, release_id)
			SELECT $1, r.id 
			FROM releases r
			JOIN products p ON p.id = $2
			WHERE (r.product_id = $2 OR (r.product_group_id = p.product_group_id AND r.product_group_id IS NOT NULL))
			AND r.version = ANY($3)
		`
		if _, err := tx.Exec(ctx, rQuery, license.ID, license.ProductID, license.Releases); err != nil {
			return fmt.Errorf("failed to link releases: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *PostgresLicenseStore) UpdateLicense(ctx context.Context, license *models.License) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE licenses SET
			type = $1, 
			expires_at = $2, 
			allowed_ips = $3,
			allowed_networks = $4,
			updated_at = $5,
			status = $6
		WHERE key = $7
	`
	res, err := tx.Exec(ctx, query,
		license.Type,
		license.ExpiresAt,
		license.AllowedIPs,
		license.AllowedNetworks,
		license.UpdatedAt,
		license.Status,
		license.Key,
	)
	if err != nil {
		return fmt.Errorf("failed to update license: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: license", ErrNotFound)
	}

	// Re-link features
	if _, err := tx.Exec(ctx, `DELETE FROM license_features WHERE license_id = $1`, license.ID); err != nil {
		return fmt.Errorf("failed to clear features: %w", err)
	}
	if len(license.Features) > 0 {
		fQuery := `
			INSERT INTO license_features (license_id, feature_id)
			SELECT $1, f.id 
			FROM features f
			JOIN products p ON p.id = $2
			WHERE (f.product_id = $2 OR (f.product_group_id = p.product_group_id AND f.product_group_id IS NOT NULL))
			AND f.code = ANY($3)
		`

		if _, err := tx.Exec(ctx, fQuery, license.ID, license.ProductID, license.Features); err != nil {
			return fmt.Errorf("failed to link features: %w", err)
		}
	}

	// Re-link releases
	if _, err := tx.Exec(ctx, `DELETE FROM license_releases WHERE license_id = $1`, license.ID); err != nil {
		return fmt.Errorf("failed to clear releases: %w", err)
	}
	if len(license.Releases) > 0 {
		rQuery := `
			INSERT INTO license_releases (license_id, release_id)
			SELECT $1, r.id 
			FROM releases r
			JOIN products p ON p.id = $2
			WHERE (r.product_id = $2 OR (r.product_group_id = p.product_group_id AND r.product_group_id IS NOT NULL))
			AND r.version = ANY($3)
		`
		if _, err := tx.Exec(ctx, rQuery, license.ID, license.ProductID, license.Releases); err != nil {
			return fmt.Errorf("failed to link releases: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *PostgresLicenseStore) GetLicenseByKey(ctx context.Context, key string) (*models.License, error) {
	query := `
		SELECT 
			l.id, l.key, l.owner_id, l.type, l.product_id, 
			l.allowed_ips::text[], l.allowed_networks::text[], 
			l.expires_at, l.created_at, l.updated_at, l.status,
			COALESCE(array_agg(DISTINCT f.code) FILTER (WHERE f.code IS NOT NULL), '{}')::text[] as features,
			COALESCE(array_agg(DISTINCT r.version) FILTER (WHERE r.version IS NOT NULL), '{}')::text[] as releases
		FROM licenses l
		LEFT JOIN license_features lf ON l.id = lf.license_id
		LEFT JOIN features f ON lf.feature_id = f.id
		LEFT JOIN license_releases lr ON l.id = lr.release_id
		LEFT JOIN releases r ON lr.release_id = r.id
		WHERE l.key = $1
		GROUP BY l.id
	`
	var l models.License
	err := s.DB.QueryRow(ctx, query, key).Scan(
		&l.ID,
		&l.Key,
		&l.OwnerID,
		&l.Type,
		&l.ProductID,
		&l.AllowedIPs,
		&l.AllowedNetworks,
		&l.ExpiresAt,
		&l.CreatedAt,
		&l.UpdatedAt,
		&l.Status,
		&l.Features,
		&l.Releases,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: license", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	return &l, nil
}

func (s *PostgresLicenseStore) DeleteLicense(ctx context.Context, key string) error {
	query := `DELETE FROM licenses WHERE key = $1`
	tag, err := s.DB.Exec(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete license: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: license", ErrNotFound)
	}
	return nil
}

func (s *PostgresLicenseStore) GetLicense(ctx context.Context, id string) (*models.License, error) {
	query := `
		SELECT 
			l.id, l.key, l.owner_id, l.type, l.product_id, 
			l.allowed_ips::text[], l.allowed_networks::text[], 
			l.expires_at, l.created_at, l.updated_at, l.status,
			COALESCE(array_agg(DISTINCT f.code) FILTER (WHERE f.code IS NOT NULL), '{}')::text[] as features,
			COALESCE(array_agg(DISTINCT r.version) FILTER (WHERE r.version IS NOT NULL), '{}')::text[] as releases
		FROM licenses l
		LEFT JOIN license_features lf ON l.id = lf.license_id
		LEFT JOIN features f ON lf.feature_id = f.id
		LEFT JOIN license_releases lr ON l.id = lr.release_id
		LEFT JOIN releases r ON lr.release_id = r.id
		WHERE l.id = $1
		GROUP BY l.id
	`
	var l models.License
	err := s.DB.QueryRow(ctx, query, id).Scan(
		&l.ID,
		&l.Key,
		&l.OwnerID,
		&l.Type,
		&l.ProductID,
		&l.AllowedIPs,
		&l.AllowedNetworks,
		&l.ExpiresAt,
		&l.CreatedAt,
		&l.UpdatedAt,
		&l.Status,
		&l.Features,
		&l.Releases,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: license", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	return &l, nil
}

func (s *PostgresLicenseStore) ListLicenses(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.License, int, error) {
	// Base query for counting
	countQuery := `SELECT count(*) FROM licenses`
	countArgs := []interface{}{}
	if ownerID != nil {
		countQuery += " WHERE owner_id = $1"
		countArgs = append(countArgs, ownerID)
	}

	// Get total count
	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of licenses: %w", err)
	}

	query := `
		SELECT 
			l.id, l.key, l.owner_id, l.type, l.product_id, 
			l.allowed_ips::text[], l.allowed_networks::text[], 
			l.expires_at, l.created_at, l.updated_at, l.status,
			COALESCE(array_agg(DISTINCT f.code) FILTER (WHERE f.code IS NOT NULL), '{}')::text[] as features,
			COALESCE(array_agg(DISTINCT r.version) FILTER (WHERE r.version IS NOT NULL), '{}')::text[] as releases
		FROM licenses l
		LEFT JOIN license_features lf ON l.id = lf.license_id
		LEFT JOIN features f ON lf.feature_id = f.id
		LEFT JOIN license_releases lr ON l.id = lr.release_id
		LEFT JOIN releases r ON lr.release_id = r.id
	`
	
	args := []interface{}{}
	if ownerID != nil {
		query += " WHERE l.owner_id = $1"
		args = append(args, ownerID)
	}
	
	query += " GROUP BY l.id ORDER BY l.created_at DESC"

	limit := pagination.Limit
	if limit <= 0 {
		limit = 10
	}
	page := pagination.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list licenses: %w", err)
	}
	defer rows.Close()

	var licenses []models.License
	for rows.Next() {
		var l models.License
		err := rows.Scan(
			&l.ID,
			&l.Key,
			&l.OwnerID,
			&l.Type,
			&l.ProductID,
			&l.AllowedIPs,
			&l.AllowedNetworks,
			&l.ExpiresAt,
			&l.CreatedAt,
			&l.UpdatedAt,
			&l.Status,
			&l.Features,
			&l.Releases,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan license: %w", err)
		}
		licenses = append(licenses, l)
	}
	
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating licenses: %w", err)
	}

	return licenses, totalCount, nil
}
