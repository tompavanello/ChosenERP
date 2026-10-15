-- 000090_billing.down.sql
DROP TABLE IF EXISTS billing_webhook_events;
DROP TABLE IF EXISTS billing_invoices;
DROP TABLE IF EXISTS billing_subscriptions;
DROP TABLE IF EXISTS billing_customers;
ALTER TABLE plans DROP COLUMN IF EXISTS gateway_refs;
ALTER TABLE plans DROP COLUMN IF EXISTS cycle;
ALTER TABLE tenants DROP COLUMN IF EXISTS suspended_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS past_due_since;
ALTER TABLE tenants DROP COLUMN IF EXISTS billing_exempt;
ALTER TABLE tenants DROP COLUMN IF EXISTS trial_ends_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS billing_status;
