-- 000081_group_members.down.sql
DROP POLICY IF EXISTS group_mem_all ON group_members;
DROP POLICY IF EXISTS group_mem_sel ON group_members;
DROP TABLE IF EXISTS group_members;
