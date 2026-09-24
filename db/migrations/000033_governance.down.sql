-- 000033_governance.down.sql
DROP TABLE IF EXISTS legal_documents;
DROP TABLE IF EXISTS vote_ballots;
DROP FUNCTION IF EXISTS vote_ballots_hash();
DROP FUNCTION IF EXISTS vote_ballots_guard();
DROP TABLE IF EXISTS vote_registrations;
DROP TABLE IF EXISTS vote_options;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS minute_signatures;
DROP FUNCTION IF EXISTS minute_signatures_hash();
DROP FUNCTION IF EXISTS minute_signatures_guard();
DROP TABLE IF EXISTS minutes;
