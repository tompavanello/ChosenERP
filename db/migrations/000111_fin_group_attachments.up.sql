-- 000111_fin_group_attachments.up.sql
-- Permite comprovantes vinculados diretamente a uma linha de agrupamento
-- bancario, alem dos anexos por lancamento que ja existiam. A regra garante
-- que cada anexo pertence a exatamente um dono (lancamento OU grupo).

ALTER TABLE financial_attachments
    ALTER COLUMN transaction_id DROP NOT NULL;

ALTER TABLE financial_attachments
    ADD COLUMN group_id uuid REFERENCES financial_transaction_groups(id) ON DELETE CASCADE;

ALTER TABLE financial_attachments
    ADD CONSTRAINT finattach_owner_chk
    CHECK ((transaction_id IS NOT NULL) <> (group_id IS NOT NULL));

CREATE INDEX idx_fin_attach_group ON financial_attachments(group_id);

-- Necessario para o desagrupamento reapontar os comprovantes do grupo para um
-- lancamento membro sem perder o arquivo.
DROP POLICY IF EXISTS finattach_upd ON financial_attachments;
CREATE POLICY finattach_upd ON financial_attachments
  FOR UPDATE USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));
