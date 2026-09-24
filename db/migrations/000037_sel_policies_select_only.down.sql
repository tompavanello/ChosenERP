-- 000037_sel_policies_select_only.down.sql
-- Reverte para o comportamento anterior (política `*_sel` como ALL). É o estado
-- legado com o defeito descrito na migração up; mantido apenas para rollback.
DO $$
DECLARE
    r record;
    names text[] := ARRAY[
        'announcements_sel','benefactors_sel','branch_sel','consent_terms_sel',
        'deliveries_sel','docs_sel','families_sel','finacct_sel','fincat_sel',
        'fintx_sel','att_sel','member_consents_sel','rels_sel','members_sel',
        'ministries_sel','ministry_mem_sel','roles_sel','groups_sel','tenant_sel',
        'transfers_sel','users_sel','guests_sel'
    ];
BEGIN
    FOR r IN
        SELECT tablename, policyname, qual
        FROM pg_policies
        WHERE schemaname = 'public'
          AND cmd = 'SELECT'
          AND policyname = ANY(names)
    LOOP
        EXECUTE format('DROP POLICY %I ON %I', r.policyname, r.tablename);
        EXECUTE format('CREATE POLICY %I ON %I USING (%s)',
                       r.policyname, r.tablename, r.qual);
    END LOOP;
END $$;
