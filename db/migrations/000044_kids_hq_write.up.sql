-- 000044_kids_hq_write.up.sql
-- Permite que a Sede (papel super_admin/admin_sede sem filial) gerencie as
-- tabelas do Kids por filial, como ja vale para repasses (`transfers`). Sem
-- isso, um usuario de Sede nao conseguia criar turma/encontro (a RLS de
-- escrita exige a filial exata).

DROP POLICY IF EXISTS kids_classes_all ON kids_classes;
CREATE POLICY kids_classes_all ON kids_classes
  USING (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id))
  WITH CHECK (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id));

DROP POLICY IF EXISTS kids_enrollments_all ON kids_enrollments;
CREATE POLICY kids_enrollments_all ON kids_enrollments
  USING (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id))
  WITH CHECK (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id));

DROP POLICY IF EXISTS kids_guardians_all ON kids_guardians;
CREATE POLICY kids_guardians_all ON kids_guardians
  USING (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id))
  WITH CHECK (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id));

DROP POLICY IF EXISTS kids_sessions_all ON kids_sessions;
CREATE POLICY kids_sessions_all ON kids_sessions
  USING (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id))
  WITH CHECK (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id));

DROP POLICY IF EXISTS kids_checkins_all ON kids_checkins;
CREATE POLICY kids_checkins_all ON kids_checkins
  USING (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id))
  WITH CHECK (rls_write(tenant_id, branch_id, false) OR rls_hq(tenant_id));
