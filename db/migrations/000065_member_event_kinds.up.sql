-- 000065_member_event_kinds.up.sql
-- Catalogo CONFIGURAVEL de eventos da vida eclesiastica do membro.
--
-- Substitui a lista fixa (MEMBER_HISTORY_KINDS no front e historyKindForStatus
-- no Go) e traz para dentro do ChosenERP os 15 tipos do cadastro legado `cadorg`
-- (MCM/IPI). Cada tipo declara O QUE MOVIMENTA no membro quando lancado:
--   * sets_status       -> nova situacao (members.membership_status);
--   * sets_exit_reason  -> motivo da baixa (members.exit_reason);
--   * clears_exit       -> reativacao: limpa motivo/data de saida;
--   * sets_baptism      -> grava a data do batismo (members.baptism_date);
--   * sets_date_field   -> grava joined_at (membro desde) ou marriage_date.
--
-- O catalogo e GLOBAL do tenant (branch_id sempre NULL): a igreja monta o
-- fluxo uma vez e usa em todas as filiais. Espelha o padrao de `cargos`
-- (migracao 000020): slug unico por tenant, RLS e seed por tenant.
--
-- `members.baptism_date` continua sendo o valor VIGENTE (relatorios atuais nao
-- quebram); o evento de batismo apenas atualiza essa coluna. O historico
-- (member_history) permanece append-only com hash-chain.

-- ---------------------------------------------------------------------------
-- Tabela
-- ---------------------------------------------------------------------------
CREATE TABLE member_event_kinds (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- NULL = evento global do tenant (nao ha catalogo por filial).
    branch_id       uuid REFERENCES branches(id) ON DELETE SET NULL,
    name            text NOT NULL,
    slug            text NOT NULL,
    category        text NOT NULL DEFAULT 'outro',
    tone            text NOT NULL DEFAULT 'zinc', -- tom do badge na UI
    sets_status     text,
    sets_exit_reason text,
    clears_exit     boolean NOT NULL DEFAULT false,
    sets_baptism    boolean NOT NULL DEFAULT false,
    sets_date_field text NOT NULL DEFAULT 'none',
    is_active       boolean NOT NULL DEFAULT true,
    sort_order      int NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT member_event_kinds_category_check CHECK (category IN
        ('batismo','profissao','recepcao','retorno','saida','ordenacao','disciplina','sistema','outro')),
    -- Espelha o CHECK de members.membership_status (000022).
    CONSTRAINT member_event_kinds_status_check CHECK (sets_status IS NULL OR sets_status IN
        ('active','member','inactive','dismissed','transferred','deceased','other')),
    -- Espelha o CHECK de members.exit_reason (000022).
    CONSTRAINT member_event_kinds_exit_check CHECK (sets_exit_reason IS NULL OR sets_exit_reason IN
        ('falecimento','desligamento','transferencia','abandono','ausencia','outro')),
    CONSTRAINT member_event_kinds_date_check CHECK (sets_date_field IN
        ('none','joined_at','marriage_date'))
);

-- Slug unico por tenant: chave estavel usada no backfill e pelo historico.
CREATE UNIQUE INDEX uq_member_event_kinds_tenant_slug ON member_event_kinds(tenant_id, slug);
-- Evita dois rotulos iguais poluindo os badges.
CREATE UNIQUE INDEX uq_member_event_kinds_tenant_name ON member_event_kinds(tenant_id, name);
CREATE INDEX idx_member_event_kinds_tenant ON member_event_kinds(tenant_id);

-- ---------------------------------------------------------------------------
-- RLS (catalogo do escopo do tenant; escrita como cargos)
-- ---------------------------------------------------------------------------
ALTER TABLE member_event_kinds ENABLE ROW LEVEL SECURITY;

CREATE POLICY member_event_kinds_sel ON member_event_kinds
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY member_event_kinds_ins ON member_event_kinds
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, true));
CREATE POLICY member_event_kinds_upd ON member_event_kinds
  FOR UPDATE USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));
