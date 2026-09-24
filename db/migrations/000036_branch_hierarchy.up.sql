-- 000036_branch_hierarchy.up.sql
-- Hierarquia de filiais (Fase 2 / item #10): Sede > Filial/Congregacao >
-- Sub-congregacao. As colunas `branches.parent_id` e `kind` ja existiam desde a
-- 000002; o que faltava era o RLS respeita-las.
--
-- Regra nova: a LEITURA de um usuario de filial passa a incluir os descendentes
-- (a congregacao enxerga as proprias sub-congregacoes), sem abrir as filiais
-- irmas. A GRAVACAO continua exata (branch do contexto) - exceto Sede/sistema -
-- para nao permitir que uma congregacao escreva no cadastro da sub-congregacao
-- sem um fluxo proprio.
--
-- Implementacao: o gateway grava em `app.branch_scope` (GUC de transacao) a
-- lista de ids da filial do contexto + descendentes (ver store.WithTenant). A
-- politica de leitura consulta essa lista - barato por linha, sem recursao no
-- caminho quente.

-- ---------------------------------------------------------------------------
-- Escopo de leitura hierarquico
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION rls_read_scope(p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT rls_branch_match(p_branch, p_allow_global)
        OR (p_branch IS NOT NULL
            AND p_branch::text = ANY(
                string_to_array(NULLIF(current_setting('app.branch_scope', true), ''), ',')
            ))
$$;

CREATE OR REPLACE FUNCTION rls_read(p_tenant uuid, p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT is_system()
        OR (current_tenant() IS NOT NULL
            AND p_tenant = current_tenant()
            AND (rls_read_scope(p_branch, p_allow_global) OR is_headquarters()))
$$;
