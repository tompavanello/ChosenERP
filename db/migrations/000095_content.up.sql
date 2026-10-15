-- 000095_content.up.sql
-- AREA DE CONTEUDO DO MEMBRO (roadmap "Palavra/Estudos" + "Cultos Online"):
--   * devocionais em planos com dias + progresso (gamificacao);
--   * videos do YouTube (sermoes/estudos/musica);
--   * culto ao vivo (canal/URL do YouTube por igreja);
--   * Biblia via API externa (api.bible) com cache local.
--
-- Modulo CORE (sem gate de plano). O acesso de staff e por `content.read`/
-- `content.write`; o membro acessa por rotas /me/* (sem permissao).

-- ---------------------------------------------------------------------------
-- 1) Config de conteudo da igreja
-- ---------------------------------------------------------------------------
ALTER TABLE tenants
    ADD COLUMN youtube_channel_id text,
    ADD COLUMN live_stream_url    text,
    ADD COLUMN bible_version_id   text;

-- ---------------------------------------------------------------------------
-- 2) Devocionais: planos + dias + progresso
-- ---------------------------------------------------------------------------
CREATE TABLE devotional_plans (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id       uuid REFERENCES branches(id) ON DELETE CASCADE, -- NULL => igreja toda
    title           text NOT NULL,
    description     text,
    cover_image_url text,
    start_date      date,
    is_active       boolean NOT NULL DEFAULT true,
    created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_devotional_plans_scope ON devotional_plans (tenant_id, branch_id, is_active);

CREATE TABLE devotional_plan_days (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id       uuid REFERENCES branches(id) ON DELETE CASCADE,
    plan_id         uuid NOT NULL REFERENCES devotional_plans(id) ON DELETE CASCADE,
    day_number      int NOT NULL,
    title           text NOT NULL,
    bible_reference text,
    bible_text      text,
    body            text NOT NULL DEFAULT '',
    prayer          text,
    video_url       text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (plan_id, day_number)
);

CREATE INDEX idx_devotional_days_plan ON devotional_plan_days (plan_id, day_number);

CREATE TABLE devotional_progress (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id      uuid NOT NULL REFERENCES devotional_plans(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    member_id    uuid REFERENCES members(id) ON DELETE SET NULL,
    day_number   int NOT NULL,
    completed_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (plan_id, user_id, day_number)
);

CREATE INDEX idx_devotional_progress_user ON devotional_progress (plan_id, user_id);

ALTER TABLE devotional_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE devotional_plan_days ENABLE ROW LEVEL SECURITY;
ALTER TABLE devotional_progress ENABLE ROW LEVEL SECURITY;

-- Planos/dias: leitura inclui o conteudo "igreja toda" (branch NULL); a escrita
-- segue o escopo exato (branch NULL so pela Sede).
CREATE POLICY devotional_plans_sel ON devotional_plans FOR SELECT USING (
    rls_read(tenant_id, branch_id, false)
    OR (tenant_id = current_tenant() AND branch_id IS NULL));
CREATE POLICY devotional_plans_all ON devotional_plans
    USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));

CREATE POLICY devotional_days_sel ON devotional_plan_days FOR SELECT USING (
    rls_read(tenant_id, branch_id, false)
    OR (tenant_id = current_tenant() AND branch_id IS NULL));
CREATE POLICY devotional_days_all ON devotional_plan_days
    USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));

-- Progresso: o membro so enxerga/grava o proprio; a Sede enxerga o tenant.
CREATE POLICY devotional_progress_sel ON devotional_progress FOR SELECT USING (
    is_system()
    OR user_id = current_user_id()
    OR (tenant_id = current_tenant() AND is_headquarters()));
CREATE POLICY devotional_progress_ins ON devotional_progress FOR INSERT
    WITH CHECK (tenant_id = current_tenant() AND user_id = current_user_id());
CREATE POLICY devotional_progress_upd ON devotional_progress FOR UPDATE
    USING (tenant_id = current_tenant() AND user_id = current_user_id())
    WITH CHECK (tenant_id = current_tenant() AND user_id = current_user_id());
CREATE POLICY devotional_progress_del ON devotional_progress FOR DELETE
    USING (tenant_id = current_tenant() AND user_id = current_user_id());

-- ---------------------------------------------------------------------------
-- 3) Videos (YouTube)
-- ---------------------------------------------------------------------------
CREATE TABLE content_videos (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid REFERENCES branches(id) ON DELETE CASCADE, -- NULL => igreja toda
    title         text NOT NULL,
    description   text,
    youtube_id    text NOT NULL,
    kind          text NOT NULL DEFAULT 'sermao',
    speaker       text,
    recorded_at   date,
    thumbnail_url text,
    is_published  boolean NOT NULL DEFAULT true,
    sort_order    int NOT NULL DEFAULT 0,
    created_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT content_videos_kind_check
        CHECK (kind IN ('sermao','estudo','musica','testemunho','outro'))
);

CREATE INDEX idx_content_videos_scope ON content_videos (tenant_id, branch_id, is_published, sort_order);

ALTER TABLE content_videos ENABLE ROW LEVEL SECURITY;

CREATE POLICY content_videos_sel ON content_videos FOR SELECT USING (
    rls_read(tenant_id, branch_id, false)
    OR (tenant_id = current_tenant() AND branch_id IS NULL));
CREATE POLICY content_videos_all ON content_videos
    USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- 4) Cache da Biblia (dado publico, sem tenant/RLS)
-- ---------------------------------------------------------------------------
CREATE TABLE bible_cache (
    cache_key  text PRIMARY KEY,
    payload    jsonb NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- 5) Permissoes de conteudo
-- ---------------------------------------------------------------------------
INSERT INTO permissions (key, module, name) VALUES
    ('content.read',  'content', 'Ler conteudo (devocionais/videos)'),
    ('content.write', 'content', 'Gerir conteudo (devocionais/videos)')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.key IN ('admin_sede', 'super_admin')
  AND p.key IN ('content.read', 'content.write')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM (VALUES
    ('pastor_filial', ARRAY['content.read','content.write']),
    ('secretario',    ARRAY['content.read','content.write']),
    ('pastor',        ARRAY['content.read','content.write']),
    ('lider',         ARRAY['content.read'])
) AS v(role_key, perms)
JOIN roles r ON r.key = v.role_key
CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
JOIN permissions p ON p.key = perm.key
ON CONFLICT DO NOTHING;

-- create_tenant: novas igrejas ja nascem com as permissoes de conteudo.
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
        ('pastor_filial', ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','members.read','reports.read']),
        ('secretario',    ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','members.read','members.write','reports.read','visitors.read']),
        ('lider',         ARRAY['announcements.read','announcements.write','content.read','families.read','members.read','ministries.read','ministries.write','reports.read']),
        ('pastor',        ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','governance.read','members.read','ministries.read','reports.read']),
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
