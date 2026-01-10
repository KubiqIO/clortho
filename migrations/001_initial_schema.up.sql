CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'license_type') THEN
        CREATE TYPE license_type AS ENUM ('perpetual', 'timed', 'trial');
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'license_status') THEN
        CREATE TYPE license_status AS ENUM ('active', 'revoked', 'expired');
    END IF;
END $$;

CREATE TABLE product_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    license_prefix VARCHAR(50) DEFAULT '',
    license_separator TEXT DEFAULT '-',
    license_charset TEXT,
    license_length INTEGER,
    owner_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    license_prefix VARCHAR(50) DEFAULT '',
    license_separator TEXT DEFAULT '-',
    license_charset TEXT,
    license_length INTEGER,
    license_type TEXT,
    license_duration TEXT,
    product_group_id UUID REFERENCES product_groups(id) ON DELETE SET NULL,
    owner_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE features (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    product_group_id UUID REFERENCES product_groups(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT,
    owner_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Features Uniqueness
CREATE UNIQUE INDEX features_product_code_idx ON features (product_id, code) WHERE product_id IS NOT NULL;
CREATE UNIQUE INDEX features_group_code_idx ON features (product_group_id, code) WHERE product_group_id IS NOT NULL;
CREATE UNIQUE INDEX features_global_code_idx ON features (code) WHERE product_id IS NULL AND product_group_id IS NULL;

CREATE TABLE releases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    product_group_id UUID REFERENCES product_groups(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    owner_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Releases Uniqueness
CREATE UNIQUE INDEX releases_product_version_idx ON releases (product_id, version) WHERE product_id IS NOT NULL;
CREATE UNIQUE INDEX releases_group_version_idx ON releases (product_group_id, version) WHERE product_group_id IS NOT NULL;
CREATE UNIQUE INDEX releases_global_version_idx ON releases (version) WHERE product_id IS NULL AND product_group_id IS NULL;

CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key TEXT NOT NULL UNIQUE,
    type license_type NOT NULL,
    status license_status NOT NULL DEFAULT 'active',
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    allowed_ips inet[],
    allowed_networks cidr[],
    expires_at TIMESTAMP WITH TIME ZONE,
    owner_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE license_features (
    license_id UUID NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (license_id, feature_id)
);

CREATE TABLE license_releases (
    license_id UUID NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    release_id UUID NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (license_id, release_id)
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id UUID NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    processor TEXT NOT NULL,
    processor_sub_id TEXT NOT NULL,
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE license_check_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID REFERENCES products(id),
    license_id UUID REFERENCES licenses(id) ON DELETE SET NULL,
    license_key VARCHAR(255),
    request_payload JSONB DEFAULT '{}',
    response_payload JSONB DEFAULT '{}',
    ip_address VARCHAR(45),
    user_agent TEXT,
    status_code INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE admin_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    action VARCHAR(255) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id UUID,
    actor VARCHAR(255),
    details JSONB DEFAULT '{}',
    owner_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
-- Indexes for owner_id (Tenant Isolation)
CREATE INDEX IF NOT EXISTS idx_product_groups_owner_id ON product_groups(owner_id);
CREATE INDEX IF NOT EXISTS idx_products_owner_id ON products(owner_id);
CREATE INDEX IF NOT EXISTS idx_features_owner_id ON features(owner_id);
CREATE INDEX IF NOT EXISTS idx_releases_owner_id ON releases(owner_id);
CREATE INDEX IF NOT EXISTS idx_licenses_owner_id ON licenses(owner_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_owner_id ON admin_logs(owner_id);

-- Indexes for Foreign Keys not covered by unique constraints
CREATE INDEX IF NOT EXISTS idx_licenses_product_id ON licenses(product_id);

-- Indexes for Logs and Common Lookups
CREATE INDEX IF NOT EXISTS idx_license_check_logs_license_key ON license_check_logs(license_key);
CREATE INDEX IF NOT EXISTS idx_license_check_logs_product_id ON license_check_logs(product_id);

-- Indexes for Sorting (Created At, Name)
CREATE INDEX IF NOT EXISTS idx_licenses_created_at ON licenses(created_at);
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
CREATE INDEX IF NOT EXISTS idx_license_check_logs_created_at ON license_check_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created_at ON admin_logs(created_at);
