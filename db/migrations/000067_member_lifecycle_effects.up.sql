-- 000067_member_lifecycle_effects.up.sql
-- Torna o estado da "vida eclesiastica" DERIVADO do historico e simplifica o
-- modelo de efeitos do catalogo (member_event_kinds, 000065).
--
-- Antes cada tipo declarava ate 5 efeitos independentes e faceis de combinar de
-- forma contraditoria: sets_status, sets_exit_reason, clears_exit, sets_baptism
-- e sets_date_field. Agora sao apenas 3, com significado unico:
--
--   * sets_status      -> situacao resultante (NULL = nao mexe);
--   * sets_exit_reason -> motivo (apenas quando sets_status = 'inactive');
--   * sets_date_field  -> data registrada: none | baptism | joined_at | marriage_date.
--
-- Dois efeitos eram redundantes:
--   * `clears_exit`  - o motor (internal/members/history.go) ja limpa o motivo/data
--     de saida sempre que a situacao resultante e ativo/membro; logo um evento de
--     reativacao so precisa de sets_status = 'active';
--   * `sets_baptism` - virou apenas sets_date_field = 'baptism' (que tambem grava
--     o local do batismo, se informado).
--
-- Os campos do membro (membership_status, exit_reason, exited_at, baptism_date,
-- baptism_location, joined_at, marriage_date) continuam existindo como CACHE da
-- ultima transicao; o cadastro nao os edita mais - so o historico.

-- ---------------------------------------------------------------------------
-- 1) Amplia o dominio de sets_date_field com 'baptism'
-- ---------------------------------------------------------------------------
ALTER TABLE member_event_kinds DROP CONSTRAINT IF EXISTS member_event_kinds_date_check;
ALTER TABLE member_event_kinds ADD CONSTRAINT member_event_kinds_date_check
    CHECK (sets_date_field IN ('none','baptism','joined_at','marriage_date'));

-- ---------------------------------------------------------------------------
-- 2) Preserva o comportamento antes de remover as colunas redundantes
-- ---------------------------------------------------------------------------
-- Quem gravava a data do batismo passa a declarar sets_date_field = 'baptism'.
UPDATE member_event_kinds SET sets_date_field = 'baptism'
WHERE sets_baptism = true AND sets_date_field = 'none';

-- Reativacao sem situacao de destino (configuracao customizada) vira 'active'.
UPDATE member_event_kinds SET sets_status = 'active'
WHERE clears_exit = true AND sets_status IS NULL;

ALTER TABLE member_event_kinds DROP COLUMN IF EXISTS clears_exit;
ALTER TABLE member_event_kinds DROP COLUMN IF EXISTS sets_baptism;

-- ---------------------------------------------------------------------------
-- 3) Seed revisado (create_tenant e tenants novos)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION seed_member_event_kinds(p_tenant uuid)
RETURNS void
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    INSERT INTO member_event_kinds
        (tenant_id, branch_id, name, slug, category, tone, sets_status,
         sets_exit_reason, sets_date_field, sort_order)
    SELECT p_tenant, NULL, v.name, v.slug, v.category, v.tone, v.sets_status,
           v.sets_exit_reason, v.sets_date_field, v.sort_order
    FROM (VALUES
        -- Sistema (automaticos; nao aparecem como opcao manual)
        ('Cadastro',                             'cadastro',                  'sistema',    'sky',    NULL,       NULL,            'none',        0),
        ('Alteracao de situacao',                'status',                    'sistema',    'zinc',   NULL,       NULL,            'none',        1),
        -- Batismo
        ('Batismo e profissao de fe',            'batismo_profissao_fe',      'batismo',    'green',  'active',   NULL,            'baptism',    10),
        ('Batismo de crianca',                   'batismo_crianca',           'batismo',    'green',  NULL,       NULL,            'baptism',    20),
        ('Apresentacao de crianca',              'apresentacao_crianca',      'batismo',    'sky',    NULL,       NULL,            'none',       30),
        -- Profissao de fe
        ('Profissao de fe',                      'profissao_fe',              'profissao',  'green',  'active',   NULL,            'none',       40),
        -- Recepcao
        ('Recebimento por transferencia',        'recebido_transferencia',    'recepcao',   'green',  'active',   NULL,            'joined_at',  50),
        ('Recebimento por jurisdicao',           'recebido_jurisdicao',       'recepcao',   'green',  'active',   NULL,            'joined_at',  60),
        -- Retorno
        ('Reativacao',                           'reativacao',                'retorno',    'green',  'active',   NULL,            'none',       70),
        -- Saida (sempre inativa + motivo)
        ('Saida por transferencia',              'saida_transferencia',       'saida',      'amber',  'inactive', 'transferencia', 'none',       80),
        ('Saida por falecimento',                'saida_falecimento',         'saida',      'zinc',   'inactive', 'falecimento',   'none',       90),
        ('Desligamento a pedido do membro',      'desligamento_pedido',       'saida',      'red',    'inactive', 'desligamento',  'none',      100),
        ('Desligamento por ato disciplinar',     'desligamento_disciplinar',  'saida',      'red',    'inactive', 'desligamento',  'none',      110),
        ('Abandono das atividades (mais de 1 ano)','abandono_atividades',     'saida',      'red',    'inactive', 'ausencia',      'none',      120),
        ('Baixa do Rol',                         'baixa_rol',                 'saida',      'red',    'inactive', 'desligamento',  'none',      130),
        -- Ordenacao
        ('Ordenacao de diacono(a)',              'ordenacao_diacono',         'ordenacao',  'indigo', NULL,       NULL,            'none',      140),
        ('Ordenacao de presbitero(a)',           'ordenacao_presbitero',      'ordenacao',  'indigo', NULL,       NULL,            'none',      150),
        -- Disciplina
        ('Dissolucao das relacoes pastorais',    'dissolucao_pastoral',       'disciplina', 'amber',  'inactive', 'outro',         'none',      160),
        ('Decisao do presbiterio',               'decisao_presbiterio',       'disciplina', 'amber',  'inactive', 'outro',         'none',      170),
        -- Outros
        ('Outros',                               'outro',                     'outro',      'zinc',   NULL,       NULL,            'none',      999)
    ) AS v(name, slug, category, tone, sets_status, sets_exit_reason,
           sets_date_field, sort_order)
    ON CONFLICT DO NOTHING;
$$;

GRANT EXECUTE ON FUNCTION seed_member_event_kinds(uuid) TO chosenerp_app;
