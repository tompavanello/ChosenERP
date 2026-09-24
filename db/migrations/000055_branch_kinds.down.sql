-- 000055_branch_kinds.down.sql
-- Volta aos valores antigos: branch | congregation | sub_congregation.

DROP TRIGGER IF EXISTS branches_validate_hierarchy ON branches;
DROP FUNCTION IF EXISTS branches_validate_hierarchy();

ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_kind_check;
ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_matriz_root_check;
ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_pae_parent_check;

UPDATE branches SET kind = CASE
    WHEN kind = 'matriz' THEN 'branch'
    WHEN kind = 'filial' THEN 'congregation'
    WHEN kind = 'pae'    THEN 'sub_congregation'
    ELSE kind
END;

ALTER TABLE branches ALTER COLUMN kind SET DEFAULT 'branch';
