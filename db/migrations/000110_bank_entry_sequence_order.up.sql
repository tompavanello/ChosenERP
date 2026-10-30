-- 000110_bank_entry_sequence_order.up.sql
-- Ordem de trabalho das linhas do extrato bancario.
--
-- line_no continua sendo a posicao ORIGINAL da linha no arquivo (auditoria).
-- sequence_no e a ordem de trabalho usada para gerar/alinhar os lancamentos e
-- pode ser alterada pelo usuario na conciliacao (sem tocar no line_no).

ALTER TABLE bank_statement_entries
    ADD COLUMN IF NOT EXISTS sequence_no integer;

UPDATE bank_statement_entries
SET sequence_no = line_no
WHERE sequence_no IS NULL;

ALTER TABLE bank_statement_entries
    ALTER COLUMN sequence_no SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_bank_entries_sequence
    ON bank_statement_entries(import_id, posted_at, sequence_no);
