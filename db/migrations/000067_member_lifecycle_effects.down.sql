-- 000067_member_lifecycle_effects.down.sql
-- Reverte a simplificacao dos efeitos, recriando as colunas redundantes. Em
-- best-effort:
--   * sets_date_field = 'baptism' volta a sets_baptism = true + 'none';
--   * a coluna clears_exit volta com false (nao ha como saber quais reativacoes
--     dependiam dela - hoje sets_status='active' ja cobre o efeito).

ALTER TABLE member_event_kinds ADD COLUMN IF NOT EXISTS clears_exit boolean NOT NULL DEFAULT false;
ALTER TABLE member_event_kinds ADD COLUMN IF NOT EXISTS sets_baptism boolean NOT NULL DEFAULT false;

UPDATE member_event_kinds SET sets_baptism = true, sets_date_field = 'none'
WHERE sets_date_field = 'baptism';

ALTER TABLE member_event_kinds DROP CONSTRAINT IF EXISTS member_event_kinds_date_check;
ALTER TABLE member_event_kinds ADD CONSTRAINT member_event_kinds_date_check
    CHECK (sets_date_field IN ('none','joined_at','marriage_date'));

-- Restaura o seed da versao 000066.
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
        ('Cadastro',                             'cadastro',                  'sistema',    'sky',    NULL,       NULL,            false, false, 'none',        0),
        ('Alteracao de situacao',                'status',                    'sistema',    'zinc',   NULL,       NULL,            false, false, 'none',        1),
        ('Batismo e profissao de fe',            'batismo_profissao_fe',      'batismo',    'green',  'active',   NULL,            false, true,  'none',       10),
        ('Batismo de crianca',                   'batismo_crianca',           'batismo',    'green',  NULL,       NULL,            false, true,  'none',       20),
        ('Apresentacao de crianca',              'apresentacao_crianca',      'batismo',    'sky',    NULL,       NULL,            false, false, 'none',       30),
        ('Profissao de fe',                      'profissao_fe',              'profissao',  'green',  'active',   NULL,            false, false, 'none',       40),
        ('Recebimento por transferencia',        'recebido_transferencia',    'recepcao',   'green',  'active',   NULL,            false, false, 'joined_at',  50),
        ('Recebimento por jurisdicao',           'recebido_jurisdicao',       'recepcao',   'green',  'active',   NULL,            false, false, 'joined_at',  60),
        ('Reativacao',                           'reativacao',                'retorno',    'green',  'active',   NULL,            true,  false, 'none',       70),
        ('Saida por transferencia',              'saida_transferencia',       'saida',      'amber',  'inactive', 'transferencia', false, false, 'none',       80),
        ('Saida por falecimento',                'saida_falecimento',         'saida',      'zinc',   'inactive', 'falecimento',   false, false, 'none',       90),
        ('Desligamento a pedido do membro',      'desligamento_pedido',       'saida',      'red',    'inactive', 'desligamento',  false, false, 'none',      100),
        ('Desligamento por ato disciplinar',     'desligamento_disciplinar',  'saida',      'red',    'inactive', 'desligamento',  false, false, 'none',      110),
        ('Abandono das atividades (mais de 1 ano)','abandono_atividades',     'saida',      'red',    'inactive', 'ausencia',      false, false, 'none',      120),
        ('Baixa do Rol',                         'baixa_rol',                 'saida',      'red',    'inactive', 'desligamento',  false, false, 'none',      130),
        ('Ordenacao de diacono(a)',              'ordenacao_diacono',         'ordenacao',  'indigo', NULL,       NULL,            false, false, 'none',      140),
        ('Ordenacao de presbitero(a)',           'ordenacao_presbitero',      'ordenacao',  'indigo', NULL,       NULL,            false, false, 'none',      150),
        ('Dissolucao das relacoes pastorais',    'dissolucao_pastoral',       'disciplina', 'amber',  'inactive', 'outro',         false, false, 'none',      160),
        ('Decisao do presbiterio',               'decisao_presbiterio',       'disciplina', 'amber',  'inactive', 'outro',         false, false, 'none',      170),
        ('Outros',                               'outro',                     'outro',      'zinc',   NULL,       NULL,            false, false, 'none',      999)
    ) AS v(name, slug, category, tone, sets_status, sets_exit_reason,
           clears_exit, sets_baptism, sets_date_field, sort_order)
    ON CONFLICT DO NOTHING;
$$;

GRANT EXECUTE ON FUNCTION seed_member_event_kinds(uuid) TO chosenerp_app;
