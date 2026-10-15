-- 000101_audit_log_hash.down.sql
-- Restaura a definicao original da 000003 (re-encadear de volta nao e feito).
CREATE OR REPLACE FUNCTION audit_log_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.tenant_id::text || '|' || COALESCE(NEW.actor_id::text,'') || '|' ||
            NEW.action || '|' || NEW.entity || '|' || COALESCE(NEW.entity_id::text,'') || '|' ||
            COALESCE(NEW.payload::text,'') || '|' || NEW.created_at::text;
    NEW.prev_hash := (SELECT hash FROM audit_log WHERE id < NEW.id ORDER BY id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;
