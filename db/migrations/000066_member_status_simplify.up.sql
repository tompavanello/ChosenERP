-- 000066_member_status_simplify.up.sql
-- Simplifica a situacao do membro (antes 7 valores, 000022) para 3, conforme o
-- modelo do cliente:
--   * active   -> Ativo Professo;
--   * member   -> Ativo Nao Professo;
--   * inactive -> Inativo, subdividido pelo MOTIVO da inatividade (exit_reason).
--
-- Os antigos dismissed / transferred / deceased / other deixam de existir como
-- situacao e passam a 'inactive', preservando a distincao no exit_reason. Quem
-- ficou 'inactive' sem motivo recebe 'outro'. A situacao 'inactive' passa a
-- exigir sempre um motivo.
--
-- Revisa tambem o catalogo da vida eclesiastica (member_event_kinds, 000065):
-- os tipos de saida que gravavam dismissed/transferred/deceased/other agora
-- gravam 'inactive' + o motivo correspondente.

-- ---------------------------------------------------------------------------
-- 1) members: motivo/data antes de colapsar a situacao
-- ---------------------------------------------------------------------------
UPDATE members SET
    exit_reason = COALESCE(exit_reason, CASE membership_status
        WHEN 'dismissed'   THEN 'desligamento'
        WHEN 'transferred' THEN 'transferencia'
        WHEN 'deceased'    THEN 'falecimento'
        ELSE 'outro' END),
    exited_at = COALESCE(exited_at, now()::date)
WHERE membership_status IN ('dismissed','transferred','deceased','other','inactive');

UPDATE members SET membership_status = 'inactive'
WHERE membership_status IN ('dismissed','transferred','deceased','other');

-- ---------------------------------------------------------------------------
-- 2) members: fecha o dominio em 3 valores + exige motivo quando inativo
-- ---------------------------------------------------------------------------
ALTER TABLE members DROP CONSTRAINT IF EXISTS members_status_check;
ALTER TABLE members ADD CONSTRAINT members_status_check
    CHECK (membership_status IN ('active','member','inactive'));

ALTER TABLE members DROP CONSTRAINT IF EXISTS members_inactive_reason_check;
ALTER TABLE members ADD CONSTRAINT members_inactive_reason_check
    CHECK (membership_status <> 'inactive' OR exit_reason IS NOT NULL);

-- ---------------------------------------------------------------------------
-- 3) member_event_kinds: espelha o novo dominio e converte os tipos de saida
-- ---------------------------------------------------------------------------
-- Converte antes de trocar o CHECK: 'inactive' ja e aceito pelo dominio antigo.
UPDATE member_event_kinds SET sets_status = 'inactive'
WHERE sets_status IN ('dismissed','transferred','deceased','other');

-- Todo evento que leva o membro a inativo precisa declarar o motivo.
UPDATE member_event_kinds SET sets_exit_reason = CASE slug
        WHEN 'saida_transferencia'      THEN 'transferencia'
        WHEN 'saida_falecimento'        THEN 'falecimento'
        WHEN 'desligamento_pedido'      THEN 'desligamento'
        WHEN 'desligamento_disciplinar' THEN 'desligamento'
        WHEN 'abandono_atividades'      THEN 'ausencia'
        WHEN 'baixa_rol'                THEN 'desligamento'
        ELSE 'outro' END
WHERE sets_status = 'inactive' AND sets_exit_reason IS NULL;

ALTER TABLE member_event_kinds DROP CONSTRAINT IF EXISTS member_event_kinds_status_check;
ALTER TABLE member_event_kinds ADD CONSTRAINT member_event_kinds_status_check
    CHECK (sets_status IS NULL OR sets_status IN ('active','member','inactive'));

-- ---------------------------------------------------------------------------
-- 4) Seed dos tenants novos (create_tenant usa esta funcao)
-- ---------------------------------------------------------------------------
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
        ('Cadastro',                             'cadastro',                  'sistema',    'sky',    NULL,       NULL,            false, false, 'none',        0),
        ('Alteracao de situacao',                'status',                    'sistema',    'zinc',   NULL,       NULL,            false, false, 'none',        1),
        -- Batismo
        ('Batismo e profissao de fe',            'batismo_profissao_fe',      'batismo',    'green',  'active',   NULL,            false, true,  'none',       10),
        ('Batismo de crianca',                   'batismo_crianca',           'batismo',    'green',  NULL,       NULL,            false, true,  'none',       20),
        ('Apresentacao de crianca',              'apresentacao_crianca',      'batismo',    'sky',    NULL,       NULL,            false, false, 'none',       30),
        -- Profissao de fe
        ('Profissao de fe',                      'profissao_fe',              'profissao',  'green',  'active',   NULL,            false, false, 'none',       40),
        -- Recepcao
        ('Recebimento por transferencia',        'recebido_transferencia',    'recepcao',   'green',  'active',   NULL,            false, false, 'joined_at',  50),
        ('Recebimento por jurisdicao',           'recebido_jurisdicao',       'recepcao',   'green',  'active',   NULL,            false, false, 'joined_at',  60),
        -- Retorno
        ('Reativacao',                           'reativacao',                'retorno',    'green',  'active',   NULL,            true,  false, 'none',       70),
        -- Saida (sempre inativa + motivo)
        ('Saida por transferencia',              'saida_transferencia',       'saida',      'amber',  'inactive', 'transferencia', false, false, 'none',       80),
        ('Saida por falecimento',                'saida_falecimento',         'saida',      'zinc',   'inactive', 'falecimento',   false, false, 'none',       90),
        ('Desligamento a pedido do membro',      'desligamento_pedido',       'saida',      'red',    'inactive', 'desligamento',  false, false, 'none',      100),
        ('Desligamento por ato disciplinar',     'desligamento_disciplinar',  'saida',      'red',    'inactive', 'desligamento',  false, false, 'none',      110),
        ('Abandono das atividades (mais de 1 ano)','abandono_atividades',     'saida',      'red',    'inactive', 'ausencia',      false, false, 'none',      120),
        ('Baixa do Rol',                         'baixa_rol',                 'saida',      'red',    'inactive', 'desligamento',  false, false, 'none',      130),
        -- Ordenacao
        ('Ordenacao de diacono(a)',              'ordenacao_diacono',         'ordenacao',  'indigo', NULL,       NULL,            false, false, 'none',      140),
        ('Ordenacao de presbitero(a)',           'ordenacao_presbitero',      'ordenacao',  'indigo', NULL,       NULL,            false, false, 'none',      150),
        -- Disciplina
        ('Dissolucao das relacoes pastorais',    'dissolucao_pastoral',       'disciplina', 'amber',  'inactive', 'outro',         false, false, 'none',      160),
        ('Decisao do presbiterio',               'decisao_presbiterio',       'disciplina', 'amber',  'inactive', 'outro',         false, false, 'none',      170),
        -- Outros
        ('Outros',                               'outro',                     'outro',      'zinc',   NULL,       NULL,            false, false, 'none',      999)
    ) AS v(name, slug, category, tone, sets_status, sets_exit_reason,
           clears_exit, sets_baptism, sets_date_field, sort_order)
    ON CONFLICT DO NOTHING;
$$;

GRANT EXECUTE ON FUNCTION seed_member_event_kinds(uuid) TO chosenerp_app;
