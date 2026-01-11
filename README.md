![Clortho the Keymaster](https://cdn.clortho.com/img/banner.png)

![build-badge]
![latest-version-badge]
[![Discord][discord-img]][discord-join]

# Clortho

Clortho is an API server for managing license keys and subscriptions. It handles license generation, validation, and offline verification.

## Features

- **License Management**: Generate, validate, update, and revoke license keys.
- **Flexible Licensing**: Support for Perpetual, Timed, and Trial licenses.
- **Feature & Release Control**: Restrict licenses to specific product features or software releases. Features and releases can be scoped to a product, a product group, or defined globally.
- **Product Management**: Organize licenses by products, releases, and features.
- **Product Groups**: Bundle products together with shared settings.
- **Configurable Separators**: Customize the separator between prefix and key per product (e.g., `-`, `_`, or `#`).
- **Offline Verification**: Option to include signed JWT tokens in license check responses for offline validation.
- **Secure**: JWT Authentication for management endpoints, bcrypt password hashing.
- **Rate Limiting**: Protects against abuse with configurable IP-based rate limiting.
- **IP Restrictions**: Restrict licenses to specific IP addresses or CIDR networks.
- **Response Signing**: Ed25519 signatures for resources (e.g. valid: true/false) for offline verification.
- **Resource Ownership**: Optional `owner_id` field on all resources (Products, Licenses, etc.) to support multi-tenancy and filtering.

## Tech Stack

- **Language**: Go (Golang) 1.25+
- **Framework**: Gin Web Framework
- **Database**: PostgreSQL
- **Driver**: pgx (with connection pooling)
- **Migrations**: golang-migrate
- **Logging**: `log/slog` (structured logging)

## Project Structure

```
clortho/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/            # HTTP handlers (organized by domain)
│   │   │   ├── feature_handlers.go
│   │   │   ├── license_handlers.go
│   │   │   ├── log_handlers.go
│   │   │   ├── product_group_handlers.go
│   │   │   ├── product_handlers.go
│   │   │   ├── release_handlers.go
│   │   │   ├── stats_handlers.go
│   │   │   └── utils.go
│   │   ├── middleware/          # JWT auth, rate limiting, response signing
│   │   │   ├── auth.go
│   │   │   ├── rate_limit.go
│   │   │   └── signature.go
│   │   └── server.go            # Server setup and routing
│   ├── config/                  # Configuration loading
│   ├── database/                # Database connection and migrations
│   ├── models/                  # Data models
│   │   ├── models.go
│   │   └── pagination.go
│   ├── service/                 # Business logic
│   │   ├── license_generator.go
│   │   ├── logging.go
│   │   └── signature.go
│   └── store/                   # Data access layer
│       ├── errors.go            # Custom error types
│       ├── feature_store.go
│       ├── license_store.go
│       ├── log_store.go
│       ├── product_group_store.go
│       ├── product_store.go
│       ├── release_store.go
│       └── stats_store.go
├── migrations/                  # Database migrations
├── scripts/                     # Utility scripts
│   ├── generate_keys.go
│   ├── generate_token.go
│   ├── migrate.go
│   └── verify_token.go
├── config.yaml                  # Configuration file
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.25 or higher
- PostgreSQL database

### Configuration

Clortho is configured using a `config.yaml` file in the root directory.

**Example `config.yaml`**:
```yaml
port: "8080"
database_url: "postgres://user:password@localhost:5432/clortho?sslmode=disable"
admin_secret: "your-super-secret-key"
rate_limit:
  requests_per_second: 5
  burst: 10
  enabled: true
response_signing_private_key: "BASE64_ENCODED_ED25519_PRIVATE_KEY"
```

### Scripts

The `scripts/` directory contains useful utilities:

#### Generate Signing Keys
Generates Ed25519 keys. You must add the output to your `config.yaml` or set them as environment variables.
```bash
go run scripts/generate_keys.go
```

#### Generate Admin Token
Generates a JWT token for accessing protected admin endpoints.
```bash
go run scripts/generate_token.go
```

#### Verify Token
Verifies the self-contained JWT token from the API response.
```bash
go run scripts/verify_token.go -pubkey "YOUR_PUBLIC_KEY" -token "JWT_TOKEN_FROM_RESPONSE"
```

#### Database Migrations
Run database migrations using the provided script or Makefile.
```bash
# Using Makefile (defaults to reading config.yaml)
make migrate-up
make migrate-down

# Using script directly
go run scripts/migrate.go -direction up
```

### Installation & Running

1. **Clone the repository**
   ```bash
   git clone https://github.com/KubiqIO/clortho.git
   cd clortho
   ```

2. **Setup Configuration**
   Create a `config.yaml` file based on the example above.

3. **Setup Database**
   Ensure your PostgreSQL database is running.

4. **Run Tests**
   ```bash
   make test
   ```

5. **Build and Run the Server**
   ```bash
   make build
   ./clortho-server
   ```

6. **Docker Support**
   You can also run Clortho using Docker.

   **Build the image:**
   ```bash
   docker build -t clortho .
   ```

   **Run the container:**
   ```bash
   docker run -p 8080:8080 \
     -v $(pwd)/config.yaml:/app/config.yaml \
     clortho
   ```
   *Note: Ensure your `config.yaml` points to a database accessible from the container (e.g., use `host.docker.internal` instead of `localhost` on some systems, or use a Docker network).*

   **Docker Compose:**
   To run the full stack (App + PostgreSQL):
   ```bash
   docker-compose up --build
   ```

   **Token Generation:**
   The docker image includes a pre-compiled binary to generate admin tokens.
   ```bash
   docker run --rm clortho ./generate-token
   # or if using docker-compose
   docker-compose exec app ./generate-token
   ```


## API Usage

### Authentication

All admin endpoints require a JWT token.  The token can be generated using the `scripts/generate_token.go` script.  The token should be included in the `Authorization` header as a `Bearer` token.

To revoke all admin tokens you can change the `admin_secret` in the `config.yaml` file and then restart the server.

### Response Signing (Security)

Clortho signs all API responses using Ed25519 if a `response_signing_private_key` is configured.
Applications can verify the authenticity of the response using the corresponding public key.

### Public Endpoints

#### Check a License
**Endpoint**: `GET /check`

**Headers**:
- `X-License-Key`: The license key to validate (required)

**Query Parameters** (optional):
| Parameter | Description |
|-----------|-------------|
| `version` | Validate if license is authorized for this release version |
| `feature` | Validate if license has this feature code enabled |

**Examples**:
```bash
# Basic license check
curl -H "X-License-Key: DEMO-aBc123..." http://localhost:8080/check

# Check if license is valid for version 2.0.0
curl -H "X-License-Key: DEMO-aBc123..." "http://localhost:8080/check?version=2.0.0"

# Check if license has SSO feature enabled
curl -H "X-License-Key: DEMO-aBc123..." "http://localhost:8080/check?feature=sso"
```

**Response**:
```json
{
  "expires_at": "2024-12-31T23:59:59Z",
  "reason": "License not valid for version 2.0.0",
  "token": "eyJhbG...",
  "valid": false
}
```

> [!NOTE]
> - The `reason` field is only present when `valid` is `false`
> - If a license has no release restrictions, all versions are allowed
> - Features must be explicitly enabled on the license to pass validation


### Admin Endpoints
**Auth**: Bearer Token (JWT) required.

#### License Management

| Method | Endpoint | Description | Body |
|--------|----------|-------------|------|
| GET | `/admin/keys` | Get license | - |
| POST | `/admin/keys` | Create license | See below |
| PUT | `/admin/keys` | Update license | See below |
| DELETE | `/admin/keys` | Revoke license (Soft Delete) | - |
| DELETE | `/admin/keys/purge` | Delete license (Hard Delete) | - |

**Filtering**: List endpoints (GET) support filtering by `owner_id` query parameter.
- `GET /admin/keys?owner_id=<UUID>`
- `GET /admin/products?owner_id=<UUID>`
- `GET /admin/product-groups?owner_id=<UUID>`
- `GET /admin/product-groups?owner_id=<UUID>`
- `GET /admin/features?product_id=<UUID>`
- `GET /admin/features?product_group_id=<UUID>`
- `GET /admin/releases?product_id=<UUID>`
- `GET /admin/releases?product_group_id=<UUID>`
- `GET /admin/features?owner_id=<UUID>`
- `GET /admin/releases?owner_id=<UUID>`

##### Generate a License
**Endpoint**: `POST /admin/keys`

```bash
curl -X POST http://localhost:8080/admin/keys \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -H "X-License-Key: <YOUR_LICENSE_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "YOUR_PRODUCT_UUID",
    "type": "timed",
    "prefix": "PRO",
    "length": 25,
    "duration": "1y",
    "feature_codes": ["sso", "premium"],
    "release_versions": ["1.0.0", "2.0.0"],
    "allowed_ips": ["192.168.1.10"],
    "allowed_networks": ["10.0.0.0/24"]
  }'
