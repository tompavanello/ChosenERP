-- 000073_link_member_identity.up.sql
-- Vinculo identidade <-> pessoa (membro) POR IGREJA.
--
-- Ate aqui `users` (identidade global) + `memberships` (user x igreja) nao
-- apontavam para o registro em `members`. Sem esse vinculo um login de membro
-- nao consegue achar o proprio cadastro, o que bloqueia qualquer app do membro
-- (perfil, contribuicoes, familia...).
--
-- O vinculo mora em `memberships.member_id` (e nao em `members.user_id`) porque
-- a relacao e por igreja: a mesma pessoa pode ser membro na igreja A e ter
-- outro papel na igreja B. `members` ja e por tenant.
--
-- O preenchimento e feito por backfill idempotente (`cmd/link-members`,
-- rodado uma vez, fora das migracoes), porque casar por CPF/e-mail pode ter
-- ambiguidade e merece revisao humana.

ALTER TABLE memberships
    ADD COLUMN member_id uuid REFERENCES members(id) ON DELETE SET NULL;

-- Um membro so pode estar ligado a uma unica identidade dentro do tenant.
CREATE UNIQUE INDEX uq_memberships_tenant_member
    ON memberships (tenant_id, member_id)
    WHERE member_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- auth_memberships passa a devolver member_id.
-- O tipo de retorno muda, entao a funcao precisa ser removida antes (o
-- CREATE OR REPLACE nao altera a assinatura de retorno).
-- ---------------------------------------------------------------------------
DROP FUNCTION IF EXISTS auth_memberships(uuid);

CREATE FUNCTION auth_memberships(p_user_id uuid)
RETURNS TABLE (
    tenant_id   text,
    tenant_name text,
    tenant_slug text,
    role_key    text,
    branch_id   text,
    is_active   boolean,
    member_id   text
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.id::text, t.name, t.slug, r.key, m.branch_id::text, m.is_active,
           m.member_id::text
    FROM memberships m
    JOIN tenants t ON t.id = m.tenant_id
    JOIN roles   r ON r.id = m.role_id
    WHERE m.user_id = p_user_id
    ORDER BY m.is_active DESC, t.name;
$$;

GRANT EXECUTE ON FUNCTION auth_memberships(uuid) TO public;
