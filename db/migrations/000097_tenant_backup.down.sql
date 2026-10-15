-- 000097_tenant_backup.down.sql
DROP FUNCTION IF EXISTS restore_tenant(uuid, jsonb);
DROP FUNCTION IF EXISTS export_tenant(uuid);
DROP FUNCTION IF EXISTS _backup_insert_rows(text, jsonb);
DROP FUNCTION IF EXISTS _backup_child_predicate(text, text[]);
DROP FUNCTION IF EXISTS _backup_child_tables(text[]);
DROP FUNCTION IF EXISTS _backup_tenant_tables();
DROP TABLE IF EXISTS backup_settings;
DROP TABLE IF EXISTS tenant_backups;
