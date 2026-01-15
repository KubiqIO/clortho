package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/models"
)

type ProductStore interface {
	ListProducts(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Product, int, error)
	CreateProduct(ctx context.Context, product *models.Product) error
	GetProduct(ctx context.Context, id string) (*models.Product, error)
	UpdateProduct(ctx context.Context, product *models.Product) error
	DeleteProduct(ctx context.Context, id string) error
}


type PostgresProductStore struct {
	DB *pgxpool.Pool
}

func NewPostgresProductStore(db *pgxpool.Pool) *PostgresProductStore {
	return &PostgresProductStore{DB: db}
}

func (s *PostgresProductStore) ListProducts(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Product, int, error) {
	query := `
		SELECT id, owner_id, name, COALESCE(description, ''), COALESCE(license_prefix, ''), COALESCE(license_separator, '-'), COALESCE(license_charset, ''), COALESCE(license_length, 0), COALESCE(license_type, ''), COALESCE(license_duration, ''), auto_allowed_ip, auto_allowed_ip_limit, product_group_id, created_at, updated_at
		FROM products
	`
	countQuery := `SELECT count(*) FROM products`

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
		return nil, 0, fmt.Errorf("failed to get total count of products: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.LicensePrefix, &p.LicenseSeparator, &p.LicenseCharset, &p.LicenseLength, &p.LicenseType, &p.LicenseDuration, &p.AutoAllowedIP, &p.AutoAllowedIPLimit, &p.ProductGroupID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return products, totalCount, nil
}

func (s *PostgresProductStore) CreateProduct(ctx context.Context, product *models.Product) error {
	query := `
		INSERT INTO products (id, owner_id, name, description, license_prefix, license_separator, license_charset, license_length, license_type, license_duration, auto_allowed_ip, auto_allowed_ip_limit, product_group_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := s.DB.Exec(ctx, query, product.ID, product.OwnerID, product.Name, product.Description, product.LicensePrefix, product.LicenseSeparator, product.LicenseCharset, product.LicenseLength, product.LicenseType, product.LicenseDuration, product.AutoAllowedIP, product.AutoAllowedIPLimit, product.ProductGroupID, product.CreatedAt, product.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	return nil
}

func (s *PostgresProductStore) GetProduct(ctx context.Context, id string) (*models.Product, error) {
	query := `
		SELECT id, owner_id, name, COALESCE(description, ''), COALESCE(license_prefix, ''), COALESCE(license_separator, '-'), COALESCE(license_charset, ''), COALESCE(license_length, 0), COALESCE(license_type, ''), COALESCE(license_duration, ''), auto_allowed_ip, auto_allowed_ip_limit, product_group_id, created_at, updated_at
		FROM products
		WHERE id = $1
	`
	var p models.Product
	err := s.DB.QueryRow(ctx, query, id).Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.LicensePrefix, &p.LicenseSeparator, &p.LicenseCharset, &p.LicenseLength, &p.LicenseType, &p.LicenseDuration, &p.AutoAllowedIP, &p.AutoAllowedIPLimit, &p.ProductGroupID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}
	return &p, nil
}

func (s *PostgresProductStore) UpdateProduct(ctx context.Context, product *models.Product) error {
	query := `
		UPDATE products
		SET name = $1, description = $2, license_prefix = $3, license_separator = $4, license_charset = $5, license_length = $6, license_type = $7, license_duration = $8, auto_allowed_ip = $9, auto_allowed_ip_limit = $10, product_group_id = $11, updated_at = $12
		WHERE id = $13
	`
	
	tag, err := s.DB.Exec(ctx, query, product.Name, product.Description, product.LicensePrefix, product.LicenseSeparator, product.LicenseCharset, product.LicenseLength, product.LicenseType, product.LicenseDuration, product.AutoAllowedIP, product.AutoAllowedIPLimit, product.ProductGroupID, product.UpdatedAt, product.ID)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

func (s *PostgresProductStore) DeleteProduct(ctx context.Context, id string) error {
	query := `DELETE FROM products WHERE id = $1`
	tag, err := s.DB.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}
