-- 000054_tenant_branding.down.sql
DROP FUNCTION IF EXISTS public_tenant(text);
ALTER TABLE tenants
    DROP COLUMN logo_url,
    DROP COLUMN brand_color,
    DROP COLUMN favicon_url,
    DROP COLUMN custom_domain;
