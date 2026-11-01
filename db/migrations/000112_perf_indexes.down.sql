-- 000112_perf_indexes.down.sql
-- Reverte apenas os indices/extensao criados na 000112. Nenhum dado e tocado
-- (a extensao pg_trgm fica instalada; remove-la exigiria garantir que nenhum
-- outro objeto a usa, o que nao e necessario para o rollback de performance).

DROP INDEX IF EXISTS idx_kids_enrollments_tenant;
DROP INDEX IF EXISTS idx_kids_checkins_tenant;
DROP INDEX IF EXISTS idx_devotional_progress_tenant;
DROP INDEX IF EXISTS idx_devotional_days_tenant;
DROP INDEX IF EXISTS idx_fin_event_alloc_tenant;
DROP INDEX IF EXISTS idx_group_attendance_tenant;
DROP INDEX IF EXISTS idx_prayer_reactions_tenant;
DROP INDEX IF EXISTS idx_member_consents_tenant;
DROP INDEX IF EXISTS idx_roster_assignments_tenant;
DROP INDEX IF EXISTS idx_event_invitees_tenant;
DROP INDEX IF EXISTS idx_event_attendance_tenant;
DROP INDEX IF EXISTS idx_visitors_tenant_branch;
DROP INDEX IF EXISTS idx_fin_tx_list;
DROP INDEX IF EXISTS idx_fin_tx_branch;
DROP INDEX IF EXISTS idx_fin_tx_category;
DROP INDEX IF EXISTS idx_fin_tx_donor;
DROP INDEX IF EXISTS idx_documents_member;
DROP INDEX IF EXISTS idx_documents_receipt_tx;
DROP INDEX IF EXISTS idx_members_full_name_trgm;
