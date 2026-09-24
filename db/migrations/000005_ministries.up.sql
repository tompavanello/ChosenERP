-- 000005_ministries.up.sql
-- Ministerios, voluntariado, celulas/pequenos grupos e frequencia

CREATE TABLE ministries (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    name          text NOT NULL,
    slug          text NOT NULL,
    description   text,
    leader_id     uuid REFERENCES members(id) ON DELETE SET NULL,
    extra_json    jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (branch_id, slug)
);

CREATE TABLE ministry_members (
    ministry_id  uuid NOT NULL REFERENCES ministries(id) ON DELETE CASCADE,
    member_id    uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    role         text NOT NULL DEFAULT 'volunteer', -- leader | coordinator | volunteer
    started_at   date,
    PRIMARY KEY (ministry_id, member_id)
);

CREATE TABLE small_groups (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    ministry_id   uuid REFERENCES ministries(id) ON DELETE SET NULL,
    name          text NOT NULL,
    kind          text NOT NULL DEFAULT 'cell', -- cell | ebd | family
    leader_id     uuid REFERENCES members(id) ON DELETE SET NULL,
    address       jsonb,
    geo           jsonb, -- { lat, lng } para mapa de calor
    max_members   int,
    weekday       smallint, -- 0 = domingo ... 6 = sabado
    meeting_time  time,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE group_attendance (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    small_group_id uuid NOT NULL REFERENCES small_groups(id) ON DELETE CASCADE,
    member_id     uuid REFERENCES members(id) ON DELETE SET NULL,
    attended_at   timestamptz NOT NULL DEFAULT now(),
    present       boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ministries_branch ON ministries(tenant_id, branch_id);
CREATE INDEX idx_groups_branch ON small_groups(tenant_id, branch_id);
CREATE INDEX idx_group_attendance_group ON group_attendance(small_group_id, attended_at);
