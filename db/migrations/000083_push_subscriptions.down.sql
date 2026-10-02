-- 000083_push_subscriptions.down.sql
DROP POLICY IF EXISTS push_sub_del ON push_subscriptions;
DROP POLICY IF EXISTS push_sub_upd ON push_subscriptions;
DROP POLICY IF EXISTS push_sub_ins ON push_subscriptions;
DROP POLICY IF EXISTS push_sub_sel ON push_subscriptions;
DROP TABLE IF EXISTS push_subscriptions;
