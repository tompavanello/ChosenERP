-- 000003_users_rbac_audit.down.sql
DROP TRIGGER IF EXISTS audit_log_hash ON audit_log;
DROP TRIGGER IF EXISTS audit_log_no_update ON audit_log;
DROP FUNCTION IF EXISTS audit_log_hash();
DROP FUNCTION IF EXISTS audit_log_guard();
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
