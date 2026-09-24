-- 000037_sel_policies_select_only.up.sql
-- Correção de política latente: as políticas `*_sel` foram criadas (000007,
-- 000016 e seguintes) como `FOR ALL` por omissão — sem `FOR SELECT`. Pela regra
-- do PostgreSQL, uma política ALL sem `WITH CHECK` reaproveita o `USING` também
-- para gravar. Ou seja: quem podia LER a linha também podia INSERIR/ALTERAR
-- (a política `_all` somava-se por OR, não restringia).
--
-- Enquanto a leitura era o branch exato, isso não aparecia. Com a leitura
-- hierárquica (000036), a filial passaria a gravar na sub-congregação pela
-- política de leitura. Esta migração torna cada `*_sel` realmente `FOR SELECT`;
-- a escrita continua coberta pelas políticas `*_all` / `*_ins` / `*_upd`, que
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
