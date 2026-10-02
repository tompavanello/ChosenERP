-- 000083_push_subscriptions.up.sql
-- Inscricoes de Web Push do app do membro (uma por navegador/dispositivo).
-- O envio usa VAPID (chaves VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY) e o service
-- worker exibe a notificacao; a inscricao pertence a uma identidade (users) e a
-- uma igreja (tenant).
CREATE TABLE push_subscriptions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid REFERENCES branches(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint     text NOT NULL UNIQUE,
    p256dh       text NOT NULL,
    auth         text NOT NULL,
    user_agent   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_subs_user ON push_subscriptions (user_id);
CREATE INDEX idx_push_subs_tenant ON push_subscriptions (tenant_id);

ALTER TABLE push_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE push_subscriptions FORCE ROW LEVEL SECURITY;

-- Cada usuario gerencia apenas as proprias inscricoes; o worker (system) le
-- todas para enviar. rls_read/rls_write garantem o isolamento tenant/filial.
DROP POLICY IF EXISTS push_sub_sel ON push_subscriptions;
CREATE POLICY push_sub_sel ON push_subscriptions FOR SELECT
  USING (rls_read(tenant_id, branch_id, true) AND (is_system() OR user_id = current_user_id()));
DROP POLICY IF EXISTS push_sub_ins ON push_subscriptions;
CREATE POLICY push_sub_ins ON push_subscriptions FOR INSERT
  WITH CHECK (rls_write(tenant_id, branch_id, true) AND (is_system() OR user_id = current_user_id()));
DROP POLICY IF EXISTS push_sub_upd ON push_subscriptions;
CREATE POLICY push_sub_upd ON push_subscriptions FOR UPDATE
  USING (rls_read(tenant_id, branch_id, true) AND (is_system() OR user_id = current_user_id()))
  WITH CHECK (rls_write(tenant_id, branch_id, true) AND (is_system() OR user_id = current_user_id()));
DROP POLICY IF EXISTS push_sub_del ON push_subscriptions;
CREATE POLICY push_sub_del ON push_subscriptions FOR DELETE
  USING (rls_read(tenant_id, branch_id, true) AND (is_system() OR user_id = current_user_id()));
