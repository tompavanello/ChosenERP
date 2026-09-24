-- 000043_kids.up.sql
-- Ministerio Infantil (Kids): trilha de conteudo, turmas, participantes,
-- check-in com responsavel/codigo de seguranca e relatorios de evolucao.
--
-- Modelo:
--   kids_tracks      curriculo (ex.: "Fundamentos da Fe") - pode ser global do tenant
--   kids_lessons     licoes ordenadas da trilha (objetivo, versiculo, conteudo, material)
--   kids_classes     turmas por faixa etaria, opcionalmente ligadas a uma trilha
--   kids_enrollments matricula de um MEMBRO crianca em uma turma
--   kids_guardians   responsaveis da matricula (membros adultos)
--   kids_sessions    encontros da turma (ministram uma licao)
--   kids_checkins    presenca por encontro: check-in/check-out, responsavel, codigo
--
-- Isolamento: tudo usa rls_read/rls_write; trilhas/licoes tem branch NULL = global
-- do tenant, o resto e por filial (branch NOT NULL).

-- ---------------------------------------------------------------------------
-- Trilha / conteudo
-- ---------------------------------------------------------------------------
CREATE TABLE kids_tracks (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid REFERENCES branches(id) ON DELETE CASCADE, -- NULL = todas as filiais
    name        text NOT NULL,
    description text,
    age_min     smallint,
    age_max     smallint,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT kids_tracks_age_check CHECK (
        (age_min IS NULL OR age_min >= 0) AND (age_max IS NULL OR age_max >= age_min)
    )
);

CREATE INDEX idx_kids_tracks_tenant ON kids_tracks(tenant_id, name);

ALTER TABLE kids_tracks ENABLE ROW LEVEL SECURITY;
CREATE POLICY kids_tracks_sel ON kids_tracks
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY kids_tracks_all ON kids_tracks
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

CREATE TABLE kids_lessons (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid REFERENCES branches(id) ON DELETE CASCADE, -- espelha a trilha
    track_id    uuid NOT NULL REFERENCES kids_tracks(id) ON DELETE CASCADE,
    position    int NOT NULL,
    title       text NOT NULL,
    objective   text,
    verse       text,
    content     text,
    materials   text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (track_id, position)
);

CREATE INDEX idx_kids_lessons_track ON kids_lessons(track_id, position);

ALTER TABLE kids_lessons ENABLE ROW LEVEL SECURITY;
CREATE POLICY kids_lessons_sel ON kids_lessons
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY kids_lessons_all ON kids_lessons
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- Turmas
-- ---------------------------------------------------------------------------
CREATE TABLE kids_classes (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id        uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    name             text NOT NULL,
    age_min          smallint,
    age_max          smallint,
    track_id         uuid REFERENCES kids_tracks(id) ON DELETE SET NULL,
    room             text,
    leader_member_id uuid REFERENCES members(id) ON DELETE SET NULL,
    is_active        boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT kids_classes_age_check CHECK (
        (age_min IS NULL OR age_min >= 0) AND (age_max IS NULL OR age_max >= age_min)
    )
);

CREATE INDEX idx_kids_classes_branch ON kids_classes(tenant_id, branch_id, name);

ALTER TABLE kids_classes ENABLE ROW LEVEL SECURITY;
CREATE POLICY kids_classes_sel ON kids_classes
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY kids_classes_all ON kids_classes
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Participantes (matriculas de membros criancas) e responsaveis
-- ---------------------------------------------------------------------------
CREATE TABLE kids_enrollments (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id            uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    class_id             uuid NOT NULL REFERENCES kids_classes(id) ON DELETE CASCADE,
    member_id            uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    status               text NOT NULL DEFAULT 'active',
    start_date           date NOT NULL DEFAULT current_date,
    end_date             date,
    dietary_restrictions text,
    notes                text,
    created_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT kids_enrollments_status_check CHECK (status IN ('active','paused','ended')),
    CONSTRAINT kids_enrollments_period_check CHECK (end_date IS NULL OR end_date >= start_date),
    UNIQUE (class_id, member_id)
);

CREATE INDEX idx_kids_enrollments_class ON kids_enrollments(class_id, status);
CREATE INDEX idx_kids_enrollments_member ON kids_enrollments(member_id);

ALTER TABLE kids_enrollments ENABLE ROW LEVEL SECURITY;
CREATE POLICY kids_enrollments_sel ON kids_enrollments
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY kids_enrollments_all ON kids_enrollments
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

CREATE TABLE kids_guardians (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    enrollment_id uuid NOT NULL REFERENCES kids_enrollments(id) ON DELETE CASCADE,
    member_id     uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    relationship  text,
    is_primary    boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (enrollment_id, member_id)
);

CREATE INDEX idx_kids_guardians_enrollment ON kids_guardians(enrollment_id);

ALTER TABLE kids_guardians ENABLE ROW LEVEL SECURITY;
CREATE POLICY kids_guardians_sel ON kids_guardians
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY kids_guardians_all ON kids_guardians
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Encontros e check-in
-- ---------------------------------------------------------------------------
CREATE TABLE kids_sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    class_id   uuid NOT NULL REFERENCES kids_classes(id) ON DELETE CASCADE,
    lesson_id  uuid REFERENCES kids_lessons(id) ON DELETE SET NULL,
    starts_at  timestamptz NOT NULL,
    ends_at    timestamptz,
    status     text NOT NULL DEFAULT 'scheduled',
    notes      text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT kids_sessions_status_check CHECK (status IN ('scheduled','open','closed')),
    CONSTRAINT kids_sessions_period_check CHECK (ends_at IS NULL OR ends_at >= starts_at)
);

CREATE INDEX idx_kids_sessions_class ON kids_sessions(class_id, starts_at DESC);

ALTER TABLE kids_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY kids_sessions_sel ON kids_sessions
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY kids_sessions_all ON kids_sessions
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

CREATE TABLE kids_checkins (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id                uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    session_id               uuid NOT NULL REFERENCES kids_sessions(id) ON DELETE CASCADE,
    enrollment_id            uuid NOT NULL REFERENCES kids_enrollments(id) ON DELETE CASCADE,
    status                   text NOT NULL DEFAULT 'present',
    security_code            text,
    checkin_at               timestamptz,
    checkout_at              timestamptz,
    dropoff_guardian_id      uuid REFERENCES kids_guardians(id) ON DELETE SET NULL,
    pickup_guardian_id       uuid REFERENCES kids_guardians(id) ON DELETE SET NULL,
    notes                    text,
    created_at               timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT kids_checkins_status_check CHECK (status IN ('present','absent')),
    UNIQUE (session_id, enrollment_id)
);

CREATE INDEX idx_kids_checkins_session ON kids_checkins(session_id);

ALTER TABLE kids_checkins ENABLE ROW LEVEL SECURITY;
CREATE POLICY kids_checkins_sel ON kids_checkins
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY kids_checkins_all ON kids_checkins
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
