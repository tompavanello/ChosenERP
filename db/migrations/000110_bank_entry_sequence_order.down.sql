-- 000110_bank_entry_sequence_order.down.sql

DROP INDEX IF EXISTS idx_bank_entries_sequence;

ALTER TABLE bank_statement_entries
    DROP COLUMN IF EXISTS sequence_no;
