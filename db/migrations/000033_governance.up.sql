-- 000033_governance.up.sql
-- Etapa 7 - Governanca (PRD Modulo 6 / itens G1-G7 do plano):
--   * G1 livro de atas digital (minutes);
--   * G2 votacao eletronica com quorum obrigatorio e voto secreto
--        (votes + vote_options + vote_registrations + vote_ballots);
--   * G3 apuracao automatica (result_summary jsonb em votes; a ata e atualizada
--        pelo servico ao encerrar);
--   * G4 assinatura eletronica interna da ata (minute_signatures);
--   * G6 convenios e documentacao legal com alerta de vencimento
--        (legal_documents);
--   * G7 trilha imutavel: minute_signatures e vote_ballots sao append-only com
--        hash-chain (mesmo padrao de financial_transactions/member_history).
--
-- Decisoes do cliente (9 do plano): quorum obrigatorio, voto secreto e
-- assinatura interna (nao ICP-Brasil nesta fase).
--
-- Sobre o voto secreto: a PARTICIPACAO (quem votou, para conferir o quorum) fica
-- em `vote_registrations`; a ESCOLHA fica em `vote_ballots`, que NAO guarda
-- vinculo com o eleitor. Sem a correlacao por linha, a apuracao so devolve
-- contagens por opcao.

-- ---------------------------------------------------------------------------
-- G1 - Livro de atas
-- ---------------------------------------------------------------------------
CREATE TABLE minutes (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    title        text NOT NULL,
    meeting_at   timestamptz NOT NULL,
    kind         text NOT NULL DEFAULT 'assembleia',
    -- Pauta e deliberacoes em texto livre (markdown simples aceito na UI).
    body         text,
    -- rascunho -> aprovada -> assinada. Assinar congela a ata (nao edita mais).
    status       text NOT NULL DEFAULT 'rascunho',
    -- Quorum minimo de presentes declarado na ata (0 = nao exigido).
    quorum_required int NOT NULL DEFAULT 0 CHECK (quorum_required >= 0),
    attendance_count int NOT NULL DEFAULT 0 CHECK (attendance_count >= 0),
    created_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT minutes_kind_check CHECK (kind IN
        ('assembleia','ordinaria','extraordinaria','diretoria','reuniao','outra')),
    CONSTRAINT minutes_status_check CHECK (status IN
        ('rascunho','aprovada','assinada','cancelada'))
);

CREATE INDEX idx_minutes_tenant ON minutes(tenant_id, branch_id, meeting_at DESC);

ALTER TABLE minutes ENABLE ROW LEVEL SECURITY;
CREATE POLICY minutes_sel ON minutes
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY minutes_ins ON minutes
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY minutes_upd ON minutes
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY minutes_del ON minutes
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- G4/G7 - Assinatura eletronica interna (append-only + hash-chain)
-- ---------------------------------------------------------------------------
CREATE TABLE minute_signatures (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    minute_id     uuid NOT NULL REFERENCES minutes(id) ON DELETE CASCADE,
    user_id       uuid REFERENCES users(id) ON DELETE SET NULL,
    -- Snapshot do assinante: o nome/funcao no momento da assinatura nao muda
    -- depois, mesmo que o cadastro do usuario mude.
    signer_name   text NOT NULL,
    signer_role   text,
    -- Hash do conteudo da ata no momento da assinatura (a ata nao pode ser
    -- alterada depois sem invalidar a assinatura).
    document_hash text NOT NULL,
    signed_at     timestamptz NOT NULL DEFAULT now(),
    created_at    timestamptz NOT NULL DEFAULT now(),
    prev_hash     text,
    hash          text NOT NULL DEFAULT ''
);

CREATE INDEX idx_minute_signatures_minute ON minute_signatures(minute_id, signed_at);
CREATE INDEX idx_minute_signatures_tenant ON minute_signatures(tenant_id, branch_id);

-- Imutabilidade (o guard libera DELETE em cascata quando a ata e apagada).
CREATE OR REPLACE FUNCTION minute_signatures_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'minute_signatures is append-only (no UPDATE/DELETE)';
END;
$$;

CREATE TRIGGER minute_signatures_no_update
BEFORE UPDATE OR DELETE ON minute_signatures
FOR EACH ROW EXECUTE FUNCTION minute_signatures_guard();

