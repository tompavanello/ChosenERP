-- 000020_cargos.up.sql
-- Cargos (funcoes/ministerios) customizaveis pela igreja + vinculo do membro
-- com controle de mandato.
--
-- Atende o CAD100 do cliente:
--   1.3 Funcao/Ministerio - "o sistema devera permitir que um mesmo membro
--       possua MAIS DE UMA funcao/ministerio, quando aplicavel".
--   1.4 Controle de mandato - data de inicio, data de vencimento e situacao
--       (Ativo / Encerrado).
--
-- Antes disso existia apenas `members.office`, um texto livre com um unico
-- valor por membro (diacono, presbitero, evangelista...). A coluna e mantida
-- como legado (codigo antigo ainda a le) mas a fonte de verdade passa a ser
-- `member_cargos`; o backfill abaixo migra os valores existentes.

CREATE TABLE cargos (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- NULL = cargo global do tenant (catalogo da Sede, visivel a todas as filiais).
    branch_id     uuid REFERENCES branches(id) ON DELETE SET NULL,
    name          text NOT NULL,
    slug          text NOT NULL,
    kind          text NOT NULL DEFAULT 'outro', -- eclesiastico | lideranca | ensino | apoio | outro
    requires_term boolean NOT NULL DEFAULT false, -- exige data de vencimento de mandato
    is_active     boolean NOT NULL DEFAULT true,
    sort_order    int NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT cargos_kind_check CHECK (kind IN ('eclesiastico','lideranca','ensino','apoio','outro'))
);

-- Slug unico por tenant: e a chave estavel usada no backfill de members.office.
CREATE UNIQUE INDEX uq_cargos_tenant_slug ON cargos(tenant_id, slug);
-- Evita dois rotulos iguais ("Pastor" duplicado) poluindo os badges.
CREATE UNIQUE INDEX uq_cargos_tenant_name ON cargos(tenant_id, name);
CREATE INDEX idx_cargos_tenant ON cargos(tenant_id);
CREATE INDEX idx_cargos_branch ON cargos(branch_id);

CREATE TABLE member_cargos (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id  uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    cargo_id   uuid NOT NULL REFERENCES cargos(id) ON DELETE RESTRICT,
    started_at date,
    ends_at    date, -- vencimento do mandato
    status     text NOT NULL DEFAULT 'ativo', -- ativo | encerrado
    notes      text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT member_cargos_status_check CHECK (status IN ('ativo','encerrado')),
    CONSTRAINT member_cargos_period_check CHECK (ends_at IS NULL OR started_at IS NULL OR ends_at >= started_at)
);

-- Reeleicao e permitida (mesmo cargo em mandatos distintos, com datas
-- diferentes), mas nao duas vezes no mesmo mandato. NULL nao colide em UNIQUE,
-- por isso os indices parciais abaixo.
CREATE UNIQUE INDEX uq_member_cargos_dated
    ON member_cargos(member_id, cargo_id, started_at) WHERE started_at IS NOT NULL;
CREATE UNIQUE INDEX uq_member_cargos_open
    ON member_cargos(member_id, cargo_id) WHERE started_at IS NULL;

CREATE INDEX idx_member_cargos_member ON member_cargos(member_id);
CREATE INDEX idx_member_cargos_cargo ON member_cargos(cargo_id);

-- ---------------------------------------------------------------------------
-- RLS
-- ---------------------------------------------------------------------------
-- cargos tem tenant_id + branch_id: o teste generico
-- TestRLS_NoCrossTenantReadForAnyTable / NoCrossBranchReadForAnyTable varre
-- essas tabelas automaticamente, e TestRLS_AllTenantScopedTablesHaveRLS exige
-- relrowsecurity ligado (senao a suite falha).
ALTER TABLE cargos ENABLE ROW LEVEL SECURITY;

CREATE POLICY cargos_sel ON cargos
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY cargos_ins ON cargos
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, true));
CREATE POLICY cargos_upd ON cargos
  FOR UPDATE USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));
CREATE POLICY cargos_del ON cargos
  FOR DELETE USING (rls_write(tenant_id, branch_id, true));

-- member_cargos nao tem tenant_id proprio: herda o escopo do membro, igual a
-- member_relationships (politica rels_all da migracao 000016).
ALTER TABLE member_cargos ENABLE ROW LEVEL SECURITY;

CREATE POLICY member_cargos_sel ON member_cargos
  FOR SELECT USING (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND rls_read(m.tenant_id, m.branch_id, false)
  ));
-- O WITH CHECK valida TAMBEM o cargo: as checagens de FK rodam como dono da
-- tabela e nao passam por RLS, entao sem este EXISTS seria possivel vincular um
-- membro do tenant X a um cargo do tenant Y.
CREATE POLICY member_cargos_all ON member_cargos
  USING (EXISTS (SELECT 1 FROM members m WHERE m.id = member_id AND rls_write(m.tenant_id, m.branch_id, false)))
  WITH CHECK (
    EXISTS (SELECT 1 FROM members m WHERE m.id = member_id AND rls_write(m.tenant_id, m.branch_id, false))
    AND EXISTS (SELECT 1 FROM cargos c WHERE c.id = cargo_id AND rls_write(c.tenant_id, c.branch_id, true))
  );

