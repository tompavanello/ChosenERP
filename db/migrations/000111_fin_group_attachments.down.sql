-- 000111_fin_group_attachments.down.sql
-- Remove os anexos de grupo antes de voltar a exigir transaction_id.

DELETE FROM financial_attachments WHERE group_id IS NOT NULL;

DROP POLICY IF EXISTS finattach_upd ON financial_attachments;

DROP INDEX IF EXISTS idx_fin_attach_group;

ALTER TABLE financial_attachments
    DROP CONSTRAINT IF EXISTS finattach_owner_chk;

ALTER TABLE financial_attachments
    DROP COLUMN IF EXISTS group_id;

ALTER TABLE financial_attachments
    ALTER COLUMN transaction_id SET NOT NULL;
