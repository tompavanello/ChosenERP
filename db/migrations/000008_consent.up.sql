-- 000008_consent.up.sql
-- LGPD: termos de consentimento e consentimento digital por pessoa

CREATE TABLE consent_terms (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version     int NOT NULL DEFAULT 1,
    title       text NOT NULL,
    body        text NOT NULL,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, version)
);

CREATE TABLE member_consents (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id        uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    consent_term_id  uuid NOT NULL REFERENCES consent_terms(id) ON DELETE CASCADE,
    subject_type     text NOT NULL, -- member | visitor | benefactor | user
    subject_id       uuid NOT NULL,
    consented        boolean NOT NULL DEFAULT false,
    consented_at     timestamptz,
    ip               text,
    user_agent       text,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_consents_subject ON member_consents(subject_type, subject_id);

ALTER TABLE consent_terms    ENABLE ROW LEVEL SECURITY;
ALTER TABLE member_consents  ENABLE ROW LEVEL SECURITY;

CREATE POLICY consent_terms_sel ON consent_terms USING (tenant_id = current_tenant() OR is_headquarters());
CREATE POLICY consent_terms_all ON consent_terms FOR ALL USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());

CREATE POLICY member_consents_sel ON member_consents USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY member_consents_all ON member_consents FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());
