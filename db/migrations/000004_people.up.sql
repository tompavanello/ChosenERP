-- 000004_people.up.sql
-- Membros, famílias, genealogia, visitantes e benfeitores

CREATE TABLE members (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    external_id   text,
    first_name    text NOT NULL,
    last_name     text NOT NULL,
    full_name     text NOT NULL,
    nickname      text,
    birth_date    date,
    gender        text,
    marital_status text, -- single | married | divorced | widowed
    cpf           text,
    rg            text,
    email         citext,
    phone         text,
    whatsapp      text,
    photo_url     text,
    membership_status text NOT NULL DEFAULT 'member', -- active | inactive | visitor | transferred | deceased
    joined_at     date,
    baptism_date  date,
    profession    text,
    office        text, -- diacono, presbitero, evangelista...
    extra_json    jsonb NOT NULL DEFAULT '{}'::jsonb, -- campos dinâmicos por denominação
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE families (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    head_id    uuid REFERENCES members(id) ON DELETE SET NULL,
    name       text NOT NULL,
    address    jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE member_relationships (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id    uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    related_id   uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    family_id    uuid REFERENCES families(id) ON DELETE SET NULL,
    kind         text NOT NULL, -- spouse | child | parent | discipler | disciple | dependent | relative
    detail       text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (member_id, related_id, kind)
);

CREATE TABLE visitors (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    first_name    text NOT NULL,
    last_name     text NOT NULL,
    email         citext,
    phone         text,
    source        text, -- indicado | evento | google | porta
    journey_stage text NOT NULL DEFAULT 'welcome', -- welcome | coffee_pastor | course | cell | converted
    converted_to_member_id uuid REFERENCES members(id) ON DELETE SET NULL,
    extra_json    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE benefactors (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => benfeitor global do tenant
    name       text NOT NULL,
    cpf        text,
    email      citext,
    phone      text,
    notes      text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_members_tenant_branch ON members(tenant_id, branch_id);
CREATE INDEX idx_members_status ON members(membership_status);
CREATE INDEX idx_visitors_branch ON visitors(branch_id);
CREATE INDEX idx_rels_member ON member_relationships(member_id);
