-- 000022_member_lifecycle.up.sql
-- Ciclo de vida do membro (requisitos 1.1/1.6/1.7/1.8 do cliente):
--   * 1.1 endereço POR MEMBRO (decisão: o endereço é da pessoa, não só da família);
--   * 1.6 situação do membro com domínio fechado por CHECK;
--   * 1.7 motivo da baixa + data de saída;
--   * 1.8 histórico eclesiástico append-only com hash-chain.
--
-- Classificação no Rol (1.2): NÃO vira coluna. O cliente define Ativo = membro
-- professo e não existe "professo inativo", então a professorate é DERIVADA da
-- situação (`membership_status = 'active'`), evitando duas fontes de verdade.

-- ---------------------------------------------------------------------------
-- 1.1 Endereço do membro
-- ---------------------------------------------------------------------------
ALTER TABLE members ADD COLUMN address jsonb;

-- ---------------------------------------------------------------------------
-- 1.6 Situação do membro — fecha o domínio (antes era texto livre)
-- ---------------------------------------------------------------------------
-- Valores normalizados. `member` é mantido como "não professo (ainda ativo na
-- igreja)" — a planilha do cliente distingue NÃO PROFESSO de INATIVO; usar
-- `other` para qualquer valor legado desconhecido.
UPDATE members SET membership_status = 'other'
WHERE membership_status IS NULL
   OR membership_status NOT IN
      ('active','member','inactive','dismissed','transferred','deceased','other');

ALTER TABLE members
    ADD CONSTRAINT members_status_check
    CHECK (membership_status IN
        ('active','member','inactive','dismissed','transferred','deceased','other'));

-- Índice para os relatórios do Rol (contagem por situação/classificação).
CREATE INDEX IF NOT EXISTS idx_members_tenant_status
    ON members(tenant_id, branch_id, membership_status);

-- ---------------------------------------------------------------------------
-- 1.7 Motivo da baixa + data de saída
-- ---------------------------------------------------------------------------
ALTER TABLE members ADD COLUMN exit_reason text;
ALTER TABLE members ADD COLUMN exited_at date;

ALTER TABLE members
    ADD CONSTRAINT members_exit_reason_check
    CHECK (exit_reason IS NULL OR exit_reason IN
        ('falecimento','desligamento','transferencia','abandono','ausencia','outro'));

-- ---------------------------------------------------------------------------
-- 1.8 Histórico eclesiástico (append-only + hash-chain)
-- ---------------------------------------------------------------------------
CREATE TABLE member_history (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    member_id   uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    occurred_at timestamptz NOT NULL DEFAULT now(), -- data/hora do evento (pode ser retroativa)
    kind        text NOT NULL,                      -- batismo_infantil | profissao_fe | ...
    notes       text,
    created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    prev_hash   text,
    hash        text NOT NULL DEFAULT ''
);

CREATE INDEX idx_member_history_member ON member_history(member_id, created_at DESC);
CREATE INDEX idx_member_history_tenant ON member_history(tenant_id, branch_id);

-- Imutabilidade: sem UPDATE/DELETE (mesmo padrão de financial_transactions).
-- A exceção é a exclusão em CASCADE quando o membro é apagado (FK abaixo):
-- nesse caso o PostgreSQL dispara o trigger com pg_trigger_depth() > 1, e a
-- linha some junto com o membro em vez de travar a operação.
CREATE OR REPLACE FUNCTION member_history_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'member_history is append-only (no UPDATE/DELETE)';
END;
$$;

CREATE TRIGGER member_history_no_update
BEFORE UPDATE OR DELETE ON member_history
FOR EACH ROW EXECUTE FUNCTION member_history_guard();

-- Encadeia o hash cronológico dentro do tenant, por created_at + id (o mesmo
-- cuidado do fin_tx_hash: UUID puro produz cadeia não-temporal).
CREATE OR REPLACE FUNCTION member_history_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.member_id::text || '|' || NEW.kind || '|' ||
            COALESCE(NEW.notes,'') || '|' || COALESCE(NEW.created_by::text,'') || '|' ||
            NEW.occurred_at::text || '|' || NEW.created_at::text;
    NEW.prev_hash := (SELECT hash FROM member_history
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

CREATE TRIGGER member_history_hash
BEFORE INSERT ON member_history
FOR EACH ROW EXECUTE FUNCTION member_history_hash();

-- RLS: leitura/gravação pelo escopo da filial (como members). Sem UPDATE/DELETE
-- de propósito — a tabela é append-only.
ALTER TABLE member_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY member_history_sel ON member_history
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY member_history_ins ON member_history
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
