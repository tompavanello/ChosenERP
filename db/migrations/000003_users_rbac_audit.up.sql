-- 000003_users_rbac_audit.up.sql
-- RBAC granular + trilha de auditoria imutavel (append-only, hash-chain)

CREATE TABLE roles (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key        text NOT NULL, -- super_admin | admin_sede | pastor_filial | tesoureiro | secretario | lider | pastor | contador | membro | visitante
    name       text NOT NULL,
    is_system  boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, key)
);

CREATE TABLE permissions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    key        text NOT NULL UNIQUE, -- ex.: members.read, finance.write, governance.vote
    module     text NOT NULL,
    name       text NOT NULL
);

CREATE TABLE role_permissions (
    role_id        uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id  uuid NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => escopo Sede
    role_id        uuid NOT NULL REFERENCES roles(id),
    email          citext NOT NULL UNIQUE,
    password_hash  text NOT NULL,
    full_name      text NOT NULL,
    mfa_secret     text,                       -- TOTP secret (nullable)
    mfa_enabled    boolean NOT NULL DEFAULT false,
    is_active      boolean NOT NULL DEFAULT true,
    last_login_at  timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- Trilha de auditoria imutavel (nunca UPDATE/DELETE)
CREATE TABLE audit_log (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    action      text NOT NULL,           -- ex.: member.created, finance.tx.inserted
    entity      text NOT NULL,           -- ex.: members, financial_transactions
    entity_id   uuid,
    payload     jsonb,
    prev_hash   text,
    hash        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_branch ON users(branch_id);
CREATE INDEX idx_audit_tenant ON audit_log(tenant_id, created_at);

-- ---------------------------------------------------------------------------
-- Append-only: bloqueia UPDATE/DELETE e encadeia hash de integridade
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION audit_log_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only (no UPDATE/DELETE)';
END;
$$;

CREATE TRIGGER audit_log_no_update
BEFORE UPDATE OR DELETE ON audit_log
FOR EACH ROW EXECUTE FUNCTION audit_log_guard();

-- Hash de integridade (chain): SHA-256 do conteudo + hash anterior
CREATE OR REPLACE FUNCTION audit_log_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.tenant_id::text || '|' || COALESCE(NEW.actor_id::text,'') || '|' ||
            NEW.action || '|' || NEW.entity || '|' || COALESCE(NEW.entity_id::text,'') || '|' ||
            COALESCE(NEW.payload::text,'') || '|' || NEW.created_at::text;
    NEW.prev_hash := (SELECT hash FROM audit_log WHERE id < NEW.id ORDER BY id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

CREATE TRIGGER audit_log_hash
BEFORE INSERT ON audit_log
FOR EACH ROW EXECUTE FUNCTION audit_log_hash();
