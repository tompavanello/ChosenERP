-- 000052_financial_audit_delete.up.sql
-- Permite EXCLUIR uma auditoria (ex.: criada indevidamente). O guard de itens
-- bloqueava o DELETE em cascata de uma auditoria fechada; passa a liberar
-- quando a exclusao vem do cascade do pai (pg_trigger_depth() > 1), no mesmo
-- padrao de minute_signatures. A protecao contra "manutencao" segue valendo
-- para INSERT/UPDATE/DELETE diretos nos itens.
CREATE OR REPLACE FUNCTION fin_audit_items_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    aid uuid;
    st  text;
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN COALESCE(NEW, OLD);
    END IF;
    aid := COALESCE(NEW.audit_id, OLD.audit_id);
    SELECT status INTO st FROM financial_audits WHERE id = aid;
    IF st = 'fechada' THEN
        RAISE EXCEPTION 'auditoria fechada: sem manutencao nos itens';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;
