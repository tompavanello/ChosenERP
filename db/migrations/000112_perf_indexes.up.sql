-- 000112_perf_indexes.up.sql
-- Indices para os caminhos quentes identificados na auditoria de performance.
-- APENAS aditivo: nenhum dado e alterado/removido.

-- Busca de membro por nome (members.List usa ILIKE '%termo%').
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_members_full_name_trgm
    ON members USING gin (full_name gin_trgm_ops);

-- Listagem do financeiro: join do recibo por jsonb sem indice
-- (finance.List: d.content->>'tx_id' = t.id::text).
CREATE INDEX IF NOT EXISTS idx_documents_receipt_tx
    ON documents ((content->>'tx_id'))
    WHERE kind = 'receipt';

-- Documentos por membro (secretaria/certificados/cartas).
CREATE INDEX IF NOT EXISTS idx_documents_member
    ON documents (member_id, created_at DESC)
    WHERE member_id IS NOT NULL;

-- Contribuicoes do proprio membro no app.
CREATE INDEX IF NOT EXISTS idx_fin_tx_donor
    ON financial_transactions (donor_member_id)
    WHERE donor_member_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_fin_tx_category
    ON financial_transactions (category_id)
    WHERE category_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_fin_tx_branch
    ON financial_transactions (branch_id);

-- Ordem de listagem do ledger (cursor por data).
CREATE INDEX IF NOT EXISTS idx_fin_tx_list
    ON financial_transactions (tenant_id, occurred_at DESC, created_at DESC, id DESC);

-- Tabelas tenant-scoped de maior volume que ainda varriam a tabela inteira
-- sob RLS (tenant_id/branch_id sem indice lider).
CREATE INDEX IF NOT EXISTS idx_visitors_tenant_branch
    ON visitors (tenant_id, branch_id);

CREATE INDEX IF NOT EXISTS idx_event_attendance_tenant
    ON event_attendance (tenant_id, event_id);

CREATE INDEX IF NOT EXISTS idx_event_invitees_tenant
    ON event_invitees (tenant_id, event_id);

CREATE INDEX IF NOT EXISTS idx_roster_assignments_tenant
    ON roster_assignments (tenant_id, roster_id);

CREATE INDEX IF NOT EXISTS idx_member_consents_tenant
    ON member_consents (tenant_id, subject_id);

CREATE INDEX IF NOT EXISTS idx_prayer_reactions_tenant
    ON prayer_reactions (tenant_id, prayer_request_id);

CREATE INDEX IF NOT EXISTS idx_group_attendance_tenant
    ON group_attendance (tenant_id, small_group_id);

CREATE INDEX IF NOT EXISTS idx_fin_event_alloc_tenant
    ON financial_event_allocations (tenant_id, event_id);

CREATE INDEX IF NOT EXISTS idx_devotional_days_tenant
    ON devotional_plan_days (tenant_id, plan_id);

CREATE INDEX IF NOT EXISTS idx_devotional_progress_tenant
    ON devotional_progress (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS idx_kids_checkins_tenant
    ON kids_checkins (tenant_id, session_id);

CREATE INDEX IF NOT EXISTS idx_kids_enrollments_tenant
    ON kids_enrollments (tenant_id, class_id);
