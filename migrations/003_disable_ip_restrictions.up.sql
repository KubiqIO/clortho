-- Preserve existing enforcement. Disabling skips IP/CIDR checks and auto-registration.
ALTER TABLE products ADD COLUMN ip_restrictions_disabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE licenses ADD COLUMN ip_restrictions_disabled BOOLEAN NOT NULL DEFAULT FALSE;
