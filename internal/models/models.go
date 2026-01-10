package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductGroup struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       *string    `json:"owner_id,omitempty"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	LicensePrefix     string    `json:"license_prefix,omitempty"`
	LicenseSeparator  string    `json:"license_separator,omitempty"`
	LicenseCharset    string    `json:"license_charset,omitempty"`
	LicenseLength     int       `json:"license_length,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Product struct {
	ID             uuid.UUID  `json:"id"`
	OwnerID        *string    `json:"owner_id,omitempty"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	LicensePrefix    string     `json:"license_prefix,omitempty"`
	LicenseSeparator string     `json:"license_separator,omitempty"`
	LicenseCharset    string     `json:"license_charset,omitempty"`
	LicenseLength     int        `json:"license_length,omitempty"`
	LicenseType       LicenseType `json:"license_type,omitempty"`
	LicenseDuration   string     `json:"license_duration,omitempty"`
	ProductGroupID    *uuid.UUID `json:"product_group_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}



type Feature struct {
	ID             uuid.UUID  `json:"id"`
	OwnerID        *string    `json:"owner_id,omitempty"`
	ProductID      *uuid.UUID `json:"product_id,omitempty"`
	ProductGroupID *uuid.UUID `json:"product_group_id,omitempty"`
	Name           string     `json:"name"`
	Code           string     `json:"code"`
	Description    string     `json:"description,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Release struct {
	ID             uuid.UUID  `json:"id"`
	OwnerID        *string    `json:"owner_id,omitempty"`
	ProductID      *uuid.UUID `json:"product_id,omitempty"`
	ProductGroupID *uuid.UUID `json:"product_group_id,omitempty"`
	Version        string     `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
}

type LicenseType string

const (
	LicenseTypePerpetual LicenseType = "perpetual"
	LicenseTypeTimed     LicenseType = "timed"
	LicenseTypeTrial     LicenseType = "trial"
)

type LicenseStatus string

const (
	LicenseStatusActive  LicenseStatus = "active"
	LicenseStatusRevoked LicenseStatus = "revoked"
	LicenseStatusExpired LicenseStatus = "expired"
)

type License struct {
	ID              uuid.UUID     `json:"id"`
	Key             string        `json:"key"`
	OwnerID         *string       `json:"owner_id,omitempty"`
	Type            LicenseType   `json:"type"`
	ProductID       uuid.UUID     `json:"product_id"`
	AllowedIPs      []string      `json:"allowed_ips,omitempty"`
	AllowedNetworks []string      `json:"allowed_networks,omitempty"`
	ExpiresAt       *time.Time    `json:"expires_at,omitempty"`
	Features        []string      `json:"features,omitempty"`
	Releases        []string      `json:"releases,omitempty"`
	Status          LicenseStatus `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type LicenseCheckLog struct {
	ID              uuid.UUID              `json:"id"`
	ProductID       *uuid.UUID             `json:"product_id,omitempty"`
	LicenseID       *uuid.UUID             `json:"license_id,omitempty"`
	LicenseKey      string                 `json:"license_key,omitempty"`
	RequestPayload  map[string]interface{} `json:"request_payload"`
	ResponsePayload map[string]interface{} `json:"response_payload"`
	IPAddress       string                 `json:"ip_address"`
	UserAgent       string                 `json:"user_agent"`
	StatusCode      int                    `json:"status_code"`
	CreatedAt       time.Time              `json:"created_at"`
}

type AdminLog struct {
	ID         uuid.UUID              `json:"id"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   *uuid.UUID             `json:"entity_id,omitempty"`
	OwnerID    *string                `json:"owner_id,omitempty"`
	Details    map[string]interface{} `json:"details"`
	CreatedAt  time.Time              `json:"created_at"`
}

type DashboardStats struct {
	TotalProducts      int `json:"total_products"`
	TotalProductsChange int `json:"total_products_change"`
	TotalLicenses      int `json:"total_licenses"`
	TotalLicensesChange int `json:"total_licenses_change"`
	TotalLicenseChecks int `json:"total_license_checks"`
	TotalLicenseChecksChange int `json:"total_license_checks_change"`
	TotalLicenseCheckErrors int `json:"total_license_check_errors"`
	TotalLicenseCheckErrorsChange int `json:"total_license_check_errors_change"`
	TotalAdminActions  int `json:"total_admin_actions"`
	RecentAdminLogs    []AdminLog `json:"recent_admin_logs"`
}
