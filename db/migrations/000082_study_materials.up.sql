-- 000082_study_materials.up.sql
-- Materiais de estudo (arquivo ou link) publicados para a igreja/filial ou para
-- um grupo/celula especifico. Alimenta a tela "Materiais" do app do membro.
--
-- O arquivo fica no disco local (mesmo UPLOAD_DIR da foto de membro) sob um nome
-- opaco; o download passa por endpoint autenticado que confere o tenant via RLS.
CREATE TABLE study_materials (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    group_id     uuid REFERENCES small_groups(id) ON DELETE CASCADE,
    ministry_id  uuid REFERENCES ministries(id) ON DELETE SET NULL,
    title        text NOT NULL,
    description  text,
    kind         text NOT NULL DEFAULT 'file',
    url          text,
    disk_name    text,
    file_name    text,
    file_size    bigint,
    mime_type    text,
    is_published boolean NOT NULL DEFAULT true,
    created_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT study_materials_kind_check CHECK (kind IN ('file', 'link'))
);

CREATE INDEX idx_study_materials_scope ON study_materials (tenant_id, branch_id, is_published);
CREATE INDEX idx_study_materials_group ON study_materials (group_id);

ALTER TABLE study_materials ENABLE ROW LEVEL SECURITY;
ALTER TABLE study_materials FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS materials_sel ON study_materials;
CREATE POLICY materials_sel ON study_materials USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS materials_all ON study_materials;
CREATE POLICY materials_all ON study_materials
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
