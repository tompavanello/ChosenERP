-- 000063_financial_audit_lock.up.sql
-- Trava de AUDITORIA: nao permitir duas auditorias com periodos sobrepostos na
-- mesma filial. Vale para qualquer status (aberta ou fechada), impedindo abrir
-- uma nova auditoria para um periodo ja auditado/selecionado.
CREATE OR REPLACE FUNCTION fin_audits_period_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_id     uuid := COALESCE(NEW.id, OLD.id);
    v_tenant uuid := COALESCE(NEW.tenant_id, OLD.tenant_id);
    v_branch uuid := COALESCE(NEW.branch_id, OLD.branch_id);
    v_start  date := COALESCE(NEW.period_start, OLD.period_start);
    v_end    date := COALESCE(NEW.period_end, OLD.period_end);
    v_count  int;
BEGIN
    SELECT count(*) INTO v_count
    FROM financial_audits a
    WHERE a.tenant_id = v_tenant
      AND a.branch_id = v_branch
      AND a.id <> v_id
      AND daterange(a.period_start, a.period_end, '[]')
          && daterange(v_start, v_end, '[]');

    IF v_count > 0 THEN
        RAISE EXCEPTION 'ja existe auditoria para periodo sobreposto nesta filial'
            USING ERRCODE = '23505';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS fin_audits_period_guard ON financial_audits;
CREATE TRIGGER fin_audits_period_guard
BEFORE INSERT OR UPDATE ON financial_audits
FOR EACH ROW EXECUTE FUNCTION fin_audits_period_guard();