-- Encadeia o hash por tenant, na ordem (created_at, id) - como member_history.
CREATE OR REPLACE FUNCTION minute_signatures_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.minute_id::text || '|' || COALESCE(NEW.user_id::text,'') || '|' ||
            NEW.signer_name || '|' || COALESCE(NEW.signer_role,'') || '|' ||
            NEW.document_hash || '|' || NEW.signed_at::text || '|' || NEW.created_at::text;
    NEW.prev_hash := (SELECT hash FROM minute_signatures
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

CREATE TRIGGER minute_signatures_hash
BEFORE INSERT ON minute_signatures
FOR EACH ROW EXECUTE FUNCTION minute_signatures_hash();

ALTER TABLE minute_signatures ENABLE ROW LEVEL SECURITY;
CREATE POLICY minute_signatures_sel ON minute_signatures
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY minute_signatures_ins ON minute_signatures
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- G2 - Votacao eletronica
-- ---------------------------------------------------------------------------
CREATE TABLE votes (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id        uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    -- Ata a qual a votacao pertence (o resultado entra nela ao encerrar).
    minute_id        uuid REFERENCES minutes(id) ON DELETE SET NULL,
    title            text NOT NULL,
    description      text,
    kind             text NOT NULL DEFAULT 'assembleia',
    -- Voto secreto: a escolha nunca e ligada ao eleitor (decisao do cliente).
    secret           boolean NOT NULL DEFAULT true,
    -- Quorum obrigatorio: minimo de participantes para a votacao valer.
    quorum_required  int NOT NULL DEFAULT 0 CHECK (quorum_required >= 0),
    min_attendance   int NOT NULL DEFAULT 0 CHECK (min_attendance >= 0),
    opens_at         timestamptz,
    closes_at        timestamptz,
    status           text NOT NULL DEFAULT 'rascunho',
    -- Apuracao gravada ao encerrar (opcao -> votos, total, quorum atingido).
    result_summary   jsonb,
    created_by       uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT votes_kind_check CHECK (kind IN
        ('assembleia','diretoria','orcamento','mocao','eleicao','outra')),
    CONSTRAINT votes_status_check CHECK (status IN
        ('rascunho','aberta','encerrada','cancelada'))
);

CREATE INDEX idx_votes_tenant ON votes(tenant_id, branch_id, created_at DESC);
CREATE INDEX idx_votes_minute ON votes(minute_id);

ALTER TABLE votes ENABLE ROW LEVEL SECURITY;
CREATE POLICY votes_sel ON votes
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY votes_ins ON votes
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY votes_upd ON votes
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY votes_del ON votes
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

CREATE TABLE vote_options (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    vote_id    uuid NOT NULL REFERENCES votes(id) ON DELETE CASCADE,
    label      text NOT NULL,
    sort_order int NOT NULL DEFAULT 0,
    UNIQUE (vote_id, label)
);

CREATE INDEX idx_vote_options_vote ON vote_options(vote_id, sort_order);

ALTER TABLE vote_options ENABLE ROW LEVEL SECURITY;
CREATE POLICY vote_options_sel ON vote_options
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY vote_options_all ON vote_options
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- Quem participou (para o quorum). NAO guarda a opcao escolhida.
CREATE TABLE vote_registrations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    vote_id    uuid NOT NULL REFERENCES votes(id) ON DELETE CASCADE,
    voter_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    voter_name text,
    voted_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (vote_id, voter_id)
);

CREATE INDEX idx_vote_registrations_vote ON vote_registrations(vote_id);

ALTER TABLE vote_registrations ENABLE ROW LEVEL SECURITY;
CREATE POLICY vote_registrations_sel ON vote_registrations
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY vote_registrations_all ON vote_registrations
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- A escolha, SEM vinculo com o eleitor (voto secreto). Append-only + hash-chain.
CREATE TABLE vote_ballots (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    vote_id    uuid NOT NULL REFERENCES votes(id) ON DELETE CASCADE,
    option_id  uuid NOT NULL REFERENCES vote_options(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    prev_hash  text,
    hash       text NOT NULL DEFAULT ''
);

CREATE INDEX idx_vote_ballots_vote ON vote_ballots(vote_id);

CREATE OR REPLACE FUNCTION vote_ballots_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'vote_ballots is append-only (no UPDATE/DELETE)';
END;
$$;

CREATE TRIGGER vote_ballots_no_update
BEFORE UPDATE OR DELETE ON vote_ballots
FOR EACH ROW EXECUTE FUNCTION vote_ballots_guard();

CREATE OR REPLACE FUNCTION vote_ballots_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.vote_id::text || '|' || NEW.option_id::text || '|' || NEW.created_at::text;
    NEW.prev_hash := (SELECT hash FROM vote_ballots
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

CREATE TRIGGER vote_ballots_hash
BEFORE INSERT ON vote_ballots
FOR EACH ROW EXECUTE FUNCTION vote_ballots_hash();

ALTER TABLE vote_ballots ENABLE ROW LEVEL SECURITY;
CREATE POLICY vote_ballots_sel ON vote_ballots
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY vote_ballots_ins ON vote_ballots
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- G6 - Convenios e documentacao legal (com alerta de vencimento)
-- ---------------------------------------------------------------------------
CREATE TABLE legal_documents (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    kind        text NOT NULL DEFAULT 'convenio',
    title       text NOT NULL,
    description text,
    -- No do documento/processo (referencia externa).
    reference   text,
    issued_at   date,
    expires_at  date,
    -- Caminho do arquivo digitalizado (upload reaproveita o storage local).
    file_url    text,
    created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT legal_documents_kind_check CHECK (kind IN
        ('escritura','alvara','contrato','seguro','convenio','certidao','outro')),
    CONSTRAINT legal_documents_period_check CHECK (expires_at IS NULL OR issued_at IS NULL OR expires_at >= issued_at)
);

CREATE INDEX idx_legal_documents_tenant ON legal_documents(tenant_id, branch_id, expires_at);

ALTER TABLE legal_documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY legal_documents_sel ON legal_documents
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY legal_documents_ins ON legal_documents
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY legal_documents_upd ON legal_documents
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY legal_documents_del ON legal_documents
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));
