package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/models"
)

type FeatureStore interface {
	ListAllFeatures(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error)
	ListGlobalFeatures(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error)
	ListFeaturesByProduct(ctx context.Context, productID string, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error)
	ListFeaturesByProductGroup(ctx context.Context, productGroupID string, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error)
	GetFeature(ctx context.Context, featureID string) (*models.Feature, error)
	CreateFeature(ctx context.Context, feature *models.Feature) error
	UpdateFeature(ctx context.Context, feature *models.Feature) error
	DeleteFeature(ctx context.Context, featureID string) error
}

type PostgresFeatureStore struct {
	DB *pgxpool.Pool
}

func NewPostgresFeatureStore(db *pgxpool.Pool) *PostgresFeatureStore {
	return &PostgresFeatureStore{DB: db}
}

func (s *PostgresFeatureStore) ListAllFeatures(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, name, code, COALESCE(description, ''), created_at
		FROM features
	`
	countQuery := `SELECT count(*) FROM features`

	var args []interface{}
	if ownerID != nil {
		query += ` WHERE owner_id = $1`
		countQuery += ` WHERE owner_id = $1`
		args = append(args, ownerID)
	}
	query += ` ORDER BY name ASC`

	// Pagination
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

	// Get total count
	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of all features: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list all features: %w", err)
	}
	defer rows.Close()

	var features []models.Feature
	for rows.Next() {
		var f models.Feature
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.ProductID, &f.ProductGroupID, &f.Name, &f.Code, &f.Description, &f.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return features, totalCount, nil
}

func (s *PostgresFeatureStore) ListGlobalFeatures(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, name, code, COALESCE(description, ''), created_at
		FROM features
		WHERE product_id IS NULL AND product_group_id IS NULL
	`
	countQuery := `SELECT count(*) FROM features WHERE product_id IS NULL AND product_group_id IS NULL`

	var args []interface{}
	if ownerID != nil {
		query += ` AND owner_id = $1`
		countQuery += ` AND owner_id = $1`
		args = append(args, ownerID)
	}
	query += ` ORDER BY name ASC`

	// Pagination
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

	// Get total count
	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of global features: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list features: %w", err)
	}
	defer rows.Close()

	var features []models.Feature
	for rows.Next() {
		var f models.Feature
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.ProductID, &f.ProductGroupID, &f.Name, &f.Code, &f.Description, &f.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return features, totalCount, nil
}

func (s *PostgresFeatureStore) ListFeaturesByProduct(ctx context.Context, productID string, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, name, code, COALESCE(description, ''), created_at
		FROM features
		WHERE product_id = $1
	`
	countQuery := `SELECT count(*) FROM features WHERE product_id = $1`

	args := []interface{}{productID}
	if ownerID != nil {
		query += ` AND owner_id = $2`
		countQuery += ` AND owner_id = $2`
		args = append(args, ownerID)
	}
	query += ` ORDER BY name ASC`

	// Pagination
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

	// Get total count
	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of product features: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list features: %w", err)
	}
	defer rows.Close()

	var features []models.Feature
	for rows.Next() {
		var f models.Feature
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.ProductID, &f.ProductGroupID, &f.Name, &f.Code, &f.Description, &f.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return features, totalCount, nil
}

func (s *PostgresFeatureStore) ListFeaturesByProductGroup(ctx context.Context, productGroupID string, ownerID *string, pagination models.PaginationParams) ([]models.Feature, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, name, code, COALESCE(description, ''), created_at
		FROM features
		WHERE product_group_id = $1
	`
	countQuery := `SELECT count(*) FROM features WHERE product_group_id = $1`

	args := []interface{}{productGroupID}
	if ownerID != nil {
		query += ` AND owner_id = $2`
		countQuery += ` AND owner_id = $2`
		args = append(args, ownerID)
	}
	query += ` ORDER BY name ASC`

	// Pagination
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

	// Get total count
	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of product group features: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list group features: %w", err)
	}
	defer rows.Close()

	var features []models.Feature
	for rows.Next() {
		var f models.Feature
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.ProductID, &f.ProductGroupID, &f.Name, &f.Code, &f.Description, &f.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan feature: %w", err)
		}
		features = append(features, f)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return features, totalCount, nil
}

func (s *PostgresFeatureStore) GetFeature(ctx context.Context, featureID string) (*models.Feature, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, name, code, COALESCE(description, ''), created_at
		FROM features
		WHERE id = $1
	`
	row := s.DB.QueryRow(ctx, query, featureID)
	var f models.Feature
	if err := row.Scan(&f.ID, &f.OwnerID, &f.ProductID, &f.ProductGroupID, &f.Name, &f.Code, &f.Description, &f.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to scan feature: %w", err)
	}
	return &f, nil
}

func (s *PostgresFeatureStore) CreateFeature(ctx context.Context, feature *models.Feature) error {
	query := `
		INSERT INTO features (id, owner_id, product_id, product_group_id, name, code, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.DB.Exec(ctx, query, feature.ID, feature.OwnerID, feature.ProductID, feature.ProductGroupID, feature.Name, feature.Code, feature.Description, feature.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create feature: %w", err)
	}
	return nil
}

func (s *PostgresFeatureStore) UpdateFeature(ctx context.Context, feature *models.Feature) error {
	query := `
		UPDATE features
		SET name = $2, code = $3, description = $4
		WHERE id = $1
	`
	tag, err := s.DB.Exec(ctx, query, feature.ID, feature.Name, feature.Code, feature.Description)
	if err != nil {
		return fmt.Errorf("failed to update feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("feature not found")
	}
	return nil
}

func (s *PostgresFeatureStore) DeleteFeature(ctx context.Context, featureID string) error {
	query := `DELETE FROM features WHERE id = $1`
	tag, err := s.DB.Exec(ctx, query, featureID)
	if err != nil {
		return fmt.Errorf("failed to delete feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("feature not found")
	}
	return nil
}
