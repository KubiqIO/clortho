package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/models"
)

type LogStore interface {
	CreateLicenseCheckLog(ctx context.Context, log *models.LicenseCheckLog) error
	CreateAdminLog(ctx context.Context, log *models.AdminLog) error
	GetLicenseCheckLogsByLicenseKey(ctx context.Context, licenseKey string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error)
	GetLicenseCheckLogsByProductID(ctx context.Context, productID string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error)
	GetLicenseCheckLogsByProductGroupID(ctx context.Context, productGroupID string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error)
	ListAdminLogs(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.AdminLog, int, error)
}

type PostgresLogStore struct {
	DB *pgxpool.Pool
}

func NewPostgresLogStore(db *pgxpool.Pool) *PostgresLogStore {
	return &PostgresLogStore{DB: db}
}

func (s *PostgresLogStore) CreateLicenseCheckLog(ctx context.Context, log *models.LicenseCheckLog) error {
	query := `
		INSERT INTO license_check_logs (product_id, license_id, license_key, request_payload, response_payload, ip_address, user_agent, status_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	requestPayloadJSON, err := json.Marshal(log.RequestPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	responsePayloadJSON, err := json.Marshal(log.ResponsePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal response payload: %w", err)
	}

	return s.DB.QueryRow(
		ctx,
		query,
		log.ProductID,
		log.LicenseID,
		log.LicenseKey,
		requestPayloadJSON,
		responsePayloadJSON,
		log.IPAddress,
		log.UserAgent,
		log.StatusCode,
	).Scan(&log.ID, &log.CreatedAt)
}

func (s *PostgresLogStore) CreateAdminLog(ctx context.Context, log *models.AdminLog) error {
	query := `
		INSERT INTO admin_logs (action, entity_type, entity_id, owner_id, details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	return s.DB.QueryRow(
		ctx,
		query,
		log.Action,
		log.EntityType,
		log.EntityID,
		log.OwnerID,
		detailsJSON,
	).Scan(&log.ID, &log.CreatedAt)
}

func (s *PostgresLogStore) GetLicenseCheckLogsByLicenseKey(ctx context.Context, licenseKey string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error) {
	query := `
		SELECT id, product_id, license_id, license_key, request_payload, response_payload, ip_address, user_agent, status_code, created_at
		FROM license_check_logs
		WHERE license_key = $1`
	countQuery := `SELECT count(*) FROM license_check_logs WHERE license_key = $1`

	args := []interface{}{licenseKey}
	if statusCode != nil {
		query += fmt.Sprintf(" AND status_code = $%d", len(args)+1)
		countQuery += fmt.Sprintf(" AND status_code = $%d", len(args)+1)
		args = append(args, *statusCode)
	}

	query += ` ORDER BY created_at DESC`

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

	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of log entries: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query license check logs: %w", err)
	}
	defer rows.Close()

	var logs []models.LicenseCheckLog
	for rows.Next() {
		var log models.LicenseCheckLog
		var requestPayloadJSON, responsePayloadJSON []byte
		if err := rows.Scan(
			&log.ID,
			&log.ProductID,
			&log.LicenseID,
			&log.LicenseKey,
			&requestPayloadJSON,
			&responsePayloadJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.StatusCode,
			&log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan license check log: %w", err)
		}

		if err := json.Unmarshal(requestPayloadJSON, &log.RequestPayload); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal request payload: %w", err)
		}
		if err := json.Unmarshal(responsePayloadJSON, &log.ResponsePayload); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal response payload: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, totalCount, nil
}