CREATE POLICY member_event_kinds_del ON member_event_kinds
  FOR DELETE USING (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- Liga o historico ao tipo (mantem `kind`=slug como chave textual estavel).
-- ---------------------------------------------------------------------------
ALTER TABLE member_history
    ADD COLUMN event_kind_id uuid REFERENCES member_event_kinds(id) ON DELETE SET NULL;

CREATE INDEX idx_member_history_kind ON member_history(event_kind_id);

-- ---------------------------------------------------------------------------
-- Seed
-- ---------------------------------------------------------------------------
-- Funcao reusada pelo seed de tenants existentes e pelo create_tenant (000059).
CREATE OR REPLACE FUNCTION seed_member_event_kinds(p_tenant uuid)
RETURNS void
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    INSERT INTO member_event_kinds
        (tenant_id, branch_id, name, slug, category, tone, sets_status,
         sets_exit_reason, clears_exit, sets_baptism, sets_date_field, sort_order)
    SELECT p_tenant, NULL, v.name, v.slug, v.category, v.tone, v.sets_status,
           v.sets_exit_reason, v.clears_exit, v.sets_baptism, v.sets_date_field, v.sort_order
    FROM (VALUES
        -- Sistema (automaticos; nao aparecem como opcao manual)
        ('Cadastro',                             'cadastro',                  'sistema',    'sky',    NULL,          NULL,           false, false, 'none',        0),
        ('Alteracao de situacao',                'status',                    'sistema',    'zinc',   NULL,          NULL,           false, false, 'none',        1),
        -- Batismo
        ('Batismo e profissao de fe',            'batismo_profissao_fe',      'batismo',    'green',  'active',      NULL,           false, true,  'none',       10),
        ('Batismo de crianca',                   'batismo_crianca',           'batismo',    'green',  NULL,          NULL,           false, true,  'none',       20),
        ('Apresentacao de crianca',              'apresentacao_crianca',      'batismo',    'sky',    NULL,          NULL,           false, false, 'none',       30),
        -- Profissao de fe
        ('Profissao de fe',                      'profissao_fe',              'profissao',  'green',  'active',      NULL,           false, false, 'none',       40),
        -- Recepcao
        ('Recebimento por transferencia',        'recebido_transferencia',    'recepcao',   'green',  'active',      NULL,           false, false, 'joined_at',  50),
        ('Recebimento por jurisdicao',           'recebido_jurisdicao',       'recepcao',   'green',  'active',      NULL,           false, false, 'joined_at',  60),
        -- Retorno
        ('Reativacao',                           'reativacao',                'retorno',    'green',  'active',      NULL,           true,  false, 'none',       70),
        -- Saida
        ('Saida por transferencia',              'saida_transferencia',       'saida',      'amber',  'transferred', 'transferencia',false, false, 'none',       80),
        ('Saida por falecimento',                'saida_falecimento',         'saida',      'zinc',   'deceased',    'falecimento',  false, false, 'none',       90),
        ('Desligamento a pedido do membro',      'desligamento_pedido',       'saida',      'red',    'dismissed',   'desligamento', false, false, 'none',      100),
        ('Desligamento por ato disciplinar',     'desligamento_disciplinar',  'saida',      'red',    'dismissed',   'desligamento', false, false, 'none',      110),
        ('Abandono das atividades (mais de 1 ano)','abandono_atividades',     'saida',      'red',    'dismissed',   'ausencia',     false, false, 'none',      120),
        ('Baixa do Rol',                         'baixa_rol',                 'saida',      'red',    'dismissed',   'desligamento', false, false, 'none',      130),
        -- Ordenacao
        ('Ordenacao de diacono(a)',              'ordenacao_diacono',         'ordenacao',  'indigo', NULL,          NULL,           false, false, 'none',      140),
        ('Ordenacao de presbitero(a)',           'ordenacao_presbitero',      'ordenacao',  'indigo', NULL,          NULL,           false, false, 'none',      150),
        -- Disciplina
        ('Dissolucao das relacoes pastorais',    'dissolucao_pastoral',       'disciplina', 'amber',  'other',       NULL,           false, false, 'none',      160),
        ('Decisao do presbiterio',               'decisao_presbiterio',       'disciplina', 'amber',  'other',       NULL,           false, false, 'none',      170),
        -- Outros
        ('Outros',                               'outro',                     'outro',      'zinc',   NULL,          NULL,           false, false, 'none',      999)
    ) AS v(name, slug, category, tone, sets_status, sets_exit_reason,
           clears_exit, sets_baptism, sets_date_field, sort_order)
    ON CONFLICT DO NOTHING;
$$;

-- Backfill do tipo nas linhas de historico ja existentes. O guard de
-- append-only bloqueia UPDATE, entao o trigger e desabilitado temporariamente
-- (rodando como dono na migracao). O hash-chain nao inclui event_kind_id, logo
-- permanece valido.
ALTER TABLE member_history DISABLE TRIGGER member_history_no_update;

UPDATE member_history h
SET event_kind_id = mk.id
FROM member_event_kinds mk
WHERE mk.tenant_id = h.tenant_id
  AND mk.slug = CASE h.kind
      WHEN 'batismo_infantil' THEN 'batismo_crianca'
      WHEN 'desligamento'     THEN 'desligamento_pedido'
      WHEN 'abandono'         THEN 'abandono_atividades'
      WHEN 'transferencia'    THEN 'saida_transferencia'
      WHEN 'falecimento'      THEN 'saida_falecimento'
      ELSE h.kind
  END;

ALTER TABLE member_history ENABLE TRIGGER member_history_no_update;

-- Seed para os tenants ja existentes.
SELECT seed_member_event_kinds(t.id) FROM tenants t;

GRANT EXECUTE ON FUNCTION seed_member_event_kinds(uuid) TO chosenerp_app;

-- ---------------------------------------------------------------------------
-- create_tenant passa a semear o catalogo para igrejas novas.
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

-- ---------------------------------------------------------------------------
-- reset_operational_data: preserva o catalogo (e configuracao, nao operacao).
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION reset_operational_data()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    keep text[] := ARRAY[
        'tenants', 'branches', 'roles', 'permissions', 'role_permissions',
        'users', 'memberships', 'schema_migrations', 'member_event_kinds'
    ];
    tbls text;
BEGIN
    SELECT string_agg(format('%I.%I', t.table_schema, t.table_name), ', ')
    INTO tbls
    FROM information_schema.tables t
    WHERE t.table_schema = 'public'
      AND t.table_type = 'BASE TABLE'
      AND t.table_name <> ALL (keep);

    IF tbls IS NOT NULL THEN
        EXECUTE 'TRUNCATE TABLE ' || tbls || ' RESTART IDENTITY CASCADE';
    END IF;

    DELETE FROM memberships m
    USING roles r
    WHERE m.role_id = r.id
      AND r.key <> 'super_admin';

    DELETE FROM users u
    WHERE NOT EXISTS (SELECT 1 FROM memberships m WHERE m.user_id = u.id);
END;
$$;

GRANT EXECUTE ON FUNCTION reset_operational_data() TO chosenerp_app;
