-- 000066_member_status_simplify.down.sql
-- Reverte o dominio para os 7 valores de 000022. Em best-effort:
--   * a situacao volta a aceitar os 7 valores, mas os membros hoje 'inactive'
--     NAO sao redistribuidos (a informacao de qual valor legado eram se perdeu
--     ao colapsar) - ficam todos como 'inactive';
--   * os tipos de saida voltam a gravar 'dismissed' (nao ha como reconstruir
--     transferred/deceased por tipo).
ALTER TABLE members DROP CONSTRAINT IF EXISTS members_inactive_reason_check;

ALTER TABLE members DROP CONSTRAINT IF EXISTS members_status_check;
ALTER TABLE members ADD CONSTRAINT members_status_check
    CHECK (membership_status IN
        ('active','member','inactive','dismissed','transferred','deceased','other'));

ALTER TABLE member_event_kinds DROP CONSTRAINT IF EXISTS member_event_kinds_status_check;
ALTER TABLE member_event_kinds ADD CONSTRAINT member_event_kinds_status_check
    CHECK (sets_status IS NULL OR sets_status IN
        ('active','member','inactive','dismissed','transferred','deceased','other'));

UPDATE member_event_kinds SET sets_status = 'dismissed'
WHERE sets_status = 'inactive' AND sets_exit_reason <> 'transferencia'
  AND sets_exit_reason <> 'falecimento';
UPDATE member_event_kinds SET sets_status = 'transferred'
WHERE sets_status = 'inactive' AND sets_exit_reason = 'transferencia';
UPDATE member_event_kinds SET sets_status = 'deceased'
WHERE sets_status = 'inactive' AND sets_exit_reason = 'falecimento';

-- Restaura o seed da versao 000065 (7 valores).
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
        ('Cadastro',                             'cadastro',                  'sistema',    'sky',    NULL,          NULL,           false, false, 'none',        0),
        ('Alteracao de situacao',                'status',                    'sistema',    'zinc',   NULL,          NULL,           false, false, 'none',        1),
        ('Batismo e profissao de fe',            'batismo_profissao_fe',      'batismo',    'green',  'active',      NULL,           false, true,  'none',       10),
        ('Batismo de crianca',                   'batismo_crianca',           'batismo',    'green',  NULL,          NULL,           false, true,  'none',       20),
        ('Apresentacao de crianca',              'apresentacao_crianca',      'batismo',    'sky',    NULL,          NULL,           false, false, 'none',       30),
        ('Profissao de fe',                      'profissao_fe',              'profissao',  'green',  'active',      NULL,           false, false, 'none',       40),
        ('Recebimento por transferencia',        'recebido_transferencia',    'recepcao',   'green',  'active',      NULL,           false, false, 'joined_at',  50),
        ('Recebimento por jurisdicao',           'recebido_jurisdicao',       'recepcao',   'green',  'active',      NULL,           false, false, 'joined_at',  60),
        ('Reativacao',                           'reativacao',                'retorno',    'green',  'active',      NULL,           true,  false, 'none',       70),
        ('Saida por transferencia',              'saida_transferencia',       'saida',      'amber',  'transferred', 'transferencia',false, false, 'none',       80),
        ('Saida por falecimento',                'saida_falecimento',         'saida',      'zinc',   'deceased',    'falecimento',  false, false, 'none',       90),
        ('Desligamento a pedido do membro',      'desligamento_pedido',       'saida',      'red',    'dismissed',   'desligamento', false, false, 'none',      100),
        ('Desligamento por ato disciplinar',     'desligamento_disciplinar',  'saida',      'red',    'dismissed',   'desligamento', false, false, 'none',      110),
        ('Abandono das atividades (mais de 1 ano)','abandono_atividades',     'saida',      'red',    'dismissed',   'ausencia',     false, false, 'none',      120),
        ('Baixa do Rol',                         'baixa_rol',                 'saida',      'red',    'dismissed',   'desligamento', false, false, 'none',      130),
        ('Ordenacao de diacono(a)',              'ordenacao_diacono',         'ordenacao',  'indigo', NULL,          NULL,           false, false, 'none',      140),
        ('Ordenacao de presbitero(a)',           'ordenacao_presbitero',      'ordenacao',  'indigo', NULL,          NULL,           false, false, 'none',      150),
        ('Dissolucao das relacoes pastorais',    'dissolucao_pastoral',       'disciplina', 'amber',  'other',       NULL,           false, false, 'none',      160),
        ('Decisao do presbiterio',               'decisao_presbiterio',       'disciplina', 'amber',  'other',       NULL,           false, false, 'none',      170),
        ('Outros',                               'outro',                     'outro',      'zinc',   NULL,          NULL,           false, false, 'none',      999)
    ) AS v(name, slug, category, tone, sets_status, sets_exit_reason,
           clears_exit, sets_baptism, sets_date_field, sort_order)
    ON CONFLICT DO NOTHING;
$$;

GRANT EXECUTE ON FUNCTION seed_member_event_kinds(uuid) TO chosenerp_app;
