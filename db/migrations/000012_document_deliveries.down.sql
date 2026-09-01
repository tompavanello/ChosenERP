-- 000012_document_deliveries.down.sql
DROP POLICY IF EXISTS deliveries_all ON document_deliveries;
DROP POLICY IF EXISTS deliveries_sel ON document_deliveries;
ALTER TABLE document_deliveries DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS document_deliveries;
