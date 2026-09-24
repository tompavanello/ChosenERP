-- 000056_branch_channels.down.sql
ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_whatsapp_status_check;
ALTER TABLE branches
    DROP COLUMN whatsapp_phone,
    DROP COLUMN whatsapp_instance,
    DROP COLUMN whatsapp_status,
    DROP COLUMN smtp_host,
    DROP COLUMN smtp_port,
    DROP COLUMN smtp_user,
    DROP COLUMN smtp_password,
    DROP COLUMN smtp_from,
    DROP COLUMN smtp_from_name,
    DROP COLUMN smtp_secure;
