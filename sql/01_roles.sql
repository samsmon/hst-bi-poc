-- Read-only role for the BI. NOT executed by P1: run in P2 after approval,
-- as a superuser, against the raw database:
--   psql -d hst_dataset -v bi_reader_password="$BI_READER_PASSWORD" -f sql/01_roles.sql
-- The password comes from the environment, never from this repo.
-- Idempotent: safe to run again.

SELECT format('CREATE ROLE bi_reader LOGIN PASSWORD %L', :'bi_reader_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'bi_reader')
\gexec

ALTER ROLE bi_reader PASSWORD :'bi_reader_password';
ALTER ROLE bi_reader SET statement_timeout = '30s';
ALTER ROLE bi_reader SET lock_timeout = '2s';
ALTER ROLE bi_reader SET idle_in_transaction_session_timeout = '10s';
ALTER ROLE bi_reader SET default_transaction_read_only = on;

GRANT CONNECT ON DATABASE hst_dataset TO bi_reader;
GRANT USAGE ON SCHEMA public TO bi_reader;
REVOKE CREATE ON SCHEMA public FROM bi_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO bi_reader;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO bi_reader;
