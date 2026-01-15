package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/models"
)

type ProductGroupStore interface {
	ListProductGroups(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.ProductGroup, int, error)
	CreateProductGroup(ctx context.Context, group *models.ProductGroup) error
	GetProductGroup(ctx context.Context, id string) (*models.ProductGroup, error)
	UpdateProductGroup(ctx context.Context, group *models.ProductGroup) error
	DeleteProductGroup(ctx context.Context, id string) error
}

type PostgresProductGroupStore struct {
	DB *pgxpool.Pool
}

func NewPostgresProductGroupStore(db *pgxpool.Pool) *PostgresProductGroupStore {
	return &PostgresProductGroupStore{DB: db}
}

func (s *PostgresProductGroupStore) ListProductGroups(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.ProductGroup, int, error) {
	query := `
		SELECT id, owner_id, name, COALESCE(description, ''), COALESCE(license_prefix, ''), COALESCE(license_separator, '-'), COALESCE(license_charset, ''), COALESCE(license_length, 0), auto_allowed_ip, auto_allowed_ip_limit, created_at, updated_at
		FROM product_groups
	`
	countQuery := `SELECT count(*) FROM product_groups`

	args := []interface{}{}
	if ownerID != nil {
		query += ` WHERE owner_id = $1`
		countQuery += ` WHERE owner_id = $1`
		args = append(args, ownerID)
	}
	query += ` ORDER BY name ASC`

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

	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of product groups: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list product groups: %w", err)
	}
	defer rows.Close()

	var groups []models.ProductGroup
	for rows.Next() {
		var g models.ProductGroup
		if err := rows.Scan(&g.ID, &g.OwnerID, &g.Name, &g.Description, &g.LicensePrefix, &g.LicenseSeparator, &g.LicenseCharset, &g.LicenseLength, &g.AutoAllowedIP, &g.AutoAllowedIPLimit, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan product group: %w", err)
		}
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return groups, totalCount, nil
}

func (s *PostgresProductGroupStore) CreateProductGroup(ctx context.Context, group *models.ProductGroup) error {
	query := `
		INSERT INTO product_groups (id, owner_id, name, description, license_prefix, license_separator, license_charset, license_length, auto_allowed_ip, auto_allowed_ip_limit, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.DB.Exec(ctx, query, group.ID, group.OwnerID, group.Name, group.Description, group.LicensePrefix, group.LicenseSeparator, group.LicenseCharset, group.LicenseLength, group.AutoAllowedIP, group.AutoAllowedIPLimit, group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create product group: %w", err)
	}
	return nil
}

func (s *PostgresProductGroupStore) GetProductGroup(ctx context.Context, id string) (*models.ProductGroup, error) {
	query := `
		SELECT id, owner_id, name, COALESCE(description, ''), COALESCE(license_prefix, ''), COALESCE(license_separator, '-'), COALESCE(license_charset, ''), COALESCE(license_length, 0), auto_allowed_ip, auto_allowed_ip_limit, created_at, updated_at
		FROM product_groups
		WHERE id = $1
	`
	var g models.ProductGroup
	err := s.DB.QueryRow(ctx, query, id).Scan(&g.ID, &g.OwnerID, &g.Name, &g.Description, &g.LicensePrefix, &g.LicenseSeparator, &g.LicenseCharset, &g.LicenseLength, &g.AutoAllowedIP, &g.AutoAllowedIPLimit, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get product group: %w", err)
	}
	return &g, nil
}

func (s *PostgresProductGroupStore) UpdateProductGroup(ctx context.Context, group *models.ProductGroup) error {
	query := `
		UPDATE product_groups
		SET name = $1, description = $2, license_prefix = $3, license_separator = $4, license_charset = $5, license_length = $6, auto_allowed_ip = $7, auto_allowed_ip_limit = $8, updated_at = $9
		WHERE id = $10
	`
	tag, err := s.DB.Exec(ctx, query, group.Name, group.Description, group.LicensePrefix, group.LicenseSeparator, group.LicenseCharset, group.LicenseLength, group.AutoAllowedIP, group.AutoAllowedIPLimit, group.UpdatedAt, group.ID)
	if err != nil {
		return fmt.Errorf("failed to update product group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product group not found")
	}
	return nil
}

func (s *PostgresProductGroupStore) DeleteProductGroup(ctx context.Context, id string) error {
	query := `DELETE FROM product_groups WHERE id = $1`
	tag, err := s.DB.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product group not found")
	}
	return nil
}
