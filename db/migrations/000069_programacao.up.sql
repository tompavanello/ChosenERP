-- 000069_programacao.up.sql
-- Fase 1 da consolidacao: "Programacao" generica + vinculo unificado na agenda.
--
-- 1) `cultos` -> `programacoes`, com um TIPO FIXO (`kind`): culto, oracao,
--    celula, ebd, ensaio, reuniao, outro. "Culto" deixa de ser um modulo solto
--    e vira apenas kind='culto'; relogio de oracao vira kind='oracao' etc.
--    A coluna `event_kind_id` sai: a publicacao resolve kind -> event_kinds.
-- 2) `event_kinds` passa a ser um CATALOGO SELADO (sem CRUD): seed dos 7 tipos
--    fixos e desativacao dos demais (nao apaga - eventos antigos preservam o
--    tipo e a cor).
-- 3) Toda ocorrencia da agenda ganha a ORIGEM (`church_events.origin` +
--    `origin_id`): manual | programacao | escala. Substitui o `culto_id`
--    (direcao unica, evita vinculos contraditorios).

-- ---------------------------------------------------------------------------
-- 1) cultos -> programacoes
-- ---------------------------------------------------------------------------
ALTER TABLE cultos RENAME TO programacoes;

ALTER INDEX IF EXISTS idx_cultos_branch RENAME TO idx_programacoes_branch;
ALTER INDEX IF EXISTS idx_cultos_kind RENAME TO idx_programacoes_kind;

DROP POLICY IF EXISTS cultos_sel ON programacoes;
DROP POLICY IF EXISTS cultos_ins ON programacoes;
DROP POLICY IF EXISTS cultos_upd ON programacoes;
DROP POLICY IF EXISTS cultos_del ON programacoes;
CREATE POLICY programacoes_sel ON programacoes
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY programacoes_ins ON programacoes
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY programacoes_upd ON programacoes
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY programacoes_del ON programacoes
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- Tipo fixo. Conjunto enxuto e fechado (o catalogo event_kinds fica selado).
ALTER TABLE programacoes ADD COLUMN kind text NOT NULL DEFAULT 'culto';
ALTER TABLE programacoes ADD CONSTRAINT programacoes_kind_check
    CHECK (kind IN ('culto','oracao','celula','ebd','ensaio','reuniao','outro'));

-- Backfill do tipo a partir do antigo event_kind_id (quando houver).
UPDATE programacoes p SET kind = CASE k.slug
        WHEN 'culto'                    THEN 'culto'
        WHEN 'culto_especial'           THEN 'culto'
        WHEN 'santa_ceia'               THEN 'culto'
        WHEN 'vigilia'                  THEN 'oracao'
        WHEN 'reuniao'                  THEN 'reuniao'
        WHEN 'pequeno_grupo'            THEN 'celula'
        WHEN 'escola_biblica'           THEN 'ebd'
        WHEN 'escola_biblica_dominical' THEN 'ebd'
        ELSE 'outro' END
FROM event_kinds k
WHERE p.event_kind_id = k.id;

ALTER TABLE programacoes DROP COLUMN IF EXISTS event_kind_id;
DROP INDEX IF EXISTS idx_programacoes_kind;

-- ---------------------------------------------------------------------------
-- 2) Agenda unificada: origem da ocorrencia
-- ---------------------------------------------------------------------------
ALTER TABLE church_events ADD COLUMN origin text NOT NULL DEFAULT 'manual';
ALTER TABLE church_events ADD COLUMN origin_id uuid;
ALTER TABLE church_events ADD CONSTRAINT church_events_origin_check
    CHECK (origin IN ('manual','programacao','escala'));

-- Backfill a partir dos vinculos antigos.
UPDATE church_events SET origin = 'programacao', origin_id = culto_id
WHERE culto_id IS NOT NULL;
UPDATE church_events e SET origin = 'escala', origin_id = r.id
FROM rosters r
WHERE r.event_id = e.id AND r.generated_event AND e.origin = 'manual';

DROP INDEX IF EXISTS idx_church_events_culto;
ALTER TABLE church_events DROP COLUMN IF EXISTS culto_id;

CREATE INDEX idx_church_events_origin ON church_events(origin, origin_id);

-- ---------------------------------------------------------------------------
-- 3) Catalogo de tipos SELADO
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION seed_event_kinds(p_tenant uuid)
RETURNS void
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    INSERT INTO event_kinds (tenant_id, branch_id, name, slug, sort_order, is_active, color)
    SELECT p_tenant, NULL, v.name, v.slug, v.sort_order, true, v.color
    FROM (VALUES
        ('Culto',                  'culto',    10, '#0ea5e9'),
        ('Relogio de Oracao',      'oracao',   20, '#8b5cf6'),
        ('Celula / Pequeno Grupo', 'celula',   30, '#10b981'),
        ('Escola Biblica',         'ebd',      40, '#14b8a6'),
        ('Ensaio',                 'ensaio',   50, '#6366f1'),
        ('Reuniao',                'reuniao',  60, '#f59e0b'),
        ('Outro',                  'outro',   999, '#71717a')
    ) AS v(name, slug, sort_order, color)
    ON CONFLICT (tenant_id, slug) DO UPDATE
        SET name = EXCLUDED.name, sort_order = EXCLUDED.sort_order,
            is_active = true, color = EXCLUDED.color;
$$;

-- Garante os tipos fixos nos tenants existentes e DESATIVA os demais
-- (catalogo selado). Nao apaga: eventos antigos preservam tipo/cor.
SELECT seed_event_kinds(t.id) FROM tenants t;
UPDATE event_kinds SET is_active = false
WHERE slug NOT IN ('culto','oracao','celula','ebd','ensaio','reuniao','outro');

GRANT EXECUTE ON FUNCTION seed_event_kinds(uuid) TO chosenerp_app;

-- ---------------------------------------------------------------------------
-- 4) create_tenant passa a semear tambem os tipos de evento
-- ---------------------------------------------------------------------------
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

    -- Filial raiz (Matriz) -- sem ela, lancamentos que exigem branch_id falham.
    INSERT INTO branches (tenant_id, name, slug, kind)
    VALUES (v_tenant, 'Sede Matriz', 'matriz', 'matriz');

    -- Catalogo de eventos da vida eclesiastica (000065).
    PERFORM seed_member_event_kinds(v_tenant);

    -- Catalogo selado de tipos de evento (000069).
    PERFORM seed_event_kinds(v_tenant);

    -- Papeis padrao da igreja.
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

    -- super_admin e admin_sede: todas as permissoes.
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = v_tenant AND r.key IN ('super_admin', 'admin_sede');

    -- Demais papeis: conjunto curado por funcao (mesma matriz do seed).
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
