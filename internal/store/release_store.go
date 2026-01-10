package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/models"
)

type ReleaseStore interface {
	ListAllReleases(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error)
	ListGlobalReleases(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error)
	ListReleasesByProduct(ctx context.Context, productID string, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error)
	ListReleasesByProductGroup(ctx context.Context, productGroupID string, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error)
	GetRelease(ctx context.Context, releaseID string) (*models.Release, error)
	CreateRelease(ctx context.Context, release *models.Release) error
	UpdateRelease(ctx context.Context, release *models.Release) error
	DeleteRelease(ctx context.Context, releaseID string) error
}

type PostgresReleaseStore struct {
	DB *pgxpool.Pool
}

func NewPostgresReleaseStore(db *pgxpool.Pool) *PostgresReleaseStore {
	return &PostgresReleaseStore{DB: db}
}

func (s *PostgresReleaseStore) ListAllReleases(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, version, created_at
		FROM releases
	`
	countQuery := `SELECT count(*) FROM releases`

	var args []interface{}
	if ownerID != nil {
		query += ` WHERE owner_id = $1`
		countQuery += ` WHERE owner_id = $1`
		args = append(args, ownerID)
	}
	query += ` ORDER BY version DESC`

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
		return nil, 0, fmt.Errorf("failed to get total count of all releases: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list all releases: %w", err)
	}
	defer rows.Close()

	var releases []models.Release
	for rows.Next() {
		var r models.Release
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.ProductID, &r.ProductGroupID, &r.Version, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan release: %w", err)
		}
		releases = append(releases, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return releases, totalCount, nil
}

func (s *PostgresReleaseStore) ListGlobalReleases(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, version, created_at
		FROM releases
		WHERE product_id IS NULL AND product_group_id IS NULL
	`
	countQuery := `SELECT count(*) FROM releases WHERE product_id IS NULL AND product_group_id IS NULL`

	var args []interface{}
	if ownerID != nil {
		query += ` AND owner_id = $1`
		countQuery += ` AND owner_id = $1`
		args = append(args, ownerID)
	}
	query += ` ORDER BY version DESC`

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
		return nil, 0, fmt.Errorf("failed to get total count of global releases: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list releases: %w", err)
	}
	defer rows.Close()

	var releases []models.Release
	for rows.Next() {
		var r models.Release
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.ProductID, &r.ProductGroupID, &r.Version, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan release: %w", err)
		}
		releases = append(releases, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return releases, totalCount, nil
}

func (s *PostgresReleaseStore) ListReleasesByProduct(ctx context.Context, productID string, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, version, created_at
		FROM releases
		WHERE product_id = $1
	`
	countQuery := `SELECT count(*) FROM releases WHERE product_id = $1`

	args := []interface{}{productID}
	if ownerID != nil {
		query += ` AND owner_id = $2`
		countQuery += ` AND owner_id = $2`
		args = append(args, ownerID)
	}
	query += ` ORDER BY version DESC`

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
		return nil, 0, fmt.Errorf("failed to get total count of product releases: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list releases: %w", err)
	}
	defer rows.Close()

	var releases []models.Release
	for rows.Next() {
		var r models.Release
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.ProductID, &r.ProductGroupID, &r.Version, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan release: %w", err)
		}
		releases = append(releases, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return releases, totalCount, nil
}

func (s *PostgresReleaseStore) ListReleasesByProductGroup(ctx context.Context, productGroupID string, ownerID *string, pagination models.PaginationParams) ([]models.Release, int, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, version, created_at
		FROM releases
		WHERE product_group_id = $1
	`
	countQuery := `SELECT count(*) FROM releases WHERE product_group_id = $1`

	args := []interface{}{productGroupID}
	if ownerID != nil {
		query += ` AND owner_id = $2`
		countQuery += ` AND owner_id = $2`
		args = append(args, ownerID)
	}
	query += ` ORDER BY version DESC`

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
		return nil, 0, fmt.Errorf("failed to get total count of product group releases: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list group releases: %w", err)
	}
	defer rows.Close()

	var releases []models.Release
	for rows.Next() {
		var r models.Release
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.ProductID, &r.ProductGroupID, &r.Version, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan release: %w", err)
		}
		releases = append(releases, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return releases, totalCount, nil
}

func (s *PostgresReleaseStore) GetRelease(ctx context.Context, releaseID string) (*models.Release, error) {
	query := `
		SELECT id, owner_id, product_id, product_group_id, version, created_at
		FROM releases
		WHERE id = $1
	`
	row := s.DB.QueryRow(ctx, query, releaseID)
	var r models.Release
	if err := row.Scan(&r.ID, &r.OwnerID, &r.ProductID, &r.ProductGroupID, &r.Version, &r.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to scan release: %w", err)
	}
	return &r, nil
}

func (s *PostgresReleaseStore) CreateRelease(ctx context.Context, release *models.Release) error {
	query := `
		INSERT INTO releases (id, owner_id, product_id, product_group_id, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.DB.Exec(ctx, query, release.ID, release.OwnerID, release.ProductID, release.ProductGroupID, release.Version, release.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}
	return nil
}

func (s *PostgresReleaseStore) UpdateRelease(ctx context.Context, release *models.Release) error {
	query := `
		UPDATE releases
		SET version = $2
		WHERE id = $1
	`
	tag, err := s.DB.Exec(ctx, query, release.ID, release.Version)
	if err != nil {
		return fmt.Errorf("failed to update release: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("release not found")
	}
	return nil
}

func (s *PostgresReleaseStore) DeleteRelease(ctx context.Context, releaseID string) error {
	query := `DELETE FROM releases WHERE id = $1`
	tag, err := s.DB.Exec(ctx, query, releaseID)
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("release not found")
	}
	return nil
}
