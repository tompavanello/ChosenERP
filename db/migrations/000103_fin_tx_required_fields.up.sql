-- 000103_fin_tx_required_fields.up.sql
-- Todo lancamento novo precisa de uma conta bancaria vinculada e de valor
-- positivo. A coluna account_id continua anulavel para preservar o historico
-- legado; o trigger garante a regra apenas para INSERT e para alteracoes que
-- mudem a conta/valor (o recadeamento da hash-chain nao e afetado).

CREATE OR REPLACE FUNCTION fin_tx_require_account() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.account_id IS NULL THEN
            RAISE EXCEPTION 'lancamento exige conta bancaria vinculada';
        END IF;
        IF NEW.amount IS NULL OR NEW.amount <= 0 THEN
            RAISE EXCEPTION 'valor do lancamento deve ser maior que zero';
        END IF;
        RETURN NEW;
    END IF;

    -- UPDATE: bloqueia limpar a conta, mas preserva linhas legadas que ainda
    -- nao tinham conta.
    IF NEW.account_id IS DISTINCT FROM OLD.account_id AND NEW.account_id IS NULL THEN
        RAISE EXCEPTION 'lancamento exige conta bancaria vinculada';
    END IF;

    -- UPDATE: so valida o valor quando ele muda. Assim fin_tx_rechain e
    -- fin_tx_edit (quando preservam o valor) continuam funcionando mesmo em
    -- registros legados zerados.
    IF NEW.amount IS DISTINCT FROM OLD.amount
       AND (NEW.amount IS NULL OR NEW.amount <= 0) THEN
        RAISE EXCEPTION 'valor do lancamento deve ser maior que zero';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS fin_tx_require_account ON financial_transactions;
CREATE TRIGGER fin_tx_require_account
    BEFORE INSERT OR UPDATE ON financial_transactions
    FOR EACH ROW EXECUTE FUNCTION fin_tx_require_account();
