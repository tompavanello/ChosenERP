-- 000101_audit_log_hash.up.sql
-- Corrige o encadeamento de hash do `audit_log`: a funcao original (000003)
-- usava `WHERE id < NEW.id ORDER BY id DESC`, ou seja, ordenava por VALOR de UUID
-- (aleatorio/nao-temporal) e NAO filtrava por tenant. E o mesmo defeito que as
-- migracoes 000018/000027 corrigiram em financial_transactions e a 000022 em
-- member_history, mas que nunca foi aplicado ao audit_log.
--
-- Nova regra: cadeia por tenant, ordenada por (created_at, id) - temporal e
-- isolada por igreja. As linhas existentes sao re-encadeadas (desabilitando o
-- guard append-only durante a migracao).

CREATE OR REPLACE FUNCTION audit_log_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.tenant_id::text || '|' || COALESCE(NEW.actor_id::text,'') || '|' ||
            NEW.action || '|' || NEW.entity || '|' || COALESCE(NEW.entity_id::text,'') || '|' ||
            COALESCE(NEW.payload::text,'') || '|' || NEW.created_at::text;
    NEW.prev_hash := (SELECT hash FROM audit_log
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

-- Re-encadeia as linhas existentes (o guard bloqueia UPDATE por padrao).
ALTER TABLE audit_log DISABLE TRIGGER audit_log_no_update;

DO $$
DECLARE
    r          record;
    cur_tenant uuid := NULL;
    prev       text := NULL;
    body       text;
    h          text;
BEGIN
    FOR r IN
        SELECT id, tenant_id, actor_id, action, entity, entity_id, payload, created_at
        FROM audit_log
        ORDER BY tenant_id, created_at, id
    LOOP
        IF cur_tenant IS DISTINCT FROM r.tenant_id THEN
            cur_tenant := r.tenant_id;
            prev := NULL;
        END IF;
        body := r.tenant_id::text || '|' || COALESCE(r.actor_id::text,'') || '|' ||
                r.action || '|' || r.entity || '|' || COALESCE(r.entity_id::text,'') || '|' ||
                COALESCE(r.payload::text,'') || '|' || r.created_at::text;
        h := encode(digest(body || '|' || COALESCE(prev,''), 'sha256'), 'hex');
        UPDATE audit_log SET prev_hash = prev, hash = h WHERE id = r.id;
        prev := h;
    END LOOP;
END $$;

ALTER TABLE audit_log ENABLE TRIGGER audit_log_no_update;
