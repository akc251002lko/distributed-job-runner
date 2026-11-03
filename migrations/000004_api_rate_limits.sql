BEGIN;
CREATE TABLE IF NOT EXISTS api_rate_limits (client_id varchar(64) PRIMARY KEY, tokens double precision NOT NULL CHECK (tokens >= 0), updated_at timestamptz NOT NULL DEFAULT now());
ALTER TABLE api_rate_limits ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_rate_limits FORCE ROW LEVEL SECURITY;
COMMIT;
