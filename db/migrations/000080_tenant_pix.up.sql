-- 000080_tenant_pix.up.sql
-- Pix da igreja para a tela "Contribuir" do app do membro.
--
-- `pix_key` e o valor da chave (CPF/CNPJ/e-mail/telefone/aleatoria) e `pix_name`
-- e o nome do beneficiario exibido. Dado publico (serve para doar), entao sai
-- tambem no `public_tenant(slug)` usado pela tela de login/contribuir.
ALTER TABLE tenants
    ADD COLUMN pix_key  text,
    ADD COLUMN pix_name text;

-- CREATE OR REPLACE nao pode mudar o tipo de retorno: recria a funcao e
-- restaura o GRANT.
DROP FUNCTION IF EXISTS public_tenant(text);

CREATE FUNCTION public_tenant(p_slug text)
RETURNS TABLE (
    name        text,
    slug        text,
    logo_url    text,
    brand_color text,
    favicon_url text,
    pix_key     text,
    pix_name    text
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.name, t.slug, t.logo_url, t.brand_color, t.favicon_url, t.pix_key, t.pix_name
    FROM tenants t
    WHERE t.slug = p_slug
      AND t.is_active
    LIMIT 1;
$$;

GRANT EXECUTE ON FUNCTION public_tenant(text) TO public;
