-- 000080_tenant_pix.down.sql
DROP FUNCTION IF EXISTS public_tenant(text);

CREATE FUNCTION public_tenant(p_slug text)
RETURNS TABLE (
    name        text,
    slug        text,
    logo_url    text,
    brand_color text,
    favicon_url text
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.name, t.slug, t.logo_url, t.brand_color, t.favicon_url
    FROM tenants t
    WHERE t.slug = p_slug
      AND t.is_active
    LIMIT 1;
$$;

GRANT EXECUTE ON FUNCTION public_tenant(text) TO public;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS pix_key,
    DROP COLUMN IF EXISTS pix_name;
