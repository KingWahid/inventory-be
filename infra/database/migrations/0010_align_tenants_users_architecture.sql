-- Align tenants and users with ARCHITECTURE.md §7 / ERD (slug, settings, full_name, …).

-- +goose Up

ALTER TABLE tenants ADD COLUMN slug VARCHAR(100);
ALTER TABLE tenants ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE tenants ADD COLUMN settings JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE tenants
SET slug = trim(both '-' FROM regexp_replace(lower(trim(coalesce(name, ''))), '[^a-z0-9]+', '-', 'g'))
         || '-' || substring(replace(id::text, '-', ''), 1, 8)
WHERE slug IS NULL OR trim(slug) = '';

UPDATE tenants SET slug = 'tenant-' || substring(md5(random()::text || id::text), 1, 12)
WHERE slug IS NULL OR trim(slug) = '';

ALTER TABLE tenants ALTER COLUMN slug SET NOT NULL;
CREATE UNIQUE INDEX uq_tenants_slug ON tenants (slug);

ALTER TABLE users ADD COLUMN full_name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN last_login TIMESTAMPTZ;

UPDATE users SET full_name = trim(both '"' FROM split_part(email, '@', 1))
WHERE trim(full_name) = '';

-- +goose Down

ALTER TABLE users DROP COLUMN IF EXISTS last_login;
ALTER TABLE users DROP COLUMN IF EXISTS is_active;
ALTER TABLE users DROP COLUMN IF EXISTS full_name;

DROP INDEX IF EXISTS uq_tenants_slug;

ALTER TABLE tenants DROP COLUMN IF EXISTS settings;
ALTER TABLE tenants DROP COLUMN IF EXISTS is_active;
ALTER TABLE tenants DROP COLUMN IF EXISTS slug;
