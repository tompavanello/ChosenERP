-- 000109_bank_entry_fitid_not_unique.up.sql
-- Alguns bancos repetem o FITID/documento em varias linhas legitimas do mesmo
-- extrato (ex.: varios PIX com o mesmo numero de documento e valores
-- diferentes). O indice UNIQUE por (import_id, fitid) fazia o import descartar
-- essas linhas silenciosamente (ON CONFLICT DO NOTHING).
--
-- A deduplicacao dentro da importacao passa a ser feita por (import_id, line_no),
-- que e a posicao real da linha no arquivo. O fitid continua indexado apenas
-- para consulta.

DROP INDEX IF EXISTS uq_bank_entries_fitid;

CREATE INDEX IF NOT EXISTS idx_bank_entries_fitid
    ON bank_statement_entries(import_id, fitid)
    WHERE fitid IS NOT NULL;
