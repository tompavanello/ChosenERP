-- 000054_tenant_branding.up.sql
-- White-label por igreja: branding no tenant + leitura publica por slug.
--
-- O slug ja existe em tenants desde a 000002; aqui entram os ativos visuais e o
-- dominio proprio opcional. A tela de login do subdominio (igreja.dominio) le o
-- branding por `public_tenant(slug)` - uma funcao SECURITY DEFINER porque o RLS
-- de tenants exige um tenant de contexto, que ainda nao existe no login.

ALTER TABLE tenants
    ADD COLUMN logo_url      text,
    ADD COLUMN brand_color   text,
    ADD COLUMN favicon_url   text,
    ADD COLUMN custom_domain text;

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
