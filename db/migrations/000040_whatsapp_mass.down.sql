-- 000040_whatsapp_mass.down.sql
DROP TABLE IF EXISTS notification_settings;

-- Restaura as políticas originais do outbox (000017).
DROP POLICY IF EXISTS ann_dlv_all ON announcement_deliveries;
CREATE POLICY ann_dlv_all ON announcement_deliveries FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL)
  WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);
DROP POLICY IF EXISTS ann_dlv_sel ON announcement_deliveries;
CREATE POLICY ann_dlv_sel ON announcement_deliveries FOR SELECT
  USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());

DROP INDEX IF EXISTS idx_ann_dlv_dedupe;

ALTER TABLE announcement_deliveries
    DROP CONSTRAINT IF EXISTS announcement_deliveries_source_check;

ALTER TABLE announcement_deliveries
    ALTER COLUMN announcement_id SET NOT NULL,
    DROP COLUMN IF EXISTS dedupe_key,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS recipient_name,
    DROP COLUMN IF EXISTS body,
    DROP COLUMN IF EXISTS title;