```

**Duration formats**: `5m` (minutes), `1h` (hours), `1d` (days), `2w` (weeks), `3mo` (months), `1y` (years)

##### Update a License
**Endpoint**: `PUT /admin/keys/:key`

```bash
curl -X PUT http://localhost:8080/admin/keys \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -H "X-License-Key: <YOUR_LICENSE_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "perpetual",
    "expires_at": null,
    "feature_codes": ["sso", "premium"],
    "status": "active"
  }'
```

##### Revoke a License (Soft Delete)
**Endpoint**: `DELETE /admin/keys/:key`

Revoking a license sets its status to `revoked`. The license remains in the database but will fail validation checks.

```bash
curl -X DELETE http://localhost:8080/admin/keys \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -H "X-License-Key: <YOUR_LICENSE_KEY>"
```

##### Delete a License (Hard Delete)
**Endpoint**: `DELETE /admin/keys/purge`

Permanently removes the license from the database.

```bash
curl -X DELETE http://localhost:8080/admin/keys/purge \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -H "X-License-Key: <YOUR_LICENSE_KEY>"
```

#### Product Management

| Method | Endpoint | Description | Body |
|--------|----------|-------------|------|
| GET | `/admin/products` | List products | - |
| GET | `/admin/products/:id` | Get product | - |
| POST | `/admin/products` | Create product | `{"name": "...", "license_prefix": "PROD", "license_separator": "_", "license_length": 25, "license_charset": "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", "product_group_id": "YOUR_PRODUCT_GROUP_UUID"}` |
| PUT | `/admin/products/:id` | Update product | Same as create |
| DELETE | `/admin/products/:id` | Delete product | - |

#### Product Group Management

| Method | Endpoint | Description | Body |
|--------|----------|-------------|------|
| GET | `/admin/product-groups` | List groups | - |
| GET | `/admin/product-groups/:id` | Get group | - |
| POST | `/admin/product-groups` | Create group | `{"name": "Suite", "license_prefix": "SUITE", "license_separator": "_", "license_length": 25, "license_charset": "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"}` |
| PUT | `/admin/product-groups/:id` | Update group | Same as create |
| DELETE | `/admin/product-groups/:id` | Delete group | - |

**Settings Inheritance**

Products can belong to a Product Group via the `product_group_id` field. When a product belongs to a group, it inherits the following settings if they are not explicitly set on the product:

| Setting | Inheritance Behavior |
|---------|---------------------|
| `license_prefix` | Uses group's prefix if product's is empty |
| `license_separator` | Uses group's separator if product's is empty/default (`-`) |
| `license_length` | Uses group's length if product's is empty |
| `license_charset` | Uses group's charset if product's is empty |

**Example**:
1. Create a Product Group with `license_prefix: "SUITE"` and `license_separator: "_"`
2. Create a Product with `product_group_id` pointing to that group, leaving `license_prefix` empty
3. When generating a license for that product, the key will be `SUITE_abc123...`

This allows you to define common settings once at the group level and have all products in that group automatically use them, while still allowing individual products to override with their own values.

**Feature & Release Inheritance**

Features and Releases can also be defined at the Product Group level. When creating or updating a license for a Product that belongs to a Group:
- You can assign **Features** that belong to the Product OR the Product Group.
- You can assign **Releases** that belong to the Product OR the Product Group.

This is useful for shared features (e.g., "SSO", "Audit Logging") or releasing a suite of products together under a common version number.

**Global Features & Releases**

Features and Releases can also be defined globally (independent of any Product or Group). These are available for assignment to ANY license regardless of its product association.

#### Feature Management

| Method | Endpoint | Description | Body / Query |
|--------|----------|-------------|--------------|
| GET | `/admin/features` | List features | Optional: `?product_id=...`, `?product_group_id=...`, `?owner_id=...` |
| GET | `/admin/features/global` | List global features | Optional: `?owner_id=...` |
| GET | `/admin/features/:id` | Get single feature | - |
| POST | `/admin/features` | Create feature | `{"name": "...", "code": "...", "product_id": "...", "product_group_id": "..."}` |
| PUT | `/admin/features/:featureId` | Update feature | `{"name": "...", "code": "..."}` |
| DELETE | `/admin/features/:featureId` | Delete feature | - |

#### Release Management

| Method | Endpoint | Description | Body / Query |
|--------|----------|-------------|--------------|
| GET | `/admin/releases` | List releases | Optional: `?product_id=...`, `?product_group_id=...`, `?owner_id=...` |
| GET | `/admin/releases/global` | List global releases | Optional: `?owner_id=...` |
| GET | `/admin/releases/:id` | Get single release | - |
| POST | `/admin/releases` | Create release | `{"version": "...", "product_id": "...", "product_group_id": "..."}` |
| PUT | `/admin/releases/:releaseId` | Update release | `{"version": "..."}` |
| DELETE | `/admin/releases/:releaseId` | Delete release | - |

#### Log Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/logs/license-checks` | Fetch license check logs |
| GET | `/admin/logs/admin-actions` | Fetch admin logs |

