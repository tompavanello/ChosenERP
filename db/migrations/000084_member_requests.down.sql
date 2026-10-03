-- 000084_member_requests.down.sql
DROP POLICY IF EXISTS requests_all ON member_requests;
DROP POLICY IF EXISTS requests_sel ON member_requests;
DROP TABLE IF EXISTS member_requests;