func (s *PostgresLogStore) GetLicenseCheckLogsByProductID(ctx context.Context, productID string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error) {
	query := `
		SELECT id, product_id, license_id, license_key, request_payload, response_payload, ip_address, user_agent, status_code, created_at
		FROM license_check_logs
		WHERE product_id = $1`
	countQuery := `SELECT count(*) FROM license_check_logs WHERE product_id = $1`

	args := []interface{}{productID}
	if statusCode != nil {
		query += fmt.Sprintf(" AND status_code = $%d", len(args)+1)
		countQuery += fmt.Sprintf(" AND status_code = $%d", len(args)+1)
		args = append(args, *statusCode)
	}

	query += ` ORDER BY created_at DESC`

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
		return nil, 0, fmt.Errorf("failed to get total count of log entries: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query license check logs: %w", err)
	}
	defer rows.Close()

	var logs []models.LicenseCheckLog
	for rows.Next() {
		var log models.LicenseCheckLog
		var requestPayloadJSON, responsePayloadJSON []byte
		if err := rows.Scan(
			&log.ID,
			&log.ProductID,
			&log.LicenseID,
			&log.LicenseKey,
			&requestPayloadJSON,
			&responsePayloadJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.StatusCode,
			&log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan license check log: %w", err)
		}

		if err := json.Unmarshal(requestPayloadJSON, &log.RequestPayload); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal request payload: %w", err)
		}
		if err := json.Unmarshal(responsePayloadJSON, &log.ResponsePayload); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal response payload: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, totalCount, nil
}

func (s *PostgresLogStore) GetLicenseCheckLogsByProductGroupID(ctx context.Context, productGroupID string, statusCode *int, pagination models.PaginationParams) ([]models.LicenseCheckLog, int, error) {
	query := `
		SELECT l.id, l.product_id, l.license_id, l.license_key, l.request_payload, l.response_payload, l.ip_address, l.user_agent, l.status_code, l.created_at
		FROM license_check_logs l
		JOIN products p ON l.product_id = p.id
		WHERE p.product_group_id = $1`
	countQuery := `
		SELECT count(*)
		FROM license_check_logs l
		JOIN products p ON l.product_id = p.id
		WHERE p.product_group_id = $1`

	args := []interface{}{productGroupID}
	if statusCode != nil {
		query += fmt.Sprintf(" AND l.status_code = $%d", len(args)+1)
		countQuery += fmt.Sprintf(" AND l.status_code = $%d", len(args)+1)
		args = append(args, *statusCode)
	}

	query += ` ORDER BY l.created_at DESC`

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

	var totalCount int
	err := s.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count of log entries: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query license check logs: %w", err)
	}
	defer rows.Close()

	var logs []models.LicenseCheckLog
	for rows.Next() {
		var log models.LicenseCheckLog
		var requestPayloadJSON, responsePayloadJSON []byte
		if err := rows.Scan(
			&log.ID,
			&log.ProductID,
			&log.LicenseID,
			&log.LicenseKey,
			&requestPayloadJSON,
			&responsePayloadJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.StatusCode,
			&log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan license check log: %w", err)
		}

		if err := json.Unmarshal(requestPayloadJSON, &log.RequestPayload); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal request payload: %w", err)
		}
		if err := json.Unmarshal(responsePayloadJSON, &log.ResponsePayload); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal response payload: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, totalCount, nil
}

func (s *PostgresLogStore) ListAdminLogs(ctx context.Context, ownerID *string, pagination models.PaginationParams) ([]models.AdminLog, int, error) {
	query := `
		SELECT id, action, entity_type, entity_id, owner_id, details, created_at
		FROM admin_logs
	`
	countQuery := `SELECT count(*) FROM admin_logs`
	var args []interface{}
	if ownerID != nil {
		query += ` WHERE owner_id = $1`
		countQuery += ` WHERE owner_id = $1`
		args = append(args, ownerID)
	}
	query += ` ORDER BY created_at DESC`

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
		return nil, 0, fmt.Errorf("failed to get total count of admin logs: %w", err)
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query admin logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AdminLog
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
			return nil, 0, fmt.Errorf("failed to scan admin log: %w", err)
		}

		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal details: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, totalCount, nil
}
