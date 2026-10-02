-- 000081_group_members.up.sql
-- Vinculo membro <-> grupo/celula com papel.
--
-- Ate aqui so existia small_groups.leader_id e a presenca (group_attendance);
-- nao havia como dizer quem PARTICIPA do grupo. Esse vinculo e a base do
-- "Meu GD" no app do membro (participantes, anfitriao, secretario).
CREATE TABLE group_members (
    group_id  uuid NOT NULL REFERENCES small_groups(id) ON DELETE CASCADE,
    member_id uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    role      text NOT NULL DEFAULT 'member',
    joined_at date,
    PRIMARY KEY (group_id, member_id),
    CONSTRAINT group_members_role_check CHECK (role IN ('member', 'host', 'secretary', 'leader'))
);

CREATE INDEX idx_group_members_member ON group_members (member_id);

ALTER TABLE group_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE group_members FORCE ROW LEVEL SECURITY;

-- Sem tenant_id/branch_id proprio: herda o escopo do grupo (mesmo padrao de
-- ministry_members) - leitura pela leitura do grupo, escrita pela escrita.
DROP POLICY IF EXISTS group_mem_sel ON group_members;
CREATE POLICY group_mem_sel ON group_members USING (EXISTS (
    SELECT 1 FROM small_groups g WHERE g.id = group_id AND rls_read(g.tenant_id, g.branch_id, false)
));
DROP POLICY IF EXISTS group_mem_all ON group_members;
CREATE POLICY group_mem_all ON group_members
  USING (EXISTS (SELECT 1 FROM small_groups g WHERE g.id = group_id AND rls_write(g.tenant_id, g.branch_id, false)))
  WITH CHECK (EXISTS (SELECT 1 FROM small_groups g WHERE g.id = group_id AND rls_write(g.tenant_id, g.branch_id, false)));
