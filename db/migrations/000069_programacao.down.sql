-- 000069_programacao.down.sql
-- Reverte a Fase 1. Best-effort:
--   * a origem volta a ser o `culto_id` apenas para as ocorrencias de
--     programacao; a origem 'escala' some (a coluna nao existia antes);
--   * o `kind` fixo volta para `event_kind_id` mapeando pelos slugs antigos.

-- 1) church_events: volta o culto_id
ALTER TABLE church_events ADD COLUMN IF NOT EXISTS culto_id uuid;
UPDATE church_events SET culto_id = origin_id WHERE origin = 'programacao';
DROP INDEX IF EXISTS idx_church_events_origin;
ALTER TABLE church_events DROP CONSTRAINT IF EXISTS church_events_origin_check;
ALTER TABLE church_events DROP COLUMN IF EXISTS origin;
ALTER TABLE church_events DROP COLUMN IF EXISTS origin_id;

-- 2) programacoes: volta o event_kind_id a partir do kind
ALTER TABLE programacoes ADD COLUMN IF NOT EXISTS event_kind_id uuid;
UPDATE programacoes p SET event_kind_id = k.id
FROM event_kinds k
WHERE k.tenant_id = p.tenant_id
  AND k.slug = CASE p.kind
      WHEN 'culto'   THEN 'culto'
      WHEN 'oracao'  THEN 'vigilia'
      WHEN 'celula'  THEN 'pequeno_grupo'
      WHEN 'ebd'     THEN 'escola_biblica'
      WHEN 'ensaio'  THEN 'culto_especial'
      WHEN 'reuniao' THEN 'reuniao'
      ELSE 'outros' END;
ALTER TABLE programacoes DROP CONSTRAINT IF EXISTS programacoes_kind_check;
ALTER TABLE programacoes DROP COLUMN IF EXISTS kind;

DROP POLICY IF EXISTS programacoes_sel ON programacoes;
DROP POLICY IF EXISTS programacoes_ins ON programacoes;
DROP POLICY IF EXISTS programacoes_upd ON programacoes;
DROP POLICY IF EXISTS programacoes_del ON programacoes;
ALTER TABLE programacoes RENAME TO cultos;
ALTER INDEX IF EXISTS idx_programacoes_branch RENAME TO idx_cultos_branch;
CREATE INDEX IF NOT EXISTS idx_cultos_kind ON cultos(event_kind_id);
CREATE POLICY cultos_sel ON cultos
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY cultos_ins ON cultos
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY cultos_upd ON cultos
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY cultos_del ON cultos
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- 3) Desfaz o catalogo selado (reativa tudo; as cores/nomes ficam como estao).
UPDATE event_kinds SET is_active = true;
DROP FUNCTION IF EXISTS seed_event_kinds(uuid);

-- 4) create_tenant volta a NAO semear tipos de evento (versao 000065).
CREATE OR REPLACE FUNCTION create_tenant(
    p_name text,
    p_slug text,
    p_plan text DEFAULT 'starter'
)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_tenant uuid;
    v_slug   text := lower(trim(p_slug));
    reserved text[] := ARRAY['app', 'www', 'api', 'admin', 'localhost'];
BEGIN
    IF p_name IS NULL OR trim(p_name) = '' THEN
        RAISE EXCEPTION 'nome da igreja obrigatorio' USING ERRCODE = '22023';
    END IF;
    IF v_slug !~ '^[a-z0-9][a-z0-9-]{1,38}$' THEN
        RAISE EXCEPTION 'slug invalido (minusculas, numeros e hifen; 2 a 39 chars)' USING ERRCODE = '22023';
    END IF;
    IF v_slug = ANY (reserved) THEN
        RAISE EXCEPTION 'slug reservado' USING ERRCODE = '22023';
    END IF;
    IF EXISTS (SELECT 1 FROM tenants WHERE slug = v_slug) THEN
        RAISE EXCEPTION 'slug ja existe' USING ERRCODE = '23505';
    END IF;

    INSERT INTO tenants (name, slug, plan)
    VALUES (trim(p_name), v_slug, COALESCE(NULLIF(trim(p_plan), ''), 'starter'))
    RETURNING id INTO v_tenant;

    INSERT INTO branches (tenant_id, name, slug, kind)
    VALUES (v_tenant, 'Sede Matriz', 'matriz', 'matriz');

    PERFORM seed_member_event_kinds(v_tenant);

    INSERT INTO roles (tenant_id, key, name, is_system) VALUES
        (v_tenant, 'super_admin',   'Super Admin',                true),
        (v_tenant, 'admin_sede',    'Admin da Sede',              true),
        (v_tenant, 'pastor_filial', 'Pastor da Filial',           true),
        (v_tenant, 'tesoureiro',    'Tesoureiro',                 true),
        (v_tenant, 'secretario',    'Secretario(a)',              true),
        (v_tenant, 'membro',        'Membro',                     true),
        (v_tenant, 'lider',         'Lider de Ministerio/Celula', true),
        (v_tenant, 'pastor',        'Pastor/Conselheiro',         true),
        (v_tenant, 'contador',      'Contador Externo',           true),
        (v_tenant, 'visitante',     'Visitante',                  true);

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = v_tenant AND r.key IN ('super_admin', 'admin_sede');

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM (VALUES
        ('pastor_filial', ARRAY['families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','members.read','reports.read']),
        ('secretario',    ARRAY['families.read','members.read','members.write','reports.read','visitors.read']),
        ('lider',         ARRAY['families.read','members.read','ministries.read','ministries.write','reports.read']),
        ('pastor',        ARRAY['families.read','governance.read','members.read','ministries.read','reports.read']),
        ('contador',      ARRAY['finance.read','reports.read'])
    ) AS v(role_key, perms)
    JOIN roles r ON r.tenant_id = v_tenant AND r.key = v.role_key
    CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
    JOIN permissions p ON p.key = perm.key
    ON CONFLICT DO NOTHING;

    RETURN v_tenant;
END;
$$;

GRANT EXECUTE ON FUNCTION create_tenant(text, text, text) TO chosenerp_app;
