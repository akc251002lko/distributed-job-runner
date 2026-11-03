BEGIN;
CREATE TABLE IF NOT EXISTS jobs (
 id text PRIMARY KEY,
 job_type varchar(128) NOT NULL,
 payload jsonb NOT NULL DEFAULT '{}'::jsonb,
 idempotency_key varchar(255),
 status varchar(16) NOT NULL CHECK (status IN ('pending','running','succeeded','dead')),
 attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
 max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 100),
 run_at timestamptz NOT NULL DEFAULT now(),
 lease_token text,
 lease_expires_at timestamptz,
 worker_id text,
 last_error text,
 completed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK ((status='running') = (lease_token IS NOT NULL AND lease_expires_at IS NOT NULL))
);
CREATE UNIQUE INDEX IF NOT EXISTS jobs_idempotency_key_unique ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS jobs_pending_claim_idx ON jobs(run_at,created_at) WHERE status='pending';
CREATE INDEX IF NOT EXISTS jobs_expired_lease_idx ON jobs(lease_expires_at) WHERE status='running';
CREATE INDEX IF NOT EXISTS jobs_status_created_idx ON jobs(status,created_at DESC);
COMMIT;