##### Fetch License Check Logs
**Endpoint**: `GET /admin/logs/license-checks`

**Query Parameters** (one required):
- `license_key`: Filter by specific license key
- `product_id`: Filter by product UUID
- `product_group_id`: Filter by product group UUID

**Response**:
List of log entries containing:
- `license_key`
- `ip_address`
- `user_agent`
- `status_code` (e.g., 200 for valid, 403 for invalid)
- `request_payload` (features requested, version, etc.)
- `response_payload` (validation result)
- `created_at`

##### Fetch Admin Logs
**Endpoint**: `GET /admin/logs/admin-actions`

**Query Parameters**:
- `actor`: Filter by the admin user (optional)

**Response**:
List of log entries containing:
- `action` (e.g., `CREATE_PRODUCT`, `UPDATE_LICENSE`)
- `entity_type` (e.g., `product`, `license`)
- `entity_id`
- `actor`
- `details` (JSON object with specific changes or request data)
- `created_at`

[discord-img]: https://img.shields.io/badge/discord-join-7289DA.svg?logo=discord&longCache=true&style=flat

[discord-join]: https://discord.gg/heNhcnda8b

[build-badge]: https://github.com/KubiqIO/clortho/actions/workflows/ci.yml/badge.svg?branch=main (https://github.com/KubiqIO/clortho/actions/workflows/ci.yml)

[latest-version-badge]: https://img.shields.io/github/v/tag/kubiqio/clortho?label=version&logo=github

