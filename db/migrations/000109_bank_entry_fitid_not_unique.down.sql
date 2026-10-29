-- 000109_bank_entry_fitid_not_unique.down.sql
-- Recria o indice unico. Observacao: se ja houver FITIDs repetidos (situacao
-- legitima) a criacao falha; a importacao passou a depender de (import_id, line_no).

DROP INDEX IF EXISTS idx_bank_entries_fitid;

CREATE UNIQUE INDEX IF NOT EXISTS uq_bank_entries_fitid
    ON bank_statement_entries(import_id, fitid)
    WHERE fitid IS NOT NULL;
