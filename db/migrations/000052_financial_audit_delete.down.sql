-- 000052_financial_audit_delete.down.sql
-- Restaura o guard original (bloqueia qualquer DELETE em auditoria fechada,
-- inclusive o cascade).
CREATE OR REPLACE FUNCTION fin_audit_items_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    aid uuid;
    st  text;
BEGIN
    aid := COALESCE(NEW.audit_id, OLD.audit_id);
    SELECT status INTO st FROM financial_audits WHERE id = aid;
    IF st = 'fechada' THEN
        RAISE EXCEPTION 'auditoria fechada: sem manutenção nos itens';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;