-- ---------------------------------------------------------------------------
-- Catalogo inicial de cargos (requisito 1.3) para todo tenant existente.
-- Os slugs de diacono/presbitero/evangelista/pastor/missionario sao os mesmos
-- do antigo members.office, para o backfill encontrar a linha.
-- ---------------------------------------------------------------------------
INSERT INTO cargos (tenant_id, branch_id, name, slug, kind, requires_term, sort_order)
SELECT t.id, NULL, v.name, v.slug, v.kind, v.requires_term, v.sort_order
FROM tenants t
CROSS JOIN (VALUES
    ('Pastor',                'pastor',              'eclesiastico', true,  10),
    ('Presbitero',            'presbitero',          'eclesiastico', true,  20),
    ('Diacono',               'diacono',             'eclesiastico', true,  30),
    ('Evangelista',           'evangelista',         'eclesiastico', true,  40),
    ('Missionario',           'missionario',         'eclesiastico', false, 50),
    ('Lider de Jovens',       'lider_jovens',        'lideranca',    false, 60),
    ('Lider de Louvor',       'lider_louvor',        'lideranca',    false, 70),
    ('Lider de Pequeno Grupo','lider_pequeno_grupo', 'lideranca',    false, 80),
    ('Professor',             'professor',           'ensino',       false, 90),
    ('Outro',                 'outro',               'outro',        false, 999)
) AS v(name, slug, kind, requires_term, sort_order)
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Backfill: cada members.office existente vira um vinculo ativo, preservando
-- o que ja estava cadastrado.
-- ---------------------------------------------------------------------------
INSERT INTO member_cargos (member_id, cargo_id, status)
SELECT m.id, c.id, 'ativo'
FROM members m
JOIN cargos c ON c.tenant_id = m.tenant_id AND c.slug = m.office
WHERE m.office IS NOT NULL AND btrim(m.office) <> '';

-- ---------------------------------------------------------------------------
-- Carteirinha: no maximo UMA por membro.
-- Antes, cada clique em "Emitir" inseria uma linha nova com um qr_token novo, e
-- o numero exibido nao identificava ninguem. O indice unico abaixo e o que
-- torna a emissao idempotente (ON CONFLICT DO NOTHING em finance.IssueMembershipCard).
-- ---------------------------------------------------------------------------
-- Duplicatas pre-existentes: mantem a mais recente de cada membro. As entregas
-- pendentes sao reapontadas ANTES do DELETE - document_deliveries tem
-- ON DELETE CASCADE e o envio sumiria em silencio.
WITH keeper AS (
    SELECT DISTINCT ON (member_id) member_id, id
    FROM documents
    WHERE kind = 'membership_card' AND member_id IS NOT NULL
    ORDER BY member_id, created_at DESC, id DESC
), dup AS (
    SELECT d.id AS old_id, k.id AS new_id
    FROM documents d
    JOIN keeper k ON k.member_id = d.member_id
    WHERE d.kind = 'membership_card' AND d.id <> k.id
)
UPDATE document_deliveries dd SET document_id = dup.new_id
FROM dup WHERE dd.document_id = dup.old_id;

DELETE FROM documents d
USING documents k
WHERE d.kind = 'membership_card'
  AND k.kind = 'membership_card'
  AND d.member_id IS NOT NULL
  AND d.member_id = k.member_id
  AND (d.created_at, d.id) < (k.created_at, k.id);

CREATE UNIQUE INDEX uq_documents_membership_card_per_member
    ON documents(member_id)
    WHERE kind = 'membership_card' AND member_id IS NOT NULL;

-- qr_token e a capability do endpoint publico (ResolveCard usa WHERE qr_token = $1):
-- um token duplicado resolveria a carteirinha de OUTRA pessoa.
UPDATE documents SET qr_token = encode(gen_random_bytes(16), 'hex')
WHERE qr_token IS NOT NULL
  AND id NOT IN (
      SELECT DISTINCT ON (qr_token) id FROM documents
      WHERE qr_token IS NOT NULL
      ORDER BY qr_token, created_at, id
  );

CREATE UNIQUE INDEX uq_documents_qr_token
    ON documents(qr_token)
    WHERE qr_token IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Codigo da familia (#001, #002...), como na planilha do cliente
-- ("Familia AMARAL #001"). A coluna e text com zero a esquerda porque e
-- rotulo de exibicao, nao numero de ordenacao.
-- ---------------------------------------------------------------------------
ALTER TABLE families ADD COLUMN code text;

-- Numera as familias ja existentes por tenant, em ordem de criacao.
UPDATE families f SET code = n.code
FROM (
    SELECT id, lpad((row_number() OVER (PARTITION BY tenant_id ORDER BY created_at, id))::text, 3, '0') AS code
    FROM families
) n WHERE n.id = f.id;

-- Duas familias "001" no mesmo tenant tornariam o codigo inutil como
-- identificador; o indice e por tenant, entao tenants distintos nao colidem.
CREATE UNIQUE INDEX uq_families_tenant_code ON families(tenant_id, code) WHERE code IS NOT NULL;
