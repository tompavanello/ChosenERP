-- 000045_rls_write_headquarters.up.sql
-- Corrige a MANUTENÇÃO por usuários de Sede (escopo branch NULL + papéis
-- super_admin/admin_sede).
--
-- A migração 000016 deixou rls_write() SEM is_headquarters() de propósito
-- ("Escopo Sede NÃO ganha gravação em branch alheio — só via rls_hq() explícito").
-- O efeito colateral era: a Sede enxergava todo o tenant (rls_read inclui
-- is_headquarters), mas qualquer UPDATE/DELETE numa linha de filial afetava
-- 0 linhas e os handlers traduziam isso em 404:
--   * PATCH /api/v1/events/{id}                    (church_events)
--   * POST  /api/v1/finance/transactions/{id}/void (financial_transactions)
-- e o mesmo valeria para members, famílias, ministérios, escalas, atas etc.
--
-- Regra agora simétrica à leitura: quem TEM filial continua gravando apenas no
-- branch exato (TestRLS_BranchWriteStaysExact segue válido); a Sede SEM filial
-- (ou o sistema/worker) grava em qualquer filial do PRÓPRIO tenant. O isolamento
-- entre tenants permanece garantido por p_tenant = current_tenant().
CREATE OR REPLACE FUNCTION rls_write(p_tenant uuid, p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT is_system()
        OR (current_tenant() IS NOT NULL
            AND p_tenant = current_tenant()
            AND (rls_branch_match(p_branch, p_allow_global) OR is_headquarters()))
$$;
