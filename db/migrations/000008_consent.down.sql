-- 000008_consent.down.sql
DROP POLICY IF EXISTS member_consents_all ON member_consents;
DROP POLICY IF EXISTS member_consents_sel ON member_consents;
DROP POLICY IF EXISTS consent_terms_all ON consent_terms;
DROP POLICY IF EXISTS consent_terms_sel ON consent_terms;
ALTER TABLE member_consents DISABLE ROW LEVEL SECURITY;
ALTER TABLE consent_terms DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS member_consents;
DROP TABLE IF EXISTS consent_terms;
