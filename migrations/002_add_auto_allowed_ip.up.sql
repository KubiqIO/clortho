-- Add auto_allowed_ip and auto_allowed_ip_limit to product_groups
ALTER TABLE product_groups ADD COLUMN auto_allowed_ip BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE product_groups ADD COLUMN auto_allowed_ip_limit INTEGER NOT NULL DEFAULT 0;

-- Add auto_allowed_ip and auto_allowed_ip_limit to products
ALTER TABLE products ADD COLUMN auto_allowed_ip BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE products ADD COLUMN auto_allowed_ip_limit INTEGER NOT NULL DEFAULT 0;

-- Add auto_allowed_ip and auto_allowed_ip_limit to licenses
ALTER TABLE licenses ADD COLUMN auto_allowed_ip BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE licenses ADD COLUMN auto_allowed_ip_limit INTEGER NOT NULL DEFAULT 0;
