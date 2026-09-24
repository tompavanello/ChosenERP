-- 000037_sel_policies_select_only.up.sql
-- Correcao de politica latente: as politicas `*_sel` foram criadas (000007,
-- 000016 e seguintes) como `FOR ALL` por omissao - sem `FOR SELECT`. Pela regra
-- do PostgreSQL, uma politica ALL sem `WITH CHECK` reaproveita o `USING` tambem
-- para gravar. Ou seja: quem podia LER a linha tambem podia INSERIR/ALTERAR
-- (a politica `_all` somava-se por OR, nao restringia).
--
-- Enquanto a leitura era o branch exato, isso nao aparecia. Com a leitura
-- hierarquica (000036), a filial passaria a gravar na sub-congregacao pela
-- politica de leitura. Esta migracao torna cada `*_sel` realmente `FOR SELECT`;
-- a escrita continua coberta pelas politicas `*_all` / `*_ins` / `*_upd`, que
-- usam rls_write (branch exato).
--
-- Verificado pelos testes: TestRLS_BranchWriteStaysExact.

DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT tablename, policyname, qual
        FROM pg_policies
        WHERE schemaname = 'public'
          AND cmd = 'ALL'
          AND policyname LIKE '%\_sel'
    LOOP
        EXECUTE format('DROP POLICY %I ON %I', r.policyname, r.tablename);
        EXECUTE format('CREATE POLICY %I ON %I FOR SELECT USING (%s)',
                       r.policyname, r.tablename, r.qual);
    END LOOP;
END $$;
