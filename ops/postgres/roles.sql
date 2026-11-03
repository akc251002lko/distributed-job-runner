-- Execute as a database owner/superuser after migrations.
-- Supply secrets with: psql "$DATABASE_URL" -v api_password='...' -v worker_password='...' -f ops/postgres/roles.sql
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'djr_api') THEN
    CREATE ROLE djr_api LOGIN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'djr_worker') THEN
    CREATE ROLE djr_worker LOGIN;
  END IF;
END
$$;
ALTER ROLE djr_api PASSWORD :'api_password';
ALTER ROLE djr_worker PASSWORD :'worker_password';
GRANT USAGE ON SCHEMA public TO djr_api, djr_worker;
GRANT SELECT, INSERT ON TABLE jobs TO djr_api;
GRANT UPDATE (idempotency_key,status,lease_token,lease_expires_at,worker_id,updated_at) ON TABLE jobs TO djr_api;
GRANT SELECT ON TABLE jobs TO djr_worker;
GRANT UPDATE (status,attempts,run_at,lease_token,lease_expires_at,worker_id,last_error,completed_at,updated_at) ON TABLE jobs TO djr_worker;

ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS jobs_api_tenant_isolation ON jobs;
CREATE POLICY jobs_api_tenant_isolation ON jobs FOR ALL TO djr_api
  USING (client_id = current_setting('app.client_id', true))
  WITH CHECK (client_id = current_setting('app.client_id', true));
DROP POLICY IF EXISTS jobs_worker_access ON jobs;
CREATE POLICY jobs_worker_access ON jobs FOR ALL TO djr_worker
  USING (true) WITH CHECK (true);
